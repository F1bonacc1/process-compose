package loader

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"os"
	"path/filepath"
)

type LoaderOptions struct {
	workingDir string
	FileNames  []string
	projects   []*types.Project
}

func (o LoaderOptions) getWorkingDir() (string, error) {
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
