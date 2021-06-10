package agent

import (
	"bytes"
	"container/ring"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	gprofile "github.com/google/pprof/profile"
	"github.com/sirupsen/logrus"
	"github.com/xiaojiaoyu100/cast"
	"github.com/xiaojiaoyu100/profiler/profile"
)

const (
	agentCircuit = "AgentCircuit"
)

type Agent struct {
	o    *Option
	c    *cast.Cast
	stop chan struct{}
	done chan struct{}
}

type Setter func(o *Option) error

func WithCollectorAddr(addr string) Setter {
	return func(o *Option) error {
		o.CollectorAddr = addr
		return nil
	}
}

func WithService(service string) Setter {
	return func(o *Option) error {
		o.Service = service
		return nil
	}
}

func New(ff ...Setter) (*Agent, error) {
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
	if option.Service == "" {
		return nil, errors.New("no service provided")
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
		done: make(chan struct{}),
	}
	return agent, nil
}

func (a *Agent) Start(ctx context.Context) {
	go a.onSchedule(ctx)
}

func adjust(t time.Duration) time.Duration {
	return t + time.Duration(rand.Intn(5)+1)*time.Second
}

type ReceiveProfileReq struct {
	Service        string `json:"service"`
	ServiceVersion string `json:"serviceVersion"`
	Host           string `json:"host"`
	GoVersion      string `json:"goVersion"`
	ProfileType    string `json:"profileType"`
	Profile        string `json:"profile"`
	SendTime       int64  `json:"sendTime"`
	CreateTime     int64  `json:"create_time"`
}

func (a *Agent) onSchedule(ctx context.Context) {
	defer close(a.done)

	var ll = []profile.Type{
		profile.TypeCPU,
		profile.TypeHeap,
		profile.TypeAllocs,
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
				fmt.Println("timer fires: ", time.Now())
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
					profile.TypeAllocs,
					profile.TypeBlock,
					profile.TypeMutex,
					profile.TypeGoroutine,
					profile.TypeThreadCreate:
					p := pprof.Lookup(profileType.String())
					if p == nil {
						log.Println("fail to look up profile")
						return
					}
					if err := p.WriteTo(&buf, 0); err != nil {
						log.Println("fail to write profile: ", err)
						return
					}
				}

				var body ReceiveProfileReq
				body.Service = a.o.Service
				body.ServiceVersion = a.o.ServiceVersion
				hostname, err := os.Hostname()
				if err != nil {
					hostname = "unknown"
				}
				body.Host = hostname
				body.GoVersion = runtime.Version()
				body.ProfileType = profileType.String()

				pf := base64.StdEncoding.EncodeToString(buf.Bytes())

				body.Profile = pf
				body.SendTime = time.Now().UnixNano()
				pp, err := gprofile.ParseData(buf.Bytes())
				if err != nil {
					fmt.Println("parse data: ", err, profileType)
					continue
				}
				body.CreateTime = pp.TimeNanos

				req := a.c.NewRequest().Post().WithPath("/v1/profile").WithJSONBody(&body)
				resp, err := a.c.Do(ctx, req)
				if err != nil {
					fmt.Printf("send err: %s", err)
					continue
				}

				if !resp.StatusOk() {
					fmt.Println("status not ok")
					continue
				}

				buf.Reset()
				r = r.Next()
				if r.Value.(profile.Type) == profile.TypeCPU {
					ti.Reset(adjust(a.o.BreakPeriod))
				}

				ti.Reset(adjust(0))
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

func (a *Agent) Stop() {
	close(a.stop)
	<-a.done
}
