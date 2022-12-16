package command

type ShellConfig struct {
	ShellCommand  string `yaml:"shell_command"`
	ShellArgument string `yaml:"shell_argument"`
}
