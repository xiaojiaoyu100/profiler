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
	"reflect"
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

func WithService(service string, serviceVersion string) Setter {
	return func(o *Option) error {
		o.Service = service
		o.ServiceVersion = serviceVersion
		return nil
	}
}

func WithBreakPeriod(d time.Duration) Setter {
	return func(o *Option) error {
		o.BreakPeriod = d
		return nil
	}
}

func WithCPUProfiling(en bool, d time.Duration) Setter {
	return func(o *Option) error {
		o.CPUProfiling = en
		o.CPUProfilingPeriod = d
		return nil
	}
}

func WithHeapProfiling(en bool) Setter {
	return func(o *Option) error {
		o.HeapProfiling = en
		return nil
	}
}

func WithAllocsProfiling(en bool) Setter {
	return func(o *Option) error {
		o.AllocsProfiling = en
		return nil
	}
}

func WithBlockProfiling(en bool) Setter {
	return func(o *Option) error {
		o.BlockProfiling = en
		return nil
	}
}

func WithMutexProfiling(en bool) Setter {
	return func(o *Option) error {
		o.MutexProfiling = en
		return nil
	}
}

func WithGoroutineProfiling(en bool) Setter {
	return func(o *Option) error {
		o.GoroutineProfiling = en
		return nil
	}
}

func WithThreadCreateProfiling(en bool) Setter {
	return func(o *Option) error {
		o.ThreadCreateProfiling = en
		return nil
	}
}

func New(ff ...Setter) (*Agent, error) {
	option := &Option{}
	option.goVersion = runtime.Version()
	option.BreakPeriod = defaultBreakPeriod
	option.CPUProfilingPeriod = defaultCPUProfilingPeriod
	option.CPUProfiling = true
	option.HeapProfiling = true
	option.AllocsProfiling = true

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

	if option.ServiceVersion == "" {
		return nil, errors.New("no service version provided")
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
	ServiceVersion string `json:"service_version"`
	Host           string `json:"host"`
	GoVersion      string `json:"go_version"`
	ProfileType    string `json:"profile_type"`
	Profile        string `json:"profile"`
	SendTime       int64  `json:"send_time"`
	CreateTime     int64  `json:"create_time"`
}

func (a *Agent) initRing() *ring.Ring {
	ret := make(map[string]bool)
	t := reflect.TypeOf(*a.o)
	v := reflect.ValueOf(a.o).Elem()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("profile")
		if tag == "" {
			continue
		}
		if v.Field(i).Type().Kind() != reflect.Bool {
			continue
		}
		en := v.Field(i).Bool()
		if !en {
			continue
		}
		ret[tag] = true
	}
	var ll = []profile.Type{
		profile.TypeCPU,
		profile.TypeHeap,
		profile.TypeAllocs,
		profile.TypeBlock,
		profile.TypeMutex,
		profile.TypeGoroutine,
		profile.TypeThreadCreate,
	}
	var r = ring.New(len(ret))
	for i := 0; i < len(ll); i++ {
		_, ok := ret[ll[i].String()]
		if !ok {
			continue
		}
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
			return fmt.Errorf("fail to write profile[%s]: %w", profileType.String(), err)
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
		return fmt.Errorf("profile buffer is zero: %s", profileType.String())
	}

	body.Profile = pf
	body.SendTime = time.Now().UnixNano()
	pp, err := gprofile.ParseData(buf.Bytes())
	if err != nil {
		return fmt.Errorf("fail to parse profile[%s] data: %w", profileType.String(), err)
	}
	body.CreateTime = pp.TimeNanos

	req := a.c.NewRequest().Post().WithPath("/v1/profile").WithJSONBody(&body)
	resp, err := a.c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("fail to send profile[%s]: %w", profileType.String(), err)
	}
	if !resp.StatusOk() {
		return fmt.Errorf("profile[%s] response is not ok: %s", profileType.String(), resp.String())
	}
	return nil
}

func (a *Agent) prepareNextRound(t *time.Timer, buf *bytes.Buffer, r *ring.Ring, pt profile.Type) *ring.Ring {
	buf.Reset()
	r = r.Next()
	if r.Value.(profile.Type) == pt {
		t.Reset(adjust(a.o.BreakPeriod))
	}
	t.Reset(adjust(0))
	return r
}

func (a *Agent) onSchedule(ctx context.Context) {
	defer close(a.done)

	r := a.initRing()
	pt := r.Value.(profile.Type)

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
				r = a.prepareNextRound(ti, &buf, r, pt)
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
