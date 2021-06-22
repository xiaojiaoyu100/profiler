package profilemodel

const (
	ProfileId      = "profile_id"
	Service        = "service"
	ServiceVersion = "service_version"
	Host           = "host"
	IP             = "ip"
	GoVersion      = "go_version"
	ProfileType    = "profile_type"
	SendTime       = "send_time"
	CreateTime     = "create_time"
	ObjectName     = "object_name"
	Size           = "size"
)

type Model struct {
	ProfileId      string `json:"profile_id"`
	Service        string `json:"service"`
	ServiceVersion string `json:"service_version"`
	Host           string `json:"host"`
	IP             string `json:"ip"`
	GoVersion      string `json:"go_version"`
	ProfileType    string `json:"profile_type"`
	SendTime       int64  `json:"send_time"`
	CreateTime     int64  `json:"create_time"`
	ObjectName     string `json:"object_name"`
	Size           int64  `json:"size"`
}
