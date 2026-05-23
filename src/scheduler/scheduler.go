package scheduler

import (
	"errors"
	"sync"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"
)

// ProcessStarter is an interface for starting processes.
type ProcessStarter interface {
	StartProcess(name string) error
	GetProcessState(name string) (*types.ProcessState, error)
}

// Scheduler manages scheduled process execution.
type Scheduler struct {
	gocronScheduler gocron.Scheduler
	starter         ProcessStarter
	schedules       map[string]*ScheduleEntry
	mutex           sync.RWMutex
}

// ScheduleEntry tracks a scheduled process.
type ScheduleEntry struct {
	ProcessName  string
	Config       *types.ScheduleConfig
	Job          gocron.Job
	RunningCount int
	mutex        sync.Mutex
}

// New creates a new Scheduler.
func New(starter ProcessStarter) (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create gocron scheduler")
		return nil, err
	}
	return &Scheduler{
		gocronScheduler: s,
		starter:         starter,
		schedules:       make(map[string]*ScheduleEntry),
	}, nil
}

// AddProcess adds a scheduled process.
func (s *Scheduler) AddProcess(name string, config *types.ScheduleConfig) error {
	if config == nil || !config.IsScheduled() {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	entry := &ScheduleEntry{
		ProcessName: name,
		Config:      config,
	}

	err := s.addJobInternal(entry)
	if err != nil {
		return err
	}

	s.schedules[name] = entry
	log.Info().Msgf("Scheduled process %s", name)
	return nil
}

func (s *Scheduler) addJobInternal(entry *ScheduleEntry) error {
	config := entry.Config
	name := entry.ProcessName

	// Build job options
	var jobDef gocron.JobDefinition
	var opts []gocron.JobOption

	// Configure schedule type (cron or interval)
	if config.Cron != "" {
		cronExpr := config.Cron
		// Add timezone prefix if specified
		if config.Timezone != "" {
			cronExpr = "CRON_TZ=" + config.Timezone + " " + config.Cron
		}
		jobDef = gocron.CronJob(cronExpr, false) // false = standard 5-field cron
	} else if config.Interval != "" {
		duration, err := config.GetIntervalDuration()
		if err != nil {
			log.Error().Err(err).Msgf("Invalid interval for process %s", name)
			return err
		}
		jobDef = gocron.DurationJob(duration)
	}

	// Configure singleton mode based on max_concurrent
	if config.GetMaxConcurrent() == 1 {
		opts = append(opts, gocron.WithSingletonMode(gocron.LimitModeReschedule))
	}

	// Configure run on start
	if config.RunOnStart {
		opts = append(opts, gocron.WithStartAt(gocron.WithStartImmediately()))
	}

	// Create the task function
	task := gocron.NewTask(func() {
		s.runScheduledProcess(name, entry)
	})

	// Create the job
	job, err := s.gocronScheduler.NewJob(jobDef, task, opts...)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create scheduled job for process %s", name)
		return err
	}

	entry.Job = job
	return nil
}

// runScheduledProcess handles the execution of a scheduled process.
func (s *Scheduler) runScheduledProcess(name string, entry *ScheduleEntry) {
	entry.mutex.Lock()
	maxConcurrent := entry.Config.GetMaxConcurrent()
	if entry.RunningCount >= maxConcurrent {
		entry.mutex.Unlock()
		log.Debug().Msgf("Skipping scheduled run of %s: max concurrent (%d) reached", name, maxConcurrent)
		return
	}
	entry.RunningCount++
	entry.mutex.Unlock()

	defer func() {
		entry.mutex.Lock()
		entry.RunningCount--
		entry.mutex.Unlock()
	}()

	log.Info().Msgf("Starting scheduled process: %s", name)
	if err := s.starter.StartProcess(name); err != nil {
		log.Error().Err(err).Msgf("Failed to start scheduled process %s", name)
	}
}

// GetNextRunTime returns the next scheduled run time for a process.
func (s *Scheduler) GetNextRunTime(name string) *time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if entry, ok := s.schedules[name]; ok {
		if entry.Job == nil {
			return nil
		}
		nextRuns, err := entry.Job.NextRuns(1)
		if err == nil && len(nextRuns) > 0 {
			return &nextRuns[0]
		}
	}
	return nil
}

// PauseProcess pauses a scheduled job.
func (s *Scheduler) PauseProcess(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if entry, ok := s.schedules[name]; ok {
		if entry.Job == nil {
			return nil // Already paused
		}
		log.Debug().Msgf("Pausing schedule for process %s", name)
		err := s.gocronScheduler.RemoveJob(entry.Job.ID())
		entry.Job = nil
		return err
	}
	return nil
}

// RemoveProcess fully removes a scheduled process — both its gocron job and
// its entry in the scheduler's tracking map. Use this when a process is being
// deleted from the project. To temporarily disable a process's schedule while
// keeping the entry tracked (so ResumeProcess can re-arm it later), use
// PauseProcess instead.
//
// If gocron's internal job table no longer has the job (e.g. it was already
// shut down, or a race window between NewJob and the next scheduler tick),
// we still clean up our tracking map and treat the absence as success — the
// caller's intent is "make this process unscheduled" which both paths achieve.
func (s *Scheduler) RemoveProcess(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	entry, ok := s.schedules[name]
	if !ok {
		return nil
	}
	var err error
	if entry.Job != nil {
		if rmErr := s.gocronScheduler.RemoveJob(entry.Job.ID()); rmErr != nil && !errors.Is(rmErr, gocron.ErrJobNotFound) {
			err = rmErr
		}
		entry.Job = nil
	}
	delete(s.schedules, name)
	log.Debug().Msgf("Removed scheduled process %s", name)
	return err
}

// ResumeProcess resumes a paused scheduled job.
func (s *Scheduler) ResumeProcess(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if entry, ok := s.schedules[name]; ok {
		if entry.Job != nil {
			return nil // Already running
		}
		log.Debug().Msgf("Resuming schedule for process %s", name)
		return s.addJobInternal(entry)
	}
	return nil
}

// IsScheduled returns true if the process has a schedule.
func (s *Scheduler) IsScheduled(name string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	_, ok := s.schedules[name]
	return ok
}

// Start begins the scheduler.
func (s *Scheduler) Start() {
	s.gocronScheduler.Start()
	log.Info().Msg("Scheduler started")
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() error {
	err := s.gocronScheduler.Shutdown()
	if err != nil {
		log.Error().Err(err).Msg("Failed to stop scheduler gracefully")
	} else {
		log.Info().Msg("Scheduler stopped")
	}
	return err
}

// GetScheduledProcesses returns a list of scheduled process names.
func (s *Scheduler) GetScheduledProcesses() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	names := make([]string, 0, len(s.schedules))
	for name := range s.schedules {
		names = append(names, name)
	}
	return names
}
