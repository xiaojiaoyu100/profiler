package ossconfig

const (
	DataID = "OSS"
)

type Config struct {
	Endpoint        string `json:"ebdpoint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	Bucket          string `json:"bucket"`
	PathPrefix      string `json:"path_prefix"`
}
