package app

type ACMOption struct {
	Addr         string
	Tenant       string
	Group        string
	AccessKey    string
	SecretKey    string
	KmsRegionID  string
	KmsAccessKey string
	KmsSecretKey string
}

type BuildOption struct {
	Service       string // 填写稳定的进程名或者服务名，便于以后查询
	CodeVersion   string // 代码版本
	GoVersion     string // go版本
	BuildDateTime string // 构建时间
	GitCommitHash string // git提交hash
	LastCommitMsg string // git最后一次提交comment
}
