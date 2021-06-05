package build

type Option struct {
	Service       string // 填写稳定的进程名，全小写
	CodeVersion   string // 代码版本
	GoVersion     string // go版本
	BuildDateTime string // 构建时间
	GitCommitHash string // git提交hash
	LastCommitMsg string // git最后一次提交comment
}
