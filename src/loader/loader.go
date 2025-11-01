package loader

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/drone/envsubst"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const (
	defaultLogLength = 1000
)

func Load(opts *LoaderOptions) (*types.Project, error) {
	err := autoDiscoverComposeFile(opts)
	if err != nil {
		return nil, err
	}

	fileNames := make([]string, len(opts.FileNames))
	_ = copy(fileNames, opts.FileNames)

	for idx, file := range fileNames {
		prj, err := loadProjectFromFile(file, opts)
		if err != nil {
			return nil, err
		}
		err = loadExtendProject(prj, opts, file, idx)
		if err != nil {
			if opts.IsInternalLoader {
				return nil, err
			}
			log.Fatal().Err(err).Send()
		}
		opts.projects = append(opts.projects, prj)
	}
	mergedProject, err := merge(opts)
	if err != nil {
		return nil, err
	}
	mergedProject.FileNames = opts.FileNames
	mergedProject.EnvFileNames = opts.EnvFileNames
	mergedProject.IsTuiDisabled = opts.isTuiDisabled || mergedProject.IsTuiDisabled
	mergedProject.IsOrderedShutdown = opts.isOrderedShutdown || mergedProject.IsOrderedShutdown
	// If DryRun is set to validate the config, then force IsStrict to true:
	mergedProject.IsStrict = opts.DryRun || mergedProject.IsStrict

	// Override log level if given in env
	if envLevel, set := os.LookupEnv(config.LogLevelEnvVarName); set {
		mergedProject.LogLevel = envLevel
	}

	apply(mergedProject,
		setDefaultShell,
		assignDefaultProcessValues,
		cloneReplicas,
		copyWorkingDirToProbes,
		convertStrDisabledToBool,
		disableProcsInEnv,
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

func loadExtendProject(p *types.Project, opts *LoaderOptions, file string, index int) error {
	if p.ExtendsProject != "" {
		if !filepath.IsAbs(p.ExtendsProject) {
			p.ExtendsProject = filepath.Join(filepath.Dir(file), p.ExtendsProject)
		}
		if slices.Contains(opts.FileNames, p.ExtendsProject) {
			log.Error().Msgf("Project %s extends itself", p.ExtendsProject)
			return fmt.Errorf("project %s is already specified in files to load", p.ExtendsProject)
		}
		opts.FileNames = slices.Insert(opts.FileNames, index, p.ExtendsProject)
		project, err := loadProjectFromFile(p.ExtendsProject, opts)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to load the extend project %s", p.ExtendsProject)
			return fmt.Errorf("failed to load extend project %s: %w", p.ExtendsProject, err)
		}
		copyWorkingDirToProcesses(project, filepath.Dir(p.ExtendsProject))
		opts.projects = slices.Insert(opts.projects, index, project)
		err = loadExtendProject(project, opts, p.ExtendsProject, index)
		if err != nil {
			return fmt.Errorf("failed to load extend project %s: %w", p.ExtendsProject, err)
		}
	}
	return nil
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

func loadProjectFromFile(inputFile string, opts *LoaderOptions) (*types.Project, error) {
	yamlFile, err := os.ReadFile(inputFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Error().Msgf("File %s doesn't exist", inputFile)
		}
		if opts.IsInternalLoader {
			return nil, err
		}
		log.Fatal().Err(err).Msgf("Failed to read %s", inputFile)
	}

	dotEnvVars := make(map[string]string)
	if !opts.disableDotenv {
		// .env is optional we don't care if it errors
		dotEnvVars, _ = godotenv.Read(opts.EnvFileNames...)
	}
	expanderFn := func(name string) string {
		val, ok := dotEnvVars[name]
		if ok {
			return val
		}
		return os.Getenv(name)
	}

	const envEscaped = "##PC_ENV_ESCAPED##"
	// replace escaped $$ env vars in yaml
	temp := strings.ReplaceAll(string(yamlFile), "$$", envEscaped)
	temp, err = envsubst.Eval(temp, expanderFn)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse %s, environment substitution errored.", inputFile)
		return nil, err
	}
	temp = strings.ReplaceAll(temp, envEscaped, "$")

	project := &types.Project{
		LogLength: defaultLogLength,
	}
	err = yaml.Unmarshal([]byte(temp), project)
	if err != nil {
		if opts.IsInternalLoader {
			return nil, err
		}
		log.Fatal().Err(err).Msgf("Failed to parse %s", inputFile)
	}
	if project.IsStrict {
		log.Warn().Msg("Strict mode is enabled")
		err = unmarshalStrict([]byte(temp), project)
		if err != nil {
			if opts.IsInternalLoader {
				return nil, err
			}
			log.Fatal().Err(err).Msgf("Failed to parse %s", inputFile)
		}
	}
	if project.DisableEnvExpansion {
		err = yaml.Unmarshal(yamlFile, project)
		if err != nil {
			if opts.IsInternalLoader {
				return nil, err
			}
			log.Fatal().Err(err).Msgf("Failed to parse %s", inputFile)
		}
	}
	project.DotEnvVars = dotEnvVars

	log.Info().Msgf("Loaded project from %s", inputFile)
	return project, nil
}

func unmarshalStrict(data []byte, v interface{}) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	return dec.Decode(v)
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
