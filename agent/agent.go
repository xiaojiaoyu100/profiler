package agent

import (
	"bytes"
	"container/ring"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"go.uber.org/zap"

	gprofile "github.com/google/pprof/profile"
	"github.com/sirupsen/logrus"
	"github.com/xiaojiaoyu100/cast"
	"github.com/xiaojiaoyu100/profiler/profile"
)

const (
	agentCircuit = "AgentCircuit"
)

type Agent struct {
	o      *Option
	c      *cast.Cast
	logger *zap.Logger
	stop   chan struct{}
	done   chan struct{}
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

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("fail to create a logger: %w", err)
	}

	agent := &Agent{
		o:      option,
		c:      c,
		logger: logger,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
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
	CreateTime     int64  `json:"createTime"`
}

func (a *Agent) initRing() *ring.Ring {
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
	return r
}

func (a *Agent) collectAndSend(ctx context.Context, buf *bytes.Buffer, r *ring.Ring) error {
	profileType := r.Value.(profile.Type)
	switch profileType {
	case profile.TypeCPU:
		if err := pprof.StartCPUProfile(buf); err != nil {
			return fmt.Errorf("fail to start cpu profile: %w", err)
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
			return fmt.Errorf("fail to look up profile type: %s", profileType.String())
		}
		if err := p.WriteTo(buf, 0); err != nil {
			return fmt.Errorf("fail to write profile: %w", err)
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
	body.GoVersion = a.o.goVersion
	body.ProfileType = profileType.String()

	pf := base64.StdEncoding.EncodeToString(buf.Bytes())
	if len(pf) == 0 {
		return fmt.Errorf("fprofile buffer is zero: %s", profileType.String())
	}

	body.Profile = pf
	body.SendTime = time.Now().UnixNano()
	pp, err := gprofile.ParseData(buf.Bytes())
	if err != nil {
		return fmt.Errorf("fail to parse profile data: %w", err)
	}
	body.CreateTime = pp.TimeNanos

	req := a.c.NewRequest().Post().WithPath("/v1/profile").WithJSONBody(&body)
	resp, err := a.c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("fail to send profile: %w", err)
	}

	if !resp.StatusOk() {
		return fmt.Errorf("response is not ok: %s", resp.String())
	}
	return nil
}

func (a *Agent) prepareNextRound(t *time.Timer, buf *bytes.Buffer, r *ring.Ring) {
	buf.Reset()
	r = r.Next()
	if r.Value.(profile.Type) == profile.TypeCPU {
		t.Reset(adjust(a.o.BreakPeriod))
	}
	t.Reset(adjust(0))
}

func (a *Agent) onSchedule(ctx context.Context) {
	defer close(a.done)

	r := a.initRing()

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
				if err := a.collectAndSend(ctx, &buf, r); err != nil {
					a.logger.Warn(fmt.Sprintf("fail to collect and send: %v", r.Value), zap.Error(err))
				}
				a.prepareNextRound(ti, &buf, r)
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
