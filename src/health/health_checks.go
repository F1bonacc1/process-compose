package health

import (
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/f1bonacc1/go-health/v2"
	"github.com/f1bonacc1/go-health/v2/checkers"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/rs/zerolog/log"
)

const (
	OK = "ok"
)

type Prober struct {
	probe          Probe
	name           string
	onCheckEndFunc func(bool, bool, string, interface{})
	hc             *health.Health
	stopped        atomic.Bool
	env            []string
	shellConfig    command.ShellConfig
}

func New(name string, probe Probe, env []string, shellConfig command.ShellConfig, onCheckEnd func(bool, bool, string, interface{})) (*Prober, error) {
	probe.ValidateAndSetDefaults()
	p := &Prober{
		probe:          probe,
		name:           name,
		onCheckEndFunc: onCheckEnd,
		hc:             health.New(),
		env:            env,
		shellConfig:    shellConfig,
	}
	p.hc.DisableLogging()
	if probe.Exec != nil {
		err := p.addProber(p.getExecChecker)
		if err != nil {
			return nil, err
		}
		return p, err
	}
	if probe.HttpGet != nil {
		err := p.addProber(p.getHttpChecker)
		if err != nil {
			return nil, err
		}
		return p, err
	}
	return nil, fmt.Errorf("no probes [http_get, exec] configured for %s", name)
}

func (p *Prober) Start() {
	go func() {
		p.stopped.Store(false)
		time.Sleep(time.Duration(p.probe.InitialDelay) * time.Second)
		if p.stopped.Load() {
			return
		}
		err := p.hc.Start()
		if err != nil && !errors.Is(err, health.ErrAlreadyRunning) {
			log.Error().Err(err).Msgf("%s failed to start monitoring", p.name)
			return
		}
		log.Debug().Msgf("%s started monitoring", p.name)
	}()
}

func (p *Prober) Stop() {
	if p.hc != nil {
		_ = p.hc.Stop()
		p.stopped.Store(true)
	}
}

func (p *Prober) healthCheckCompleted(state *health.State) {
	fatal := false
	ok := false
	if state.ContiguousFailures == int64(p.probe.FailureThreshold) {
		fatal = true
	}
	if state.Status == OK {
		ok = true
	}
	if p.stopped.Load() {
		return
	}
	p.onCheckEndFunc(ok, fatal, state.Err, state.Details)
}

func (p *Prober) addProber(factory func() (health.ICheckable, error)) error {
	checker, err := factory()
	if err != nil {
		return err
	}
	return p.hc.AddCheck(&health.Config{
		Name:       p.name,
		Checker:    checker,
		Interval:   time.Duration(p.probe.PeriodSeconds) * time.Second,
		Fatal:      false,
		OnComplete: p.healthCheckCompleted,
	})
}

func (p *Prober) getHttpChecker() (health.ICheckable, error) {
	httpGet := p.probe.HttpGet
	url, err := httpGet.getUrl()
	if err != nil {
		return nil, err
	}

	config := &checkers.HTTPConfig{
		URL:        url,
		Timeout:    time.Duration(p.probe.TimeoutSeconds) * time.Second,
		StatusCode: httpGet.StatusCode,
	}

	if len(httpGet.Headers) > 0 {
		config.Headers = http.Header{}
		for k, v := range httpGet.Headers {
			config.Headers.Set(k, v)
		}
	}

	checker, err := checkers.NewHTTP(config)
	if err != nil {
		return nil, err
	}
	return checker, nil
}

func (p *Prober) getExecChecker() (health.ICheckable, error) {
	return &execChecker{
		command:     p.probe.Exec.Command,
		timeout:     p.probe.TimeoutSeconds,
		workingDir:  p.probe.Exec.WorkingDir,
		env:         p.env,
		shellConfig: p.shellConfig,
	}, nil
}
