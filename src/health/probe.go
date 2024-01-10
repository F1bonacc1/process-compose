package health

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Probe struct {
	Exec             *ExecProbe `yaml:"exec,omitempty"`
	HttpGet          *HttpProbe `yaml:"http_get,omitempty"`
	Http             *HttpProbe `yaml:"http,omitempty"`
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
	Host       string `yaml:"host,omitempty"`
	Path       string `yaml:"path,omitempty"`
	Scheme     string `yaml:"scheme,omitempty"`
	Port       int    `yaml:"port,omitempty"`
	Method     string `yaml:"method,omitempty"`
	StatusCode int    `yaml:"status_code,omitempty"`
}

func (h HttpProbe) getUrl() (*url.URL, error) {
	urlStr := ""
	if h.Port != 0 {
		urlStr = fmt.Sprintf("%s://%s:%d%s", h.Scheme, h.Host, h.Port, h.Path)
	}
	if h.Port == 0 {
		urlStr = fmt.Sprintf("%s://%s%s", h.Scheme, h.Host, h.Path)
	}
	return url.Parse(urlStr)
}

func (p *Probe) validateAndSetDefaults() {
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

	p.validateAndSetHttpDefaults()
}

func (p *Probe) validateAndSetHttpDefaults() {
	if p.Http == nil {
		return
	}
	if len(strings.TrimSpace(p.Http.Host)) == 0 {
		p.Http.Host = "127.0.0.1"
	}
	if len(strings.TrimSpace(p.Http.Scheme)) == 0 {
		p.Http.Scheme = "http"
	}
	if len(strings.TrimSpace(p.Http.Path)) == 0 {
		p.Http.Path = "/"
	}
	if p.Http.Port < 1 || p.Http.Port > 65535 {
		// if undefined or wrong value - will be treated as undefined
		p.Http.Port = 0
	}
	if len(strings.TrimSpace(p.Http.Method)) == 0 {
		p.Http.Method = http.MethodGet
	}
	if p.Http.StatusCode == 0 {
		p.Http.StatusCode = http.StatusOK
	}
}
