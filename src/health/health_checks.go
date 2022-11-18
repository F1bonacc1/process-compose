package health

import (
	"errors"
	"fmt"
	"time"

	"github.com/InVisionApp/go-health/v2"
	"github.com/InVisionApp/go-health/v2/checkers"
	"github.com/rs/zerolog/log"
)

const (
	OK = "ok"
)

type Prober struct {
	probe          Probe
	name           string
	onCheckEndFunc func(bool, bool, string)
	hc             *health.Health
	stopped        bool
}

func New(name string, probe Probe, onCheckEnd func(bool, bool, string)) (*Prober, error) {
	probe.validateAndSetDefaults()
	p := &Prober{
		probe:          probe,
		name:           name,
		onCheckEndFunc: onCheckEnd,
		hc:             health.New(),
		stopped:        false,
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
		p.stopped = false
		time.Sleep(time.Duration(p.probe.InitialDelay) * time.Second)
		if p.stopped {
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
		p.stopped = true
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
	p.onCheckEndFunc(ok, fatal, state.Err)
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
	url, err := p.probe.HttpGet.getUrl()
	if err != nil {
		return nil, err
	}
	checker, err := checkers.NewHTTP(&checkers.HTTPConfig{
		URL:     url,
		Timeout: time.Duration(p.probe.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return checker, nil
}

func (p *Prober) getExecChecker() (health.ICheckable, error) {
	return &execChecker{
		command: p.probe.Exec.Command,
		timeout: p.probe.TimeoutSeconds,
	}, nil
}
