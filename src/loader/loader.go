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
		p := loadProjectFromFile(file, opts.disableDotenv, opts.EnvFileNames)
		opts.projects = append(opts.projects, p)
	}
	mergedProject, err := merge(opts)
	if err != nil {
		return nil, err
	}
	mergedProject.FileNames = opts.FileNames
	mergedProject.IsTuiDisabled = opts.isTuiDisabled || mergedProject.IsTuiDisabled

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
		validatePlatformCompatibility,
		validateHealthDependencyHasHealthCheck,
		validateDependencyIsEnabled,
		validateNoIncompatibleHealthChecks,
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

func loadProjectFromFile(inputFile string, disableDotEnv bool, envFileNames []string) *types.Project {
	yamlFile, err := os.ReadFile(inputFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Error().Msgf("File %s doesn't exist", inputFile)
		}
		log.Fatal().Err(err).Msgf("Failed to read %s", inputFile)
	}

	if !disableDotEnv {
		// .env is optional we don't care if it errors
		_ = godotenv.Load(envFileNames...)
	}

	const envEscaped = "##PC_ENV_ESCAPED##"
	// replace escaped $$ env vars in yaml
	temp := strings.ReplaceAll(string(yamlFile), "$$", envEscaped)
	temp = os.ExpandEnv(temp)
	temp = strings.ReplaceAll(temp, envEscaped, "$")

	project := &types.Project{
		LogLength: defaultLogLength,
	}
	err = yaml.Unmarshal([]byte(temp), project)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse %s", inputFile)
	}
	if project.DisableEnvExpansion {
		err = yaml.Unmarshal(yamlFile, project)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to parse %s", inputFile)
		}
	}

	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to validate %s", inputFile)
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
