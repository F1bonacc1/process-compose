package health

import (
	"fmt"
	"time"

	"github.com/InVisionApp/go-health/v2"
	"github.com/InVisionApp/go-health/v2/checkers"
	"github.com/rs/zerolog/log"
)

const (
	FAIL = "failed"
	OK   = "ok"
)

type Prober struct {
	probe          Probe
	name           string
	onCheckEndFunc func(bool, bool, string)
	hc             *health.Health
}

func New(name string, probe Probe, onCheckEnd func(bool, bool, string)) (*Prober, error) {
	validateAndSetDefaults(&probe)
	p := &Prober{
		probe:          probe,
		name:           name,
		onCheckEndFunc: onCheckEnd,
		hc:             health.New(),
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
	return p, fmt.Errorf("probe settings are missing [http_get.host, exec.command] for %s", name)
}

func (p *Prober) Start() {
	go func() {
		time.Sleep(time.Duration(p.probe.InitialDelay) * time.Second)
		log.Debug().Msgf("%s starting monitoring", p.name)
		err := p.hc.Start()
		if err != nil {
			log.Error().Msgf("%s failed to start monitoring - %s", p.name, err.Error())
		}
	}()
}

func (p *Prober) Stop() {
	if p.hc != nil {
		p.hc.Stop()
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
