package tui

import (
	"errors"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type dataError struct {
	err error
}

func (de *dataError) Error() string {
	return fmt.Sprintf("# Error unmarshaling YAML %v\n# Please apply the changes again and save the file.\n\n", de.err)
}

func (pv *pcView) editSelectedProcess() {
	editor, err := findTextEditor()
	if err != nil {
		pv.showError(err.Error())
		return
	}

	name := pv.getSelectedProcName()
	info, err := pv.project.GetProcessInfo(name)
	if err != nil {
		pv.showError(err.Error())
		return
	}
	tmpDir := os.TempDir()
	filename := filepath.Join(tmpDir, fmt.Sprintf("pc-%s-config.yaml", name))
	err = writeProcInfoToFile(info, filename)
	if err != nil {
		pv.showError(fmt.Sprintf("Failed to write process to file: %s - %v", filename, err.Error()))
		return
	}
	//remove file after closing the editor
	defer os.Remove(filename)
	var updatedProc *types.ProcessConfig
	for {
		pv.runCommandInForeground(editor, filename)
		updatedProc, err = loadProcInfoFromFile(filename)
		if err != nil {
			var de *dataError
			if errors.As(err, &de) {
				err = writeProcInfoToFile(info, filename, de.Error())
				if err != nil {
					pv.showError(fmt.Sprintf("Failed to write process to file: %s - %v", filename, err.Error()))
					return
				}
				continue
			}
			pv.showError(fmt.Sprintf("Failed to load process from file: %s - %v", filename, err.Error()))
			return
		}
		break
	}
	if updatedProc != nil {
		err = pv.project.UpdateProcess(updatedProc)
		if err != nil {
			pv.showError(fmt.Sprintf("Failed to update process: %s - %v", name, err.Error()))
			return
		}
	}
}

func writeProcInfoToFile(info *types.ProcessConfig, filename string, optionalPrefix ...string) error {
	yamlData, err := yaml.Marshal(info)
	if err != nil {
		log.Err(err).Msgf("Failed to marshal file %s", filename)
		return err
	}
	for _, prefix := range optionalPrefix {
		yamlData = append([]byte(prefix), yamlData...)
	}

	// Write YAML to file
	err = os.WriteFile(filename, yamlData, 0600)
	if err != nil {
		log.Err(err).Msgf("Failed to write file %s", filename)
		return err
	}

	return nil
}

func loadProcInfoFromFile(filename string) (*types.ProcessConfig, error) {
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		log.Err(err).Msgf("Failed to read file %s", filename)
		return nil, err
	}
	info := &types.ProcessConfig{}
	err = yaml.Unmarshal(yamlFile, info)
	if err != nil {
		log.Err(err).Msgf("Failed to unmarshal file %s", filename)
		return nil, &dataError{err: err}
	}
	err = info.ValidateProcessConfig()
	if err != nil {
		return nil, &dataError{err: err}
	}
	return info, nil
}

func findTextEditor() (string, error) {
	// Check if $EDITOR environment variable is set
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor, nil
	}

	// List of popular text editors to check
	var editors []string
	if runtime.GOOS == "windows" {
		editors = []string{"notepad.exe", "notepad++.exe", "code.exe", "gvim.exe"}
	} else {
		editors = []string{"vim", "nano", "vi"}
	}

	for _, editor := range editors {
		path, err := exec.LookPath(editor)
		if err == nil {
			return path, nil
		}
	}

	return "", errors.New("no text editor found")
}
