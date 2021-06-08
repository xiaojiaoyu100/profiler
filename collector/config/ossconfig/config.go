package ossconfig

const (
	DataID = "OSS"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	PathPrefix      string
}
