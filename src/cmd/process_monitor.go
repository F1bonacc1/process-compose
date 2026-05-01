package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	monitorOutputText = "text"
	monitorOutputJSON = "json"
)

var (
	monitorOutput     string
	monitorNoSnapshot bool
)

// processMonitorCmd subscribes to process state changes and prints them.
var processMonitorCmd = &cobra.Command{
	Use:   "monitor [process-name...]",
	Short: "Subscribe to process state changes and print them as they happen",
	Long: `Subscribe to the server's process state stream and print one line per state event.

With no arguments, all processes are monitored. With one or more process names,
output is filtered to those processes only.

Output formats:
  text  Human-readable columns: timestamp, process, kind, status (default).
  json  One ProcessStateEvent JSON object per line, ready to pipe into jq.

By default, an initial snapshot of every process is emitted on connect; pass
--no-snapshot to suppress it.`,
	Run: runProcessMonitor,
}

func runProcessMonitor(_ *cobra.Command, args []string) {
	switch monitorOutput {
	case monitorOutputText, monitorOutputJSON:
	default:
		log.Fatal().Msgf("invalid --output: %q (expected text or json)", monitorOutput)
	}

	pc := getClient()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Pre-check unknown names so a typo surfaces immediately on stderr.
	// The server will accept and silently filter them too, but a CLI user
	// would otherwise see no output and have no clue why.
	warnUnknownProcesses(pc, args)

	// Server-side filter: pass the requested names so the server only sends
	// matching events (and snapshot frames). Empty args = subscribe to all.
	events, err := pc.SubscribeProcessStates(ctx, args...)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to subscribe to process state stream")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	printer := newMonitorPrinter(monitorOutput)

	for {
		select {
		case <-sigCh:
			return
		case ev, ok := <-events:
			if !ok {
				log.Warn().Msg("State stream closed by server")
				return
			}
			if monitorNoSnapshot && ev.Snapshot {
				continue
			}
			printer.print(ev)
		}
	}
}

// monitorPrinter formats and prints ProcessStateEvent values. It tracks the
// last-seen state per process so that text output can label whether the
// changed field is the Status, Health, or final exit info.
//
// The server applies the name filter (?name=...), so this type does no
// filtering of its own.
type monitorPrinter struct {
	mu      sync.Mutex
	format  string
	last    map[string]types.ProcessState
	encoder *json.Encoder
}

func newMonitorPrinter(format string) *monitorPrinter {
	return &monitorPrinter{
		format:  format,
		last:    map[string]types.ProcessState{},
		encoder: json.NewEncoder(os.Stdout),
	}
}

func (p *monitorPrinter) print(ev types.ProcessStateEvent) {
	p.mu.Lock()
	prev, hadPrev := p.last[ev.State.Name]
	p.last[ev.State.Name] = ev.State
	p.mu.Unlock()

	switch p.format {
	case monitorOutputJSON:
		if err := p.encoder.Encode(ev); err != nil {
			log.Err(err).Msg("Failed to encode state event")
		}
	default:
		p.printText(ev, prev, hadPrev)
	}
}

func (p *monitorPrinter) printText(ev types.ProcessStateEvent, prev types.ProcessState, hadPrev bool) {
	ts := monitorTimestamp(ev)

	switch {
	case ev.Snapshot:
		fmt.Printf("%s  %-20s  snapshot  %s\n", ts, ev.State.Name, ev.State.Status)
		if ev.State.HasHealthProbe && ev.State.Health != types.ProcessHealthUnknown {
			fmt.Printf("%s  %-20s  health    %s\n", ts, ev.State.Name, ev.State.Health)
		}
	case !hadPrev || prev.Status != ev.State.Status:
		extras := ""
		if isTerminalStatus(ev.State.Status) {
			extras = fmt.Sprintf("   exit=%d restarts=%d", ev.State.ExitCode, ev.State.Restarts)
		}
		fmt.Printf("%s  %-20s  status    %s%s\n", ts, ev.State.Name, ev.State.Status, extras)
	case prev.Health != ev.State.Health:
		fmt.Printf("%s  %-20s  health    %s\n", ts, ev.State.Name, ev.State.Health)
	case prev.ExitCode != ev.State.ExitCode || prev.Restarts != ev.State.Restarts:
		fmt.Printf("%s  %-20s  exit      code=%d restarts=%d\n",
			ts, ev.State.Name, ev.State.ExitCode, ev.State.Restarts)
	}
}

func monitorTimestamp(ev types.ProcessStateEvent) string {
	// Prefer end-time → ready-time → start-time → empty so the timestamp is
	// always meaningful for the kind of event being reported. The events
	// themselves don't carry an explicit publish timestamp; using process
	// lifecycle timestamps avoids duplicating data the server already exposes.
	if ev.State.ProcessEndTime != nil {
		return ev.State.ProcessEndTime.Format("2006-01-02T15:04:05Z07:00")
	}
	if ev.State.ProcessReadyTime != nil {
		return ev.State.ProcessReadyTime.Format("2006-01-02T15:04:05Z07:00")
	}
	if ev.State.ProcessStartTime != nil {
		return ev.State.ProcessStartTime.Format("2006-01-02T15:04:05Z07:00")
	}
	return "                    "
}

func isTerminalStatus(status string) bool {
	switch status {
	case types.ProcessStateCompleted, types.ProcessStateError, types.ProcessStateSkipped:
		return true
	}
	return false
}

// warnUnknownProcesses fetches the server's current process list and warns
// (on stderr — only Fatal-level zerolog reaches stderr in this binary) for
// any requested name that isn't present. Unknown names are still passed to
// the server (which silently accepts them); this helper exists so the user
// sees their typo instead of an empty terminal.
//
// A failure to reach the server is itself logged but non-fatal — if the API
// is down, SubscribeProcessStates will fail next with a more actionable
// message anyway.
func warnUnknownProcesses(pc *client.PcClient, requested []string) {
	if len(requested) == 0 {
		return
	}
	known, err := pc.GetProcessesName()
	if err != nil {
		log.Debug().Err(err).Msg("Could not list processes for name pre-check; skipping")
		return
	}
	knownSet := make(map[string]struct{}, len(known))
	for _, n := range known {
		knownSet[n] = struct{}{}
	}
	for _, name := range requested {
		if _, ok := knownSet[name]; !ok {
			fmt.Fprintf(os.Stderr,
				"warning: unknown process %q; subscribing anyway, but no events will be delivered for it\n",
				name)
			log.Warn().Str("name", name).Msg("Unknown process name passed to monitor")
		}
	}
}

func init() {
	processCmd.AddCommand(processMonitorCmd)
	processMonitorCmd.Flags().StringVarP(&monitorOutput, "output", "o", monitorOutputText,
		"Output format: text or json")
	processMonitorCmd.Flags().BoolVar(&monitorNoSnapshot, "no-snapshot", false,
		"Skip the initial state snapshot, only print live changes")
}
