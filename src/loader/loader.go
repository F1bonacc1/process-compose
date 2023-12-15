package loader

import (
	"errors"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultLogLength = 1000
)

func Load(opts *LoaderOptions) (*types.Project, error) {
	err := autoDiscoverComposeFile(opts)
	if err != nil {
		return nil, err
	}

	for _, file := range opts.FileNames {
		p := loadProjectFromFile(file)
		opts.projects = append(opts.projects, p)
	}
	mergedProject, err := merge(opts)
	if err != nil {
		return nil, err
	}
	mergedProject.FileNames = opts.FileNames

	apply(mergedProject,
		setDefaultShell,
		assignDefaultProcessValues,
		cloneReplicas,
		copyWorkingDirToProbes,
	)
	err = applyWithErr(mergedProject,
		renderTemplates,
	)
	if err != nil {
		return nil, err
	}
	apply(mergedProject,
		assignExecutableAndArgs,
	)

	err = validate(mergedProject,
		validateLogLevel,
		validateProcessConfig,
		validateNoCircularDependencies,
		validateShellConfig,
	)
	admitProcesses(opts, mergedProject)
	return mergedProject, err
}

func admitProcesses(opts *LoaderOptions, p *types.Project) *types.Project {
	if opts.admitters == nil {
		return p
	}
	for _, process := range p.Processes {
		for _, adm := range opts.admitters {
			if !adm.Admit(&process) {
				log.Info().Msgf("Process %s was removed due to admission policy", process.ReplicaName)
				delete(p.Processes, process.ReplicaName)
			}
		}
	}
	return p
}

func loadProjectFromFile(inputFile string) *types.Project {
	yamlFile, err := os.ReadFile(inputFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Error().Msgf("File %s doesn't exist", inputFile)
		}
		fmt.Printf("Failed to read %s - %s\n", inputFile, err.Error())
		log.Fatal().Msg(err.Error())
	}

	// .env is optional we don't care if it errors
	_ = godotenv.Load()

	yamlFile = []byte(os.ExpandEnv(string(yamlFile)))

	project := &types.Project{
		LogLength: defaultLogLength,
	}
	err = yaml.Unmarshal(yamlFile, project)
	if err != nil {
		fmt.Printf("Failed to parse %s - %s\n", inputFile, err.Error())
		log.Fatal().Msgf("Failed to parse %s - %s", inputFile, err.Error())
	}

	if err != nil {
		fmt.Printf("Failed to validate %s - %s\n", inputFile, err.Error())
		log.Fatal().Msgf("Failed to validate %s - %s", inputFile, err.Error())
	}
	log.Info().Msgf("Loaded project from %s", inputFile)
	return project
}

func findFiles(names []string, pwd string) []string {
	candidates := []string{}
	for _, n := range names {
		f := filepath.Join(pwd, n)
		if _, err := os.Stat(f); err == nil {
			candidates = append(candidates, f)
		}
	}
	return candidates
}

// DefaultFileNames defines the Compose file names for auto-discovery (in order of preference)
var DefaultFileNames = []string{
	"compose.yml",
	"compose.yaml",
	"process-compose.yml",
	"process-compose.yaml",
}

// DefaultOverrideFileNames defines the Compose override file names for auto-discovery (in order of preference)
var DefaultOverrideFileNames = []string{
	"compose.override.yml",
	"compose.override.yaml",
	"process-compose.override.yml",
	"process-compose.override.yaml",
}

func autoDiscoverComposeFile(opts *LoaderOptions) error {
	if len(opts.FileNames) > 0 {
		return nil
	}
	pwd, err := opts.getWorkingDir()
	if err != nil {
		return err
	}
	candidates := findFiles(DefaultFileNames, pwd)
	if len(candidates) > 0 {
		if len(candidates) > 1 {
			log.Warn().Msgf("Found multiple config files with supported names: %s", strings.Join(candidates, ", "))
			log.Warn().Msgf("Using %s", candidates[0])
		}
		opts.FileNames = append(opts.FileNames, candidates[0])

		overrides := findFiles(DefaultOverrideFileNames, pwd)
		if len(overrides) > 0 {
			if len(overrides) > 1 {
				log.Warn().Msgf("Found multiple override files with supported names: %s", strings.Join(overrides, ", "))
				log.Warn().Msgf("Using %s", overrides[0])
			}
			opts.FileNames = append(opts.FileNames, overrides[0])
		}
		return nil
	}
	return fmt.Errorf("no config files found in %s", pwd)
}
