package agent

import "time"

type Option struct {
	CollectorAddr string
	Service string
	ServiceVersion string
	goVersion string
	BreakPeriod time.Duration
	CPUProfiling bool
	CPUProfilingPeriod time.Duration
	HeapProfiling bool
	BlockProfiling bool
	MutexProfiling bool
	GoroutineProfiling bool
	ThreadCreateProfiling bool
}

const (
	defaultBreakPeriod = time.Second * 30
	defaultCPUProfilingPeriod = time.Second * 10
)


