package templater

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"text/template"

	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
)

type Templater struct {
	vars     types.Vars
	err      error
	errField string
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

	// The command family shares a single opt-out: a process that needs literal
	// braces in its command almost always needs them in the command it is
	// stopped with (and vice versa), and both are shell strings where a stray
	// `{{` is far more likely to be the payload than a variable.
	if !proc.DisableCommandRendering {
		proc.Command = t.renderField("command", proc.Command, proc.Vars)
		proc.ShutDownParams.ShutDownCommand = t.renderField("shutdown.command", proc.ShutDownParams.ShutDownCommand, proc.Vars)
		for i, arg := range proc.Entrypoint {
			proc.Entrypoint[i] = t.renderField(fmt.Sprintf("entrypoint[%d]", i), arg, proc.Vars)
		}
	}
	proc.WorkingDir = t.renderField("working_dir", proc.WorkingDir, proc.Vars)
	proc.LogLocation = t.renderField("log_location", proc.LogLocation, proc.Vars)
	proc.Description = t.renderField("description", proc.Description, proc.Vars)
	proc.ReadyLogLine = t.renderField("ready_log_line", proc.ReadyLogLine, proc.Vars)
	proc.EnvFile = t.renderField("env_file", proc.EnvFile, proc.Vars)

	for i, envVar := range proc.Environment {
		proc.Environment[i] = t.renderField(fmt.Sprintf("environment[%d]", i), envVar, proc.Vars)
	}

	t.renderLoggerConfig(proc.LoggerConfig, proc.Vars)
	t.renderProbe(proc.ReadinessProbe, proc, "readiness_probe")
	t.renderProbe(proc.LivenessProbe, proc, "liveness_probe")
}

func (t *Templater) renderLoggerConfig(logger *types.LoggerConfig, vars types.Vars) {
	if logger == nil || logger.Rotation == nil {
		return
	}
	logger.Rotation.Directory = t.renderField("log_configuration.rotation.directory", logger.Rotation.Directory, vars)
	logger.Rotation.Filename = t.renderField("log_configuration.rotation.filename", logger.Rotation.Filename, vars)
}

func (t *Templater) renderProbe(probe *health.Probe, procConf *types.ProcessConfig, field string) {
	if probe == nil {
		return
	}

	if probe.Exec != nil {
		probe.Exec.Command = t.renderField(field+".exec.command", probe.Exec.Command, procConf.Vars)
		probe.Exec.WorkingDir = t.renderField(field+".exec.working_dir", probe.Exec.WorkingDir, procConf.Vars)
	} else if probe.HttpGet != nil {
		probe.HttpGet.Path = t.renderField(field+".http_get.path", probe.HttpGet.Path, procConf.Vars)
		probe.HttpGet.Host = t.renderField(field+".http_get.host", probe.HttpGet.Host, procConf.Vars)
		probe.HttpGet.Scheme = t.renderField(field+".http_get.scheme", probe.HttpGet.Scheme, procConf.Vars)
		probe.HttpGet.Port = t.renderField(field+".http_get.port", probe.HttpGet.Port, procConf.Vars)
		for name, value := range probe.HttpGet.Headers {
			probe.HttpGet.Headers[name] = t.renderField(field+".http_get.headers."+name, value, procConf.Vars)
		}
	}
	probe.ValidateAndSetDefaults()
}

func (t *Templater) Render(str string) string {
	return t.render(str, nil)
}

func (t *Templater) RenderWithExtraVars(str string, extra types.Vars) string {
	return t.render(str, extra)
}

// renderField renders a single named process field, recording which field
// failed so the loader can point at it. Once a field has failed the remaining
// ones are left untouched rather than blanked - the caller aborts on the error,
// and an unrendered value is more useful than an empty one when it is reported.
func (t *Templater) renderField(field, str string, extra types.Vars) string {
	if t.err != nil {
		return str
	}
	rendered := t.render(str, extra)
	if t.err != nil {
		t.errField = field
		return str
	}
	return rendered
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

// GetErrorField returns the process field whose template failed to render, or
// an empty string if the failure did not originate from a named field.
func (t *Templater) GetErrorField() string {
	return t.errField
}
