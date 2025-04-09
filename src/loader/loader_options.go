package loader

import (
	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/f1bonacc1/process-compose/src/types"
	"os"
	"path/filepath"
)

type LoaderOptions struct {
	workingDir        string
	FileNames         []string
	EnvFileNames      []string
	IsInternalLoader  bool
	projects          []*types.Project
	admitters         []admitter.Admitter
	disableDotenv     bool
	isTuiDisabled     bool
	disabledProcesses []string
	enabledProcesses  []string
}

func (o *LoaderOptions) AddAdmitter(adm ...admitter.Admitter) {
	o.admitters = append(o.admitters, adm...)
}

func (o *LoaderOptions) getWorkingDir() (string, error) {
	if o.workingDir != "" {
		return o.workingDir, nil
	}
	for _, path := range o.FileNames {
		if path != "-" {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", err
			}
			return filepath.Dir(absPath), nil
		}
	}
	return os.Getwd()
}

func (o *LoaderOptions) DisableDotenv(disabled bool) {
	o.disableDotenv = disabled
}

func (o *LoaderOptions) WithTuiDisabled(disabled bool) {
	o.isTuiDisabled = disabled
}

func (o *LoaderOptions) DisableProcesses(disabled []string) {
	o.disabledProcesses = disabled
}

func (o *LoaderOptions) EnableProcesses(enabled []string) {
	o.enabledProcesses = enabled
}
