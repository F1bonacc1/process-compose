package health

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Probe struct {
	Exec                    *ExecProbe `yaml:"exec,omitempty" json:"exec,omitempty"`
	HttpGet                 *HttpProbe `yaml:"http_get,omitempty" json:"httpGet,omitempty"`
	InitialDelay            int        `yaml:"initial_delay_seconds,omitempty" json:"initialDelay,omitempty"`
	PeriodSeconds           int        `yaml:"period_seconds,omitempty" json:"periodSeconds,omitempty"`
	StartupPeriodSeconds    int        `yaml:"startup_period_seconds,omitempty" json:"startupPeriodSeconds,omitempty"`
	UnhealthyPeriodSeconds  int        `yaml:"unhealthy_period_seconds,omitempty" json:"unhealthyPeriodSeconds,omitempty"`
	FailAfterSeconds        int        `yaml:"fail_after_seconds,omitempty" json:"failAfterSeconds,omitempty"`
	StartupFailAfterSeconds int        `yaml:"startup_fail_after_seconds,omitempty" json:"startupFailAfterSeconds,omitempty"`
	TimeoutSeconds          int        `yaml:"timeout_seconds,omitempty" json:"timeoutSeconds,omitempty"`
	SuccessThreshold        int        `yaml:"success_threshold,omitempty" json:"successThreshold,omitempty"`
	FailureThreshold        int        `yaml:"failure_threshold,omitempty" json:"failureThreshold,omitempty"`
}

type ExecProbe struct {
	Command    string `yaml:"command,omitempty" json:"command,omitempty"`
	WorkingDir string `yaml:"working_dir,omitempty" json:"workingDir,omitempty"`
}

type HttpProbe struct {
	Host       string            `yaml:"host,omitempty" json:"host,omitempty"`
	Path       string            `yaml:"path,omitempty" json:"path,omitempty"`
	Scheme     string            `yaml:"scheme,omitempty" json:"scheme,omitempty"`
	Port       string            `yaml:"port,omitempty" json:"port,omitempty"`
	NumPort    int               `yaml:"num_port,omitempty" json:"numPort,omitempty"`
	Headers    map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	StatusCode int               `yaml:"status_code,omitempty" json:"statusCode,omitempty"`
}

func (h *HttpProbe) getUrl() (*url.URL, error) {
	urlStr := ""
	if h.NumPort != 0 {
		urlStr = fmt.Sprintf("%s://%s:%d%s", h.Scheme, h.Host, h.NumPort, h.Path)
	}
	if h.NumPort == 0 {
		urlStr = fmt.Sprintf("%s://%s%s", h.Scheme, h.Host, h.Path)
	}
	return url.Parse(urlStr)
}

// defaultFailAfterPeriods is the default failure budget, in probe periods,
// used when failure_threshold is not explicitly set.
const defaultFailAfterPeriods = 2

func (p *Probe) ValidateAndSetDefaults() {
	explicitThreshold := p.FailureThreshold > 0
	if p.InitialDelay < 0 {
		p.InitialDelay = 0
	}
	if p.PeriodSeconds < 1 {
		p.PeriodSeconds = 10
	}
	// Poll fast until the process first responds. The startup failure budget is
	// derived from the steady-state count, so this does not shorten the
	// wall-clock grace given to a slow-starting process.
	if p.StartupPeriodSeconds < 1 {
		p.StartupPeriodSeconds = 1
	}
	if p.UnhealthyPeriodSeconds < 1 {
		p.UnhealthyPeriodSeconds = p.PeriodSeconds
	}
	if p.TimeoutSeconds < 1 {
		p.TimeoutSeconds = 1
	}
	if p.SuccessThreshold < 1 {
		p.SuccessThreshold = 1
	}
	if p.FailureThreshold < 1 {
		p.FailureThreshold = 3
	}
	// Time budgets replace the failure count as the give-up mechanism. An
	// explicitly set legacy threshold keeps the deadline it implied; otherwise
	// the default is two probe periods. The startup budget defaults to the
	// steady one; a derived budget of 0 (failure_threshold: 1) makes the first
	// failure fatal.
	if p.FailAfterSeconds < 1 {
		if explicitThreshold {
			p.FailAfterSeconds = (p.FailureThreshold - 1) * p.PeriodSeconds
		} else {
			p.FailAfterSeconds = defaultFailAfterPeriods * p.PeriodSeconds
		}
	}
	if p.StartupFailAfterSeconds < 1 {
		p.StartupFailAfterSeconds = p.FailAfterSeconds
	}

	if p.HttpGet != nil {
		p.HttpGet.validateAndSetHttpDefaults()
	}
}

func (p *HttpProbe) validateAndSetHttpDefaults() {
	if len(strings.TrimSpace(p.Host)) == 0 {
		p.Host = "127.0.0.1"
	}
	if len(strings.TrimSpace(p.Scheme)) == 0 {
		p.Scheme = "http"
	}
	if len(strings.TrimSpace(p.Path)) == 0 {
		p.Path = "/"
	}
	if p.Port == "" {
		p.NumPort = 0
	} else {
		p.NumPort, _ = strconv.Atoi(p.Port)
	}
	if p.NumPort < 1 || p.NumPort > 65535 {
		// if undefined or wrong value - will be treated as undefined
		p.NumPort = 0
	}
	if p.StatusCode < 1 || p.StatusCode >= 400 {
		p.StatusCode = http.StatusOK
	}
}
