package templater

import (
	"bytes"
	"github.com/f1bonacc1/process-compose/src/types"
	"text/template"
)

type Templater struct {
	Vars *types.Vars
	err  error
}

func (t *Templater) Render(template string) string {
	return t.render(template, t.Vars)
}

func (t *Templater) render(str string, vars *types.Vars) string {
	if str == "" || t.err != nil {
		return ""
	}
	tpl, err := template.New("").Parse(str)
	if err != nil {
		t.err = err
		return ""
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, vars)
	if err != nil {
		t.err = err
		return ""
	}
	return buf.String()
}

func (t *Templater) GetError() error {
	return t.err
}
