package command

type ShellConfig struct {
	ShellCommand     string `yaml:"shell_command"`
	ShellArgument    string `yaml:"shell_argument"`
	ElevatedShellCmd string `yaml:"elevated_shell_command"`
	ElevatedShellArg string `yaml:"elevated_shell_argument"`
}
