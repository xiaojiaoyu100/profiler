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
	ProfileId      string `ots:"profile_id"`
	Service        string `ots:"service"`
	ServiceVersion string `ots:"service_version"`
	Host           string `ots:"host"`
	IP             string `ots:"ip"`
	GoVersion      string `ots:"go_version"`
	ProfileType    string `ots:"profile_type"`
	SendTime       int64  `ots:"send_time"`
	CreateTime     int64  `ots:"create_time"`
	ObjectName     string `ots:"object_name"`
	Size           int64  `ots:"size"`
}
