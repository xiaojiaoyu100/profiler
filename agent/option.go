package agent

import "time"

type Option struct {
	CollectorAddr         string
	Service               string
	ServiceVersion        string
	goVersion             string
	BreakPeriod           time.Duration
	CPUProfiling          bool `profile:"cpu"`
	CPUProfilingPeriod    time.Duration
	HeapProfiling         bool `profile:"heap"`
	AllocsProfiling       bool `profile:"allocs"`
	BlockProfiling        bool `profile:"block"`
	MutexProfiling        bool `profile:"mutex"`
	GoroutineProfiling    bool `profile:"goroutine"`
	ThreadCreateProfiling bool `profile:"threadcreate"`
}

const (
	defaultBreakPeriod        = time.Second * 30
	defaultCPUProfilingPeriod = time.Second * 10
)
