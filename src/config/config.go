package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/adrg/xdg"
	"github.com/rs/zerolog/log"
)

var (
	Version           = "undefined"
	Commit            = "undefined"
	Date              = "undefined"
	CheckForUpdates   = "false"
	License           = "Apache-2.0"
	Discord           = "https://discord.gg/S4xgmRSHdC"
	Author            = "Eugene Berger"
	ProjectName       = "Process Compose 🔥"
	RemoteProjectName = "Process Compose ⚡"

	scFiles = []string{
		"shortcuts.yaml",
		"shortcuts.yml",
	}
)

const (
	pcConfigEnv        = "PROC_COMP_CONFIG"
	LogPathEnvVarName  = "PC_LOG_FILE"
	LogLevelEnvVarName = "PC_LOG_LEVEL"
	LogFileFlags       = os.O_CREATE | os.O_APPEND | os.O_WRONLY | os.O_TRUNC
	LogFileMode        = os.FileMode(0600)
	RecipeFileMode     = os.FileMode(0644)
	themeFileName      = "theme.yaml"
	settingsFileName   = "settings.yaml"
	configHome         = "process-compose"
	recipesPath        = "recipes"
)

var (
	clientCommands = []string{
		"down",
		"attach",
		"process",
		"project",
		"list",
		"ls",
		"--detached-with-tui",
	}
)

func GetLogFilePath() string {
	val, found := os.LookupEnv(LogPathEnvVarName)
	if found {
		return val
	}
	userName := getUser()
	if len(userName) != 0 {
		userName = "-" + userName
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("process-compose%s%s.log", userName, mode()))
}

func getDisableTuiDefault() bool {
	val, found := os.LookupEnv(EnvVarNameTui)
	return !found || val == "" || strings.ToLower(val) == "false"
}

func getOrderedShutdownDefault() bool {
	_, found := os.LookupEnv(EnvVarNameOrderedShutdown)
	return found
}

func getNoServerDefault() bool {
	_, found := os.LookupEnv(EnvVarNameNoServer)
	return found
}

func getPortDefault() int {
	val, found := os.LookupEnv(EnvVarNamePort)
	if found {
		port, err := strconv.Atoi(val)
		if err != nil {
			log.Fatal().Err(err).Msgf("Invalid port number: %s", val)
			return DefaultPortNum
		}
		return port
	}
	return DefaultPortNum
}

func GetConfigDefault() []string {
	val, found := os.LookupEnv(EnvVarNameConfig)
	if found {
		return strings.Split(val, ",")
	}
	return []string{}
}

func CreateProcCompHome() string {
	if env := os.Getenv(pcConfigEnv); env != "" {
		return env
	}
	xdgPcHome, err := xdg.ConfigFile(configHome)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create configuration directory")
	}

	err = os.MkdirAll(xdgPcHome, 0700)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create configuration directory for process compose")
	}

	return xdgPcHome
}

func getProcConfigDir() string {
	if env := os.Getenv(pcConfigEnv); env != "" {
		return env
	}
	xdgPcHome, err := xdg.SearchConfigFile(configHome)
	if err != nil {
		log.Debug().Err(err).Msg("Path not found for process compose config home")
	}
	return xdgPcHome
}

func GetShortCutsPaths(extraFiles []string) []string {
	var existingPaths []string = make([]string, 0, len(scFiles)+len(extraFiles))

	// Load shortcuts in order of precedence, lowest to highest: first from the
	// config directory, then from environment variables, then from the CLI
	// argument.
	//
	// This lets shortcuts given on the command line override environment
	// variables and user configuration.
	//
	// See: https://clig.dev/#configuration

	pcHome := getProcConfigDir()
	if pcHome != "" {
		for _, path := range scFiles {
			scPath := filepath.Join(pcHome, path)
			if _, err := os.Stat(scPath); err == nil {
				existingPaths = append(existingPaths, scPath)
			}
		}
	}

	envShortcutsFiles, hasEnvShortcuts := os.LookupEnv(EnvVarNameShortcuts)
	if hasEnvShortcuts {
		existingPaths = append(existingPaths, strings.Split(envShortcutsFiles, ",")...)
	}

	for _, path := range extraFiles {
		if _, err := os.Stat(path); err == nil {
			existingPaths = append(existingPaths, path)
		}
	}

	return existingPaths
}

func GetThemesPath() string {
	pcHome := getProcConfigDir()
	if pcHome == "" {
		return ""
	}
	themePath := filepath.Join(pcHome, themeFileName)
	return themePath
}

func GetSettingsPath() string {
	pcHome := getProcConfigDir()
	if pcHome == "" {
		return ""
	}
	settingsPath := filepath.Join(pcHome, settingsFileName)
	return settingsPath
}

func GetRecipesDir() string {
	pcHome := getProcConfigDir()
	if pcHome == "" {
		return ""
	}
	recipesDir := filepath.Join(pcHome, recipesPath)
	return recipesDir
}

func getUser() string {
	usr, err := user.Current()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to retrieve user info.")
		return ""
	}
	username := usr.Username
	if strings.Contains(username, "\\") {
		parts := strings.Split(username, "\\")
		username = parts[len(parts)-1]
	}
	return username
}

func mode() string {
	if isClient() {
		return "-client"
	}
	return ""
}

func isClient() bool {
	for _, proc := range os.Args {
		for _, cmd := range clientCommands {
			if proc == cmd {
				return true
			}
		}
	}
	return false
}

func IsLogSelectionOn() bool {
	_, found := os.LookupEnv("WAYLAND_DISPLAY")
	return !found
}

func GetUnixSocketPath() string {
	val, found := os.LookupEnv(EnvVarUnixSocketPath)
	if found {
		return val
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("process-compose-%d.sock", os.Getpid()))
}

func getReadOnlyDefault() bool {
	_, found := os.LookupEnv(EnvVarReadOnlyMode)
	return found
}

func getDisableDotEnvDefault() bool {
	_, found := os.LookupEnv(EnvVarDisableDotEnv)
	return found
}

func getTuiFullScreenDefault() bool {
	_, found := os.LookupEnv(EnvVarTuiFullScreen)
	return found
}

func getHideDisabledDefault() bool {
	_, found := os.LookupEnv(EnvVarHideDisabled)
	return found
}

func getWithRecursiveMetricsEnvDefault() bool {
	_, found := os.LookupEnv(EnvVarWithRecursiveMetrics)
	return found
}
