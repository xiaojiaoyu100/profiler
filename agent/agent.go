package agent

import (
	"bytes"
	"container/ring"
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xiaojiaoyu100/cast"
	"github.com/xiaojiaoyu100/profiler/profile"
	"log"
	"math/rand"
	"runtime"
	"runtime/pprof"
	"time"
)

const (
	agentCircuit = "AgentCircuit"
)

type Agent struct {
	o    *Option
	c    *cast.Cast
	stop chan struct{}
}

func New(ff ...func(option *Option) error) (*Agent, error) {
	option := &Option{}
	option.goVersion = runtime.Version()
	option.BreakPeriod = defaultBreakPeriod
	option.CPUProfilingPeriod = defaultCPUProfilingPeriod

	for _, f := range ff {
		if err := f(option); err != nil {
			return nil, err
		}
	}
	if option.CollectorAddr == "" {
		return nil, errors.New("no collector addr provided")
	}
	c, err := cast.New(
		cast.WithBaseURL(option.CollectorAddr),
		cast.AddCircuitConfig(agentCircuit),
		cast.WithDefaultCircuit(agentCircuit),
		cast.WithHTTPClientTimeout(time.Second*60),
		cast.WithLogLevel(logrus.WarnLevel),
		cast.WithRetry(2),
		cast.WithExponentialBackoffDecorrelatedJitterStrategy(
			time.Millisecond*200,
			time.Millisecond*500,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create cast err: %w", err)
	}
	agent := &Agent{
		o:    option,
		c:    c,
		stop: make(chan struct{}),
	}
	return agent, nil
}

func (a *Agent) Start(ctx context.Context) error {
	go a.onSchedule(ctx)
	return nil
}

func adjust(t time.Duration) time.Duration {
	return t + time.Duration(rand.Intn(5)+1)*time.Second
}

func (a *Agent) onSchedule(ctx context.Context) {
	var ll = []profile.Type{
		profile.TypeCPU,
		profile.TypeHeap,
		profile.TypeBlock,
		profile.TypeMutex,
		profile.TypeGoroutine,
		profile.TypeThreadCreate,
	}
	var r = ring.New(len(ll))
	for i := 0; i < len(ll); i++ {
		r.Value = ll[i]
		r = r.Next()
	}

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-a.stop
		cancel()
	}()

	ti := time.NewTimer(adjust(0))
	var buf bytes.Buffer
	for {
		select {
		case <-a.stop:
			{
				if !ti.Stop() {
					<-ti.C
					return
				}
			}
		case <-ti.C:
			{
				profileType := r.Value.(profile.Type)
				switch profileType {
				case profile.TypeCPU:
					if err := pprof.StartCPUProfile(&buf); err != nil {
						log.Println("fail to start cpu profile: ", err)
						return
					}
					block(ctx, a.o.CPUProfilingPeriod)
					pprof.StopCPUProfile()
				case profile.TypeHeap,
					profile.TypeBlock,
					profile.TypeMutex,
					profile.TypeGoroutine,
					profile.TypeThreadCreate:
					if err := pprof.Lookup(profileType.String()).WriteTo(&buf, 0); err != nil {
						log.Println("fail to start heap profile: ", err)
						return
					}
				}

				log.Printf("[%s]", profileType.String())
				log.Println(buf.String())

				buf.Reset()
				r = r.Next()
				if r == profile.TypeCPU {
					ti.Reset(adjust(a.o.BreakPeriod))
				}
			}

		}
	}

}

func block(ctx context.Context, t time.Duration) {
	ti := time.NewTimer(t)
	select {
	case <-ti.C:
		{
			return
		}
	case <-ctx.Done():
		{
			if !ti.Stop() {
				<-ti.C
				return
			}
		}
	}
}

func (a *Agent) Stop() error {
	close(a.stop)
	return nil
}
