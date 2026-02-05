package collector

import (
	"MiniSysDashboard/internal/model"
	"math"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Collector struct {
	config        model.Config
	lastNetStats  []net.IOCountersStat
	lastDiskStats map[string]disk.IOCountersStat
	lastTime      time.Time
	mu            sync.Mutex
	firstRun      bool
}

func NewCollector(config model.Config) *Collector {
	return &Collector{
		config:        config,
		lastDiskStats: make(map[string]disk.IOCountersStat),
		lastTime:      time.Now(),
		firstRun:      true,
	}
}

func (c *Collector) Collect() (*model.Metrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	duration := now.Sub(c.lastTime).Seconds()
	if duration <= 0 {
		duration = 1
	}

	metrics := &model.Metrics{
		Timestamp: now.UnixMilli(),
		CreatedAt: now.Format(time.RFC3339),
	}

	// 1. CPU
	if c.config.CPUEnabled {
		// CPU 使用率
		percent, err := cpu.Percent(0, false)
		if err == nil && len(percent) > 0 {
			metrics.CPUUsage = round(percent[0])
		}

		// CPU 负载
		loadStat, err := load.Avg()
		if err == nil {
			metrics.CPULoad = round(loadStat.Load1)
		}

		// CPU 温度
		temps, err := host.SensorsTemperatures()
		if err == nil {
			for _, temp := range temps {
				// 常见的 CPU 温度传感器 Key
				if temp.SensorKey == "coretemp_package_id_0" || temp.SensorKey == "k10temp_tctl" || temp.SensorKey == "cpu_thermal" {
					metrics.CPUTemp = temp.Temperature
					break
				}
			}
		}
	}

	// 2. 内存
	if c.config.MemoryEnabled {
		v, err := mem.VirtualMemory()
		if err == nil {
			metrics.MemoryUsage = round(v.UsedPercent)
			metrics.MemoryTotal = bytesToGB(v.Total)
			metrics.MemoryUsed = bytesToGB(v.Used)
		}
	}

	// 3. 磁盘和网络
	if c.config.DiskEnabled || c.config.NetworkEnabled {
		// 磁盘 I/O
		diskStats, err := disk.IOCounters()
		if err == nil {
			var readBytesDiff, writeBytesDiff float64

			for k, v := range diskStats {
				if last, ok := c.lastDiskStats[k]; ok && !c.firstRun {
					readBytesDiff += float64(v.ReadBytes - last.ReadBytes)
					writeBytesDiff += float64(v.WriteBytes - last.WriteBytes)
				}
				// 更新状态
				c.lastDiskStats[k] = v
			}

			if !c.firstRun {
				// 字节 -> MB/s
				metrics.DiskRead = round((readBytesDiff / (1024 * 1024)) / duration)
				metrics.DiskWrite = round((writeBytesDiff / (1024 * 1024)) / duration)
			}
		}

		// 网络 I/O
		netStats, err := net.IOCounters(false) // false = 聚合所有接口
		if err == nil && len(netStats) > 0 {
			current := netStats[0]

			if !c.firstRun && len(c.lastNetStats) > 0 {
				last := c.lastNetStats[0]
				upDiff := float64(current.BytesSent - last.BytesSent)
				downDiff := float64(current.BytesRecv - last.BytesRecv)

				// 字节 -> Mbps
				metrics.NetworkUp = round(((upDiff * 8) / (1024 * 1024)) / duration)
				metrics.NetworkDown = round(((downDiff * 8) / (1024 * 1024)) / duration)
			}
			c.lastNetStats = netStats
		}

		// 连接数
		conns, err := net.Connections("all")
		if err == nil {
			metrics.NetworkConnections = len(conns)
		}
	}

	c.lastTime = now
	c.firstRun = false

	return metrics, nil
}

func round(val float64) float64 {
	return math.Round(val*100) / 100
}

func bytesToGB(bytes uint64) float64 {
	return round(float64(bytes) / (1024 * 1024 * 1024))
}
