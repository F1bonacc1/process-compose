//go:build !windows

package cmd

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func runInDetachedMode() {
	log.Info().Msg("Running in detached mode")
	fmt.Println("Starting Process Compose in detached mode. Use 'process-compose attach' to connect to it or 'process-compose down' to stop it")
	//remove detached flag
	for i, arg := range os.Args {
		if arg == "-D" || arg == "--detached" || arg == "--detached-with-tui" {
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

	// Wait for the HTTP server to be ready before returning to the caller.
	if err := waitForServerReady(getClient().IsAlive, 5*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "error: process-compose daemon did not become ready within 5s\nCheck the logs for more information: %s\n", *pcFlags.LogFile)
		os.Exit(1)
	}

	if *pcFlags.IsDetachedWithTui {
		startTui(getClient(), false)
	}
	// Exit the parent process
	os.Exit(0)
}

func waitForServerReady(isAlive func() error, timeout time.Duration) error {
	const pollInterval = 100 * time.Millisecond
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := isAlive(); err == nil {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("server did not become ready within %s", timeout)
}
