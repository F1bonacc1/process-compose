package health

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Probe struct {
	Exec             *ExecProbe `yaml:"exec,omitempty"`
	HttpGet          *HttpProbe `yaml:"http_get,omitempty"`
	InitialDelay     int        `yaml:"initial_delay_seconds,omitempty"`
	PeriodSeconds    int        `yaml:"period_seconds,omitempty"`
	TimeoutSeconds   int        `yaml:"timeout_seconds,omitempty"`
	SuccessThreshold int        `yaml:"success_threshold,omitempty"`
	FailureThreshold int        `yaml:"failure_threshold,omitempty"`
}

type ExecProbe struct {
	Command    string `yaml:"command,omitempty"`
	WorkingDir string `yaml:"working_dir,omitempty"`
}

type HttpProbe struct {
	Host    string `yaml:"host,omitempty"`
	Path    string `yaml:"path,omitempty"`
	Scheme  string `yaml:"scheme,omitempty"`
	Port    string `yaml:"port,omitempty"`
	NumPort int    `yaml:"num_port,omitempty"`
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

func (p *Probe) ValidateAndSetDefaults() {
	if p.InitialDelay < 0 {
		p.InitialDelay = 0
	}
	if p.PeriodSeconds < 1 {
		p.PeriodSeconds = 10
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
}
