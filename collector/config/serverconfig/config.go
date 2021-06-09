package serverconfig

const (
	DataID = "Server"
)

type Config struct {
	Service         string `json:"service"`          // 服务名
	Addr            string `json:"addr"`             // http服务器地址
	ShutdownTimeout int    `json:"shutdown_timeout"` // http graceful shutdown的最大等待时间
	LogLevel        string `json:"log_level"`        // 日志打印级别
}
