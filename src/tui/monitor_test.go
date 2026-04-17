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

func TestMonitorSilenceNoRenotifyAfterAcknowledge(t *testing.T) {
	m := newProcessMonitor()
	threshold := 100 * time.Millisecond
	m.initProcess("proc1", types.MonitorForSilence, threshold)

	// Activity happens (100 visible chars written), then silence
	activity := time.Now()
	m.onProcessUnfocused("proc1", activity)
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &activity, MaxLogicalLine: 100},
	}
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if !hasNotif {
		t.Fatal("expected silence notification")
	}

	// User focuses (acknowledges) then unfocuses without typing.
	// MaxLogicalLine stays at 100 — no new visible output.
	m.onProcessFocused("proc1")
	m.onProcessUnfocused("proc1", activity)

	// Should NOT re-trigger — same max logical line, already acknowledged
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ = m.getNotification("proc1")
	if hasNotif {
		t.Error("should not re-notify for already-acknowledged silence")
	}
}

func TestMonitorSilenceFocusBeforeNotification(t *testing.T) {
	m := newProcessMonitor()
	threshold := 100 * time.Millisecond
	m.initProcess("proc1", types.MonitorForSilence, threshold)

	// Activity happens
	activity := time.Now()
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &activity, MaxLogicalLine: 100},
	}
	// Update so lastSeenMaxLine is set
	m.updateNotifications(states)

	// User focuses BEFORE notification fires (within threshold)
	m.onProcessFocused("proc1")
	m.onProcessUnfocused("proc1", activity)

	// Should NOT notify — user already looked at this process, max logical line unchanged
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if hasNotif {
		t.Error("should not notify after user already focused the process")
	}
}

func TestMonitorSilenceFocusBeforeFirstUpdate(t *testing.T) {
	m := newProcessMonitor()
	threshold := 100 * time.Millisecond
	m.initProcess("proc1", types.MonitorForSilence, threshold)

	// User focuses BEFORE updateNotifications has ever run
	// (lastSeenMaxLine is still 0)
	m.onProcessFocused("proc1")
	m.onProcessUnfocused("proc1", time.Now())

	// First updateNotifications sees MaxLogicalLine for the first time
	activity := time.Now()
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &activity, MaxLogicalLine: 100},
	}
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if hasNotif {
		t.Error("should not notify when user focused before first update")
	}
}

func TestMonitorSilenceRenotifyAfterNewActivity(t *testing.T) {
	m := newProcessMonitor()
	threshold := 100 * time.Millisecond
	m.initProcess("proc1", types.MonitorForSilence, threshold)

	// Initial silence notification
	activity := time.Now()
	m.onProcessUnfocused("proc1", activity)
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &activity, MaxLogicalLine: 100},
	}
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if !hasNotif {
		t.Fatal("expected initial silence notification")
	}

	// User acknowledges and unfocuses
	m.onProcessFocused("proc1")
	m.onProcessUnfocused("proc1", activity)

	// New visible output occurs (max logical line increases)
	time.Sleep(10 * time.Millisecond)
	newActivity := time.Now()
	states[0].LastActivityTime = &newActivity
	states[0].MaxLogicalLine = 500
	m.updateNotifications(states)

	// Wait for new silence after the new activity
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ = m.getNotification("proc1")
	if !hasNotif {
		t.Error("should re-notify after new activity goes silent")
	}
}

func TestMonitorSilenceRenotifyAfterSilencePeriod(t *testing.T) {
	m := newProcessMonitor()
	threshold := 100 * time.Millisecond
	m.initProcess("proc1", types.MonitorForSilence, threshold)

	// Initial silence notification
	activity := time.Now()
	m.onProcessUnfocused("proc1", activity)
	states := []types.ProcessState{
		{Name: "proc1", IsRunning: true, LastActivityTime: &activity, MaxLogicalLine: 100},
	}
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ := m.getNotification("proc1")
	if !hasNotif {
		t.Fatal("expected initial silence notification")
	}

	// User acknowledges and unfocuses
	m.onProcessFocused("proc1")
	m.onProcessUnfocused("proc1", activity)

	// Silence continues while acknowledged — multiple ticks pass with
	// no new output. This must NOT cause maxLineAtSilence to catch up
	// to lastSeenMaxLine, which would prevent future resets.
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	m.updateNotifications(states)
	m.updateNotifications(states)
	hasNotif, _ = m.getNotification("proc1")
	if hasNotif {
		t.Fatal("should not re-notify during acknowledged silence")
	}

	// Now new output arrives (user typed something, process responded)
	newActivity := time.Now()
	states[0].LastActivityTime = &newActivity
	states[0].MaxLogicalLine = 500
	m.updateNotifications(states)

	// Wait for new silence after the new activity
	time.Sleep(150 * time.Millisecond)
	m.updateNotifications(states)
	hasNotif, _ = m.getNotification("proc1")
	if !hasNotif {
		t.Error("should re-notify after new activity goes silent")
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
