//go:build !windows

package cmd

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"syscall"
)

func runInDetachedMode() {
	log.Info().Msg("Running in detached mode")
	fmt.Println("Starting Process Compose in detached mode. Use 'process-compose attach' to connect to it or 'process-compose down' to stop it")
	//remove detached flag
	for i, arg := range os.Args {
		if arg == "-D" || arg == "--detached" {
			os.Args = append(os.Args[:i], os.Args[i+1:]...)
			break
		}
	}
	// Prepare to launch the background process
	os.Args = append(os.Args, "-t=false")
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Detach from terminal
	}

	// Redirect standard file descriptors to /dev/null
	cmd.Stdin = nil
	cmd.Stdout, _ = os.OpenFile("/dev/null", os.O_RDWR, 0)
	cmd.Stderr, _ = os.OpenFile("/dev/null", os.O_RDWR, 0)

	// Start the process in the background
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// Exit the parent process
	os.Exit(0)
}
