package influxdbconfig

const (
	DataID = "InfluxDB"
)

type InfluxDBConfig struct {
	ServerURL string
	AuthToken string
}
