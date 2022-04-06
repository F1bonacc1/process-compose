package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	//"log"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

func createProject(inputFile string) *Project {
	yamlFile, err := ioutil.ReadFile(inputFile)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Error().Msgf("File %s doesn't exist", inputFile)
		}
		log.Fatal().Msg(err.Error())
	}

	// .env is optional we don't care if it errors
	godotenv.Load()

	yamlFile = []byte(os.ExpandEnv(string(yamlFile)))

	var project Project
	err = yaml.Unmarshal(yamlFile, &project)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	if project.LogLevel != "" {
		lvl, err := zerolog.ParseLevel(project.LogLevel)
		if err != nil {
			log.Error().Msgf("Unknown log level %s defaulting to %s",
				project.LogLevel, zerolog.GlobalLevel().String())
		} else {
			zerolog.SetGlobalLevel(lvl)
		}

	}
	return &project
}

func setupLogger() {

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "06-01-02 15:04:05",
	})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
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
var DefaultFileNames = []string{"compose.yml", "compose.yaml", "process-compose.yml", "process-compose.yaml"}

func autoDiscoverComposeFile(pwd string) (string, error) {
	candidates := findFiles(DefaultFileNames, pwd)
	if len(candidates) > 0 {
		winner := candidates[0]
		if len(candidates) > 1 {
			log.Warn().Msgf("Found multiple config files with supported names: %s", strings.Join(candidates, ", "))
			log.Warn().Msgf("Using %s", winner)
		}
		return winner, nil
	}
	return "", fmt.Errorf("no config files found in %s", pwd)
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	setupLogger()
	fileName := ""
	flag.StringVar(&fileName, "f", DefaultFileNames[0], "path to file to load")
	flag.Parse()
	if !isFlagPassed("f") {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		file, err := autoDiscoverComposeFile(pwd)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		fileName = file
	}
	project := createProject(fileName)
	project.Run()
}
