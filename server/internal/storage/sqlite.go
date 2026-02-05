package storage

import (
	"MiniSysDashboard/internal/model"
	"database/sql"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	BatchSize     = 2000
	FlushInterval = 5 * time.Minute
	RetentionDays = 7
)

type Storage struct {
	db     *sql.DB
	buffer []*model.Metrics
	mu     sync.Mutex
	done   chan struct{}
}

func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// 初始化数据库表结构和优化配置
	if err := initDB(db); err != nil {
		db.Close()
		return nil, err
	}

	s := &Storage{
		db:     db,
		buffer: make([]*model.Metrics, 0, BatchSize),
		done:   make(chan struct{}),
	}

	// 启动后台刷盘协程
	go s.flusher()

	return s, nil
}

func initDB(db *sql.DB) error {
	// 启用 WAL 模式
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA cache_size=-20000;", // 20MB
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}

	// 创建表
	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp BIGINT NOT NULL UNIQUE,
		cpu_usage REAL NOT NULL,
		cpu_load REAL NOT NULL,
		cpu_temp REAL,
		memory_usage REAL NOT NULL,
		memory_total REAL NOT NULL,
		memory_used REAL NOT NULL,
		disk_read REAL NOT NULL,
		disk_write REAL NOT NULL,
		network_connections INTEGER NOT NULL,
		network_up REAL NOT NULL,
		network_down REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	return nil
}

func (s *Storage) Close() error {
	close(s.done)
	// 关闭前强制刷盘
	s.Flush()
	return s.db.Close()
}

func (s *Storage) Save(m *model.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer = append(s.buffer, m)

	// 我们主要依靠基于时间的刷盘，但如果缓冲区满了，必须刷盘。
	// 为了避免死锁，我们检查大小，如果满了，交换缓冲区并清空，然后异步或同步刷盘。
	if len(s.buffer) >= BatchSize {
		// 交换缓冲区
		toFlush := s.buffer
		s.buffer = make([]*model.Metrics, 0, BatchSize)

		// 在刷盘前解锁，允许新的写入
		s.mu.Unlock()
		err := s.flushBatch(toFlush)
		s.mu.Lock() // 重新获取锁以便 defer 解锁
		return err
	}

	return nil
}

func (s *Storage) flusher() {
	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.Flush()
		case <-s.done:
			return
		}
	}
}

func (s *Storage) Flush() error {
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return nil
	}

	toFlush := s.buffer
	s.buffer = make([]*model.Metrics, 0, BatchSize)
	s.mu.Unlock()

	return s.flushBatch(toFlush)
}

func (s *Storage) flushBatch(metrics []*model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	// SQLite 支持批量插入：INSERT INTO table (...) VALUES (...), (...), ...
	// 但是，绑定变量的数量有限制（SQLITE_MAX_VARIABLE_NUMBER，默认为 999 或 32766）。
	// 每行 13 列，如果限制是 999，我们每个语句只能插入约 76 行；如果是 32766，则可以插入 2500+ 行。
	// 为了安全和高效，我们可以分批处理。
	// 但在事务中使用预处理语句在 SQLite 中已经非常快了。
	// 当前的“for 循环 + 预处理语句”方式是标准的且经过驱动/引擎高度优化的，因为查询计划只编译一次。
	//
	// 真正的多值插入 (VALUES (...), (...)) 稍微减少了系统调用/解析开销，
	// 但需要动态生成 SQL 或仔细构建参数。
	//
	// 鉴于我们的 BatchSize 是 2000，并且我们在事务中，预处理语句循环实际上非常快（数千/秒）。
	// 如果我们真的想优化，我们可以使用单个多值插入，但这会使代码复杂化，而收益微乎其微。
	//
	// 让我们保持使用预处理语句循环，以获得可读性和可靠性，因为它已经包裹在事务中了。
	// 事务是 SQLite 最大的优化因素。

	stmt, err := tx.Prepare(`
		INSERT INTO metrics (
			timestamp, cpu_usage, cpu_load, cpu_temp, 
			memory_usage, memory_total, memory_used, 
			disk_read, disk_write, 
			network_connections, network_up, network_down, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Printf("Failed to prepare statement: %v", err)
		return err
	}
	defer stmt.Close()

	for _, m := range metrics {
		_, err = stmt.Exec(
			m.Timestamp, m.CPUUsage, m.CPULoad, m.CPUTemp,
			m.MemoryUsage, m.MemoryTotal, m.MemoryUsed,
			m.DiskRead, m.DiskWrite,
			m.NetworkConnections, m.NetworkUp, m.NetworkDown, m.CreatedAt,
		)
		if err != nil {
			log.Printf("Failed to exec statement: %v", err)
			continue // 跳过坏记录？还是整批失败？对于监控来说，跳过更安全。
		}
	}

	// 惰性清理：1/10 的机会（或简单地每次刷盘）运行清理
	// 既然刷盘是每 5 分钟一次，对于小数据库来说，每次都运行清理是可以接受的。
	s.cleanup(tx)

	return tx.Commit()
}

func (s *Storage) cleanup(tx *sql.Tx) {
	// 删除超过保留天数的数据
	cutoff := time.Now().AddDate(0, 0, -RetentionDays).UnixMilli()

	// 使用 limit 来避免在有大量数据要删除时产生巨大的锁，
	// 虽然对于 7 天的保留期来说应该是可控的。
	// SQLite 通常不支持 DELETE ... LIMIT，除非有特殊的编译选项。
	// 但带索引 WHERE 的标准 DELETE 是很快的。
	_, err := tx.Exec("DELETE FROM metrics WHERE timestamp < ?", cutoff)
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
	}
}

// QueryRange 返回时间范围内的指标
func (s *Storage) QueryRange(start, end int64) ([]*model.Metrics, error) {
	// 首先查询数据库
	rows, err := s.db.Query(`
        SELECT id, timestamp, cpu_usage, cpu_load, cpu_temp, 
               memory_usage, memory_total, memory_used, 
               disk_read, disk_write, 
               network_connections, network_up, network_down, created_at
        FROM metrics 
        WHERE timestamp BETWEEN ? AND ? 
        ORDER BY timestamp ASC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.Metrics
	for rows.Next() {
		m := &model.Metrics{}
		err := rows.Scan(
			&m.ID, &m.Timestamp, &m.CPUUsage, &m.CPULoad, &m.CPUTemp,
			&m.MemoryUsage, &m.MemoryTotal, &m.MemoryUsed,
			&m.DiskRead, &m.DiskWrite,
			&m.NetworkConnections, &m.NetworkUp, &m.NetworkDown, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}

	// 然后检查缓冲区中的最近数据
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range s.buffer {
		if m.Timestamp >= start && m.Timestamp <= end {
			results = append(results, m)
		}
	}

	return results, nil
}
