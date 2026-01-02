package scheduler

import (
	"sync"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
)

// mockProcessStarter is a mock implementation of ProcessStarter for testing.
type mockProcessStarter struct {
	startedProcesses []string
	startError       error
	mutex            sync.Mutex
}

func (m *mockProcessStarter) StartProcess(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.startError != nil {
		return m.startError
	}
	m.startedProcesses = append(m.startedProcesses, name)
	return nil
}

func (m *mockProcessStarter) GetProcessState(name string) (*types.ProcessState, error) {
	return &types.ProcessState{Name: name}, nil
}

func (m *mockProcessStarter) getStartedProcesses() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	result := make([]string, len(m.startedProcesses))
	copy(result, m.startedProcesses)
	return result
}

func TestScheduler_New(t *testing.T) {
	mock := &mockProcessStarter{}
	s, err := New(mock)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if s == nil {
		t.Fatal("New() returned nil scheduler")
	}
	if s.starter != mock {
		t.Error("New() did not set starter correctly")
	}
	if s.schedules == nil {
		t.Error("New() did not initialize schedules map")
	}
}

func TestScheduler_AddProcess_NilConfig(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	err := s.AddProcess("test", nil)
	if err != nil {
		t.Errorf("AddProcess() with nil config should not return error, got %v", err)
	}
}

func TestScheduler_AddProcess_EmptyConfig(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{}
	err := s.AddProcess("test", config)
	if err != nil {
		t.Errorf("AddProcess() with empty config should not return error, got %v", err)
	}
}

func TestScheduler_AddProcess_CronSchedule(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Cron: "* * * * *", // Every minute
	}
	err := s.AddProcess("test-cron", config)
	if err != nil {
		t.Errorf("AddProcess() with cron config returned error: %v", err)
	}

	if !s.IsScheduled("test-cron") {
		t.Error("IsScheduled() should return true for scheduled process")
	}
}

func TestScheduler_AddProcess_IntervalSchedule(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Interval: "1h",
	}
	err := s.AddProcess("test-interval", config)
	if err != nil {
		t.Errorf("AddProcess() with interval config returned error: %v", err)
	}

	if !s.IsScheduled("test-interval") {
		t.Error("IsScheduled() should return true for scheduled process")
	}
}

func TestScheduler_AddProcess_InvalidInterval(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Interval: "invalid",
	}
	err := s.AddProcess("test-invalid", config)
	if err == nil {
		t.Error("AddProcess() with invalid interval should return error")
	}
}

func TestScheduler_GetNextRunTime_Scheduled(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Interval: "1h",
	}
	_ = s.AddProcess("test", config)
	s.Start()
	defer func() { _ = s.Stop() }()

	nextRun := s.GetNextRunTime("test")
	if nextRun == nil {
		t.Error("GetNextRunTime() should return non-nil for scheduled process")
	}
	if nextRun != nil && nextRun.Before(time.Now()) {
		t.Error("GetNextRunTime() should return a future time")
	}
}

func TestScheduler_GetNextRunTime_NotScheduled(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	nextRun := s.GetNextRunTime("nonexistent")
	if nextRun != nil {
		t.Error("GetNextRunTime() should return nil for non-scheduled process")
	}
}

func TestScheduler_IsScheduled(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Interval: "1h",
	}
	_ = s.AddProcess("scheduled", config)

	if !s.IsScheduled("scheduled") {
		t.Error("IsScheduled() should return true for scheduled process")
	}
	if s.IsScheduled("not-scheduled") {
		t.Error("IsScheduled() should return false for non-scheduled process")
	}
}

func TestScheduler_GetScheduledProcesses(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Interval: "1h",
	}
	_ = s.AddProcess("process1", config)
	_ = s.AddProcess("process2", config)

	processes := s.GetScheduledProcesses()
	if len(processes) != 2 {
		t.Errorf("GetScheduledProcesses() returned %d processes, want 2", len(processes))
	}
}

func TestScheduler_StartStop(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	// Should not panic
	s.Start()
	err := s.Stop()
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestScheduler_RunOnStart(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Interval:   "1h",
		RunOnStart: true,
	}
	err := s.AddProcess("run-on-start", config)
	if err != nil {
		t.Fatalf("AddProcess() returned error: %v", err)
	}

	s.Start()
	// Give some time for the immediate execution
	time.Sleep(100 * time.Millisecond)
	_ = s.Stop()

	started := mock.getStartedProcesses()
	if len(started) == 0 {
		t.Error("RunOnStart=true should have started the process immediately")
	}
}

func TestScheduler_AddProcess_WithTimezone(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)

	config := &types.ScheduleConfig{
		Cron:     "0 2 * * *",
		Timezone: "UTC",
	}
	err := s.AddProcess("test-tz", config)
	if err != nil {
		t.Errorf("AddProcess() with timezone returned error: %v", err)
	}

	if !s.IsScheduled("test-tz") {
		t.Error("IsScheduled() should return true")
	}
}

func TestScheduler_PauseResume(t *testing.T) {
	mock := &mockProcessStarter{}
	s, _ := New(mock)
	s.Start()
	defer func() { _ = s.Stop() }()

	config := &types.ScheduleConfig{
		Interval: "1h",
	}
	_ = s.AddProcess("test-pause", config)

	// Pause
	err := s.PauseProcess("test-pause")
	if err != nil {
		t.Errorf("PauseProcess() returned error: %v", err)
	}

	nextRun := s.GetNextRunTime("test-pause")
	if nextRun != nil {
		t.Error("GetNextRunTime() should return nil for paused process")
	}

	// Resume
	err = s.ResumeProcess("test-pause")
	if err != nil {
		t.Errorf("ResumeProcess() returned error: %v", err)
	}

	nextRun = s.GetNextRunTime("test-pause")
	if nextRun == nil {
		t.Error("GetNextRunTime() should return non-nil for resumed process")
	}
}
