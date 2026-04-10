package tui

import (
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

func TestMonitorInitProcess(t *testing.T) {
	m := newProcessMonitor()

	// MonitorForNone should not register
	m.initProcess("proc-none", types.MonitorForNone, 0)
	if m.isMonitored("proc-none") {
		t.Error("MonitorForNone should not be registered")
	}

	// MonitorForActivity should register
	m.initProcess("proc-activity", types.MonitorForActivity, 0)
	if !m.isMonitored("proc-activity") {
		t.Error("MonitorForActivity should be registered")
	}

	// MonitorForSilence should register with default threshold
	m.initProcess("proc-silence", types.MonitorForSilence, 0)
	if !m.isMonitored("proc-silence") {
		t.Error("MonitorForSilence should be registered")
	}
}

func TestMonitorStartsUnfocused(t *testing.T) {
	m := newProcessMonitor()
	m.initProcess("proc1", types.MonitorForActivity, 0)

	// Process starts unfocused — new activity should trigger notification
	// without needing to select and deselect first
	now := time.Now()
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &now},
	}
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if !hasNotif {
		t.Error("process should be monitored immediately without needing focus/unfocus cycle")
	}
}

func TestMonitorFocusUnfocusClearsNotification(t *testing.T) {
	m := newProcessMonitor()
	m.initProcess("proc1", types.MonitorForActivity, 0)

	// Unfocus with a last activity time
	past := time.Now().Add(-10 * time.Second)
	m.onProcessUnfocused("proc1", past)

	// Simulate new activity
	now := time.Now()
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &now},
	}
	m.updateNotifications(states)

	// Should have notification
	hasNotif, monType := m.getNotification("proc1")
	if !hasNotif {
		t.Error("expected notification after activity")
	}
	if monType != types.MonitorForActivity {
		t.Errorf("expected MonitorForActivity, got %v", monType)
	}

	// Focus should clear it
	had := m.onProcessFocused("proc1")
	if !had {
		t.Error("onProcessFocused should return true when clearing notification")
	}
	hasNotif, _ = m.getNotification("proc1")
	if hasNotif {
		t.Error("notification should be cleared after focus")
	}
}

func TestMonitorActivityDetection(t *testing.T) {
	m := newProcessMonitor()
	m.initProcess("proc1", types.MonitorForActivity, 0)

	past := time.Now().Add(-10 * time.Second)
	m.onProcessUnfocused("proc1", past)

	// No new activity yet (same time as unfocus)
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &past},
	}
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if hasNotif {
		t.Error("should not have notification when no new activity")
	}

	// New activity arrives
	now := time.Now()
	states[0].LastActivityTime = &now
	m.updateNotifications(states)
	hasNotif, _ = m.getNotification("proc1")
	if !hasNotif {
		t.Error("should have notification after new activity")
	}
}

func TestMonitorSilenceDetection(t *testing.T) {
	m := newProcessMonitor()
	threshold := 100 * time.Millisecond
	m.initProcess("proc1", types.MonitorForSilence, threshold)

	now := time.Now()
	m.onProcessUnfocused("proc1", now)

	// Recent activity — no notification
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &now},
	}
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if hasNotif {
		t.Error("should not have notification when activity is recent")
	}

	// Wait for silence threshold
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, monType := m.getNotification("proc1")
	if !hasNotif {
		t.Error("should have notification after silence threshold")
	}
	if monType != types.MonitorForSilence {
		t.Errorf("expected MonitorForSilence, got %v", monType)
	}
}

func TestMonitorSilenceNotTriggeredWhenStopped(t *testing.T) {
	m := newProcessMonitor()
	threshold := 50 * time.Millisecond
	m.initProcess("proc1", types.MonitorForSilence, threshold)

	past := time.Now().Add(-1 * time.Second)
	m.onProcessUnfocused("proc1", past)

	// Process not running — should not trigger
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: false, LastActivityTime: &past},
	}
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if hasNotif {
		t.Error("should not trigger silence notification for stopped process")
	}
}

func TestMonitorNotificationClearsOnCompletion(t *testing.T) {
	m := newProcessMonitor()
	m.initProcess("proc1", types.MonitorForActivity, 0)

	past := time.Now().Add(-10 * time.Second)
	m.onProcessUnfocused("proc1", past)

	// Trigger notification
	now := time.Now()
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &now},
	}
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if !hasNotif {
		t.Fatal("expected notification after activity")
	}

	// Process completes — notification should clear
	states[0].IsRunning = false
	m.updateNotifications(states)
	hasNotif, _ = m.getNotification("proc1")
	if hasNotif {
		t.Error("notification should be cleared when process completes")
	}
}

func TestMonitorUnregisteredProcess(t *testing.T) {
	m := newProcessMonitor()

	// Operations on unregistered process should be no-ops
	m.onProcessUnfocused("unknown", time.Now())
	m.onProcessFocused("unknown")
	hasNotif, monType := m.getNotification("unknown")
	if hasNotif {
		t.Error("unregistered process should not have notification")
	}
	if monType != types.MonitorForNone {
		t.Errorf("unregistered process should return MonitorForNone, got %v", monType)
	}
}

func TestMonitorNotificationSticksUntilFocused(t *testing.T) {
	m := newProcessMonitor()
	m.initProcess("proc1", types.MonitorForActivity, 0)

	past := time.Now().Add(-10 * time.Second)
	m.onProcessUnfocused("proc1", past)

	now := time.Now()
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &now},
	}

	// Multiple updates should not clear the notification
	m.updateNotifications(states)
	m.updateNotifications(states)
	m.updateNotifications(states)

	hasNotif, _ := m.getNotification("proc1")
	if !hasNotif {
		t.Error("notification should persist across multiple updates")
	}
}
