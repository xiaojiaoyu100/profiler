package tablestoreconfig

const (
	DataID = "Tablestore"
)

type TablestoreConfig struct {
	EndPoint        string `json:"endpoint"`
	InstanceName    string `json:"instance_name"`
	AccessKeyId     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	TableName       string `json:"table_name"`
}
