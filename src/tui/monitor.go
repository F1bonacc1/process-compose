package tui

import (
	"sync"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

const defaultSilenceThreshold = 5 * time.Second

type processMonitorState struct {
	monitorType           types.MonitorFor
	silenceThreshold      time.Duration
	unfocusedSince        time.Time // when process lost focus (zero = currently focused or never unfocused)
	lastActivityAtUnfocus time.Time // snapshot of LastActivityTime when unfocused
	hasNotification       bool
}

type processMonitor struct {
	states map[string]*processMonitorState
	mu     sync.Mutex
}

func newProcessMonitor() *processMonitor {
	return &processMonitor{
		states: make(map[string]*processMonitorState),
	}
}

// initProcess registers a process for monitoring based on its config.
// Processes start in the unfocused state so they are monitored immediately
// without requiring the user to select and deselect them first.
func (m *processMonitor) initProcess(name string, monitorFor types.MonitorFor, silenceThreshold time.Duration) {
	if monitorFor == types.MonitorForNone {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if silenceThreshold <= 0 {
		silenceThreshold = defaultSilenceThreshold
	}
	now := time.Now()
	m.states[name] = &processMonitorState{
		monitorType:      monitorFor,
		silenceThreshold: silenceThreshold,
		unfocusedSince:   now,
	}
}

// onProcessUnfocused is called when a process loses focus in the TUI.
func (m *processMonitor) onProcessUnfocused(name string, lastActivity time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[name]
	if !ok {
		return
	}
	state.unfocusedSince = time.Now()
	state.lastActivityAtUnfocus = lastActivity
	state.hasNotification = false
}

// onProcessFocused is called when a process gains focus in the TUI.
// Returns true if the process had an active notification (was cleared).
func (m *processMonitor) onProcessFocused(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[name]
	if !ok {
		return false
	}
	had := state.hasNotification
	state.hasNotification = false
	state.unfocusedSince = time.Time{}
	return had
}

// updateNotifications checks all monitored processes and updates notification state.
// processStates provides the current state (including LastActivityTime and IsRunning).
func (m *processMonitor) updateNotifications(processStates []types.ProcessState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range processStates {
		ps := &processStates[i]
		ms, ok := m.states[ps.Name]
		if !ok || ms.unfocusedSince.IsZero() {
			continue
		}

		// Clear notification when process is no longer running
		if !ps.IsRunning {
			ms.hasNotification = false
			continue
		}

		if ms.hasNotification {
			continue
		}

		var lastActivity time.Time
		if ps.LastActivityTime != nil {
			lastActivity = *ps.LastActivityTime
		}

		switch ms.monitorType {
		case types.MonitorForActivity:
			// Notify if new output appeared since unfocus
			if lastActivity.After(ms.lastActivityAtUnfocus) {
				ms.hasNotification = true
			}
		case types.MonitorForSilence:
			// Notify if no output for longer than threshold
			if !lastActivity.IsZero() && time.Since(lastActivity) > ms.silenceThreshold {
				ms.hasNotification = true
			}
		}
	}
}

// getNotification returns whether a process has an active notification and its type.
func (m *processMonitor) getNotification(name string) (bool, types.MonitorFor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[name]
	if !ok {
		return false, types.MonitorForNone
	}
	return state.hasNotification, state.monitorType
}

// isMonitored returns true if the process has monitoring configured.
func (m *processMonitor) isMonitored(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.states[name]
	return ok
}
