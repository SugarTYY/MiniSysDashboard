package model

// Metrics 代表在特定时间点采集的系统指标。
type Metrics struct {
	ID                 int64   `json:"id" db:"id"`
	Timestamp          int64   `json:"timestamp" db:"timestamp"` // Unix 时间戳（毫秒）
	CPUUsage           float64 `json:"cpu_usage" db:"cpu_usage"`
	CPULoad            float64 `json:"cpu_load" db:"cpu_load"`
	CPUTemp            float64 `json:"cpu_temp" db:"cpu_temp"`
	MemoryUsage        float64 `json:"memory_usage" db:"memory_usage"`
	MemoryTotal        float64 `json:"memory_total" db:"memory_total"` // 单位：GB
	MemoryUsed         float64 `json:"memory_used" db:"memory_used"`   // 单位：GB
	DiskRead           float64 `json:"disk_read" db:"disk_read"`       // 单位：MB/s
	DiskWrite          float64 `json:"disk_write" db:"disk_write"`     // 单位：MB/s
	NetworkConnections int     `json:"network_connections" db:"network_connections"`
	NetworkUp          float64 `json:"network_up" db:"network_up"`     // 单位：Mbps
	NetworkDown        float64 `json:"network_down" db:"network_down"` // 单位：Mbps
	CreatedAt          string  `json:"created_at" db:"created_at"`
}

// Config 代表采集器配置。
type Config struct {
	CPUEnabled         bool `json:"cpu_enabled"`
	MemoryEnabled      bool `json:"memory_enabled"`
	DiskEnabled        bool `json:"disk_enabled"`
	NetworkEnabled     bool `json:"network_enabled"`
	CollectionInterval int  `json:"collection_interval"` // 单位：秒
	RetentionDays      int  `json:"retention_days"`
}

var DefaultConfig = Config{
	CPUEnabled:         true,
	MemoryEnabled:      true,
	DiskEnabled:        true,
	NetworkEnabled:     true,
	CollectionInterval: 1,
	RetentionDays:      7,
}
