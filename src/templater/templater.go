package templater

import (
	"bytes"
	"encoding/json"
	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"maps"
	"text/template"
)

type Templater struct {
	vars types.Vars
	err  error
}

func New(vars types.Vars) *Templater {
	return &Templater{vars: vars}
}

func (t *Templater) RenderProcess(proc *types.ProcessConfig) {
	if proc.Vars == nil {
		proc.Vars = make(types.Vars)
	}
	procConf, err := json.Marshal(proc)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal process config")
	}
	proc.OriginalConfig = string(procConf)
	proc.Vars["PC_REPLICA_NUM"] = proc.ReplicaNum
	proc.Command = t.RenderWithExtraVars(proc.Command, proc.Vars)
	proc.WorkingDir = t.RenderWithExtraVars(proc.WorkingDir, proc.Vars)
	proc.LogLocation = t.RenderWithExtraVars(proc.LogLocation, proc.Vars)
	proc.Description = t.RenderWithExtraVars(proc.Description, proc.Vars)
	t.renderProbe(proc.ReadinessProbe, proc)
	t.renderProbe(proc.LivenessProbe, proc)
}

func (t *Templater) renderProbe(probe *health.Probe, procConf *types.ProcessConfig) {
	if probe == nil {
		return
	}

	if probe.Exec != nil {
		probe.Exec.Command = t.RenderWithExtraVars(probe.Exec.Command, procConf.Vars)
	} else if probe.HttpGet != nil {
		probe.HttpGet.Path = t.RenderWithExtraVars(probe.HttpGet.Path, procConf.Vars)
		probe.HttpGet.Host = t.RenderWithExtraVars(probe.HttpGet.Host, procConf.Vars)
		probe.HttpGet.Scheme = t.RenderWithExtraVars(probe.HttpGet.Scheme, procConf.Vars)
		probe.HttpGet.Port = t.RenderWithExtraVars(probe.HttpGet.Port, procConf.Vars)
	}
	probe.ValidateAndSetDefaults()
}

func (t *Templater) Render(str string) string {
	return t.render(str, nil)
}

func (t *Templater) RenderWithExtraVars(str string, extra types.Vars) string {
	return t.render(str, extra)
}

func (t *Templater) render(str string, extra types.Vars) string {
	if str == "" || t.err != nil {
		return ""
	}
	if len(t.vars) == 0 && len(extra) == 0 {
		return str
	}

	tpl, err := template.New("").Parse(str)
	if err != nil {
		t.err = err
		return ""
	}
	var buf bytes.Buffer
	if extra == nil {
		err = tpl.Execute(&buf, t.vars)
	} else if len(t.vars) == 0 {
		err = tpl.Execute(&buf, extra)
	} else {
		m := maps.Clone(t.vars)
		maps.Copy(m, extra)
		err = tpl.Execute(&buf, m)
	}
	if err != nil {
		t.err = err
		return ""
	}
	return buf.String()
}

func (t *Templater) GetError() error {
	return t.err
}
