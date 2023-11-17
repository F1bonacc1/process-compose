package templater

import (
	"bytes"
	"github.com/f1bonacc1/process-compose/src/types"
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
	tpl, err := template.New("").Parse(str)
	if err != nil {
		t.err = err
		return ""
	}
	var buf bytes.Buffer
	if extra == nil {
		err = tpl.Execute(&buf, t.vars)
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
