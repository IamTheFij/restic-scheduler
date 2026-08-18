package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"

	"git.iamthefij.com/iamthefij/restic-scheduler/metrics"
	"git.iamthefij.com/iamthefij/restic-scheduler/tasks"
)

// In-memory job result storage (shared across scheduler instances)
var (
	jobResultsLock = sync.Mutex{}
	jobResults     = map[string]JobResult{}
)

type ScheduledJob struct {
	job tasks.Job
}

// Run runs the backup job with it's provided configuration.
func (j ScheduledJob) Run() {
	result := JobResult{
		JobName:   j.job.Name,
		JobType:   "backup",
		Success:   true,
		LastError: nil,
		Message:   "",
	}

	metrics.Metrics.JobStartTime.WithLabelValues(j.job.Name).SetToCurrentTime()

	if err := j.job.RunBackup(); err != nil {
		j.job.Logger().Printf("ERROR: Backup failed: %s", err.Error())

		result.Success = false
		result.LastError = err
	}

	snapshots, err := j.job.NewRestic().ReadSnapshots()
	if err != nil {
		// Set the last error on the result only if an actual backup error doesn't already exist
		// An error reading snapshots is less severe than a failure to backup.
		if result.LastError == nil {
			result.LastError = err
		}
	} else {
		metrics.Metrics.SnapshotCurrentCount.WithLabelValues(j.job.Name).Set(float64(len(snapshots)))

		if len(snapshots) > 0 {
			latestSnapshot := snapshots[len(snapshots)-1]
			metrics.Metrics.SnapshotLatestTime.WithLabelValues(j.job.Name).Set(float64(latestSnapshot.Time.Unix()))
		}
	}

	if result.Success {
		metrics.Metrics.JobFailureCount.WithLabelValues(j.job.Name).Set(0.0)
	} else {
		metrics.Metrics.JobFailureCount.WithLabelValues(j.job.Name).Inc()
	}

	JobComplete(result)
}

// RefreshMetrics updates the metrics for this job by reading the current snapshots from restic.
func (j ScheduledJob) RefreshMetrics() {
	snapshots, err := j.job.NewRestic().ReadSnapshots()
	if err != nil {
		j.job.Logger().Printf("ERROR: Failed to read snapshots while refreshing metrics: %s", err.Error())
		return
	}

	metrics.Metrics.SnapshotCurrentCount.WithLabelValues(j.job.Name).Set(float64(len(snapshots)))

	if len(snapshots) > 0 {
		latestSnapshot := snapshots[len(snapshots)-1]
		metrics.Metrics.SnapshotLatestTime.WithLabelValues(j.job.Name).Set(float64(latestSnapshot.Time.Unix()))
	}
}

// Scheduler manages a cron instance and a set of scheduled jobs.
type Scheduler struct {
	mu       sync.Mutex
	cron     *cron.Cron
	jobs     []ScheduledJob
	jobNames []string
	started  bool
}

// NewScheduler constructs an empty Scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// Start schedules the provided jobs and starts the internal cron instance.
// It returns an error if scheduling any job fails. If the scheduler is already
// started, Start will return an error.
func (s *Scheduler) Start(jobs []tasks.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var schedJobs []ScheduledJob
	for _, job := range jobs {
		schedJobs = append(schedJobs, ScheduledJob{job: job})
	}

	if s.started {
		return fmt.Errorf("scheduler already started")
	}

	c := cron.New()
	names := make([]string, 0, len(jobs))

	for _, sj := range schedJobs {
		log.Printf("Scheduling %s", sj.job.Name)

		if _, err := c.AddJob(sj.job.Schedule, sj); err != nil {
			return fmt.Errorf("error scheduling job %s: %w", sj.job.Name, err)
		}

		names = append(names, sj.job.Name)
	}

	// start the scheduler
	c.Start()

	s.cron = c
	s.jobs = schedJobs
	s.jobNames = names
	s.started = true

	return nil
}

// ReplaceJobs stops the current scheduler (waiting for in-flight jobs to finish)
// and starts a new scheduler with newJobs. This is a graceful replacement.
func (s *Scheduler) ReplaceJobs(newJobs []tasks.Job) error {
	// Swap out safely: stop old cron and wait for running jobs to finish.
	s.mu.Lock()
	// Keep a snapshot of previous jobs for potential fallback (caller may handle)
	var prevJobs []tasks.Job
	for _, job := range s.jobs {
		prevJobs = append(prevJobs, job.job)
	}
	s.mu.Unlock()

	s.StopGraceful()

	// Start new scheduler with new jobs.
	if err := s.Start(newJobs); err != nil {
		// If starting new scheduler fails, attempt to restart previous jobs if available.
		if len(prevJobs) > 0 {
			log.Printf("failed to start new scheduler: %v; attempting to restart previous jobs", err)

			if restartErr := s.Start(prevJobs); restartErr != nil {
				return fmt.Errorf("failed to restart previous scheduler after replace error: %w (original replace err: %w)", restartErr, err)
			}

			// Returning original error but scheduler recovered to previous state.
			return fmt.Errorf("replace failed but previous scheduler restarted: %w", err)
		}

		return fmt.Errorf("replace failed and no previous jobs to restart: %w", err)
	}

	return nil
}

// StopNow stops scheduling and returns immediately (does not wait for running jobs to finish).
func (s *Scheduler) StopNow() {
	s.mu.Lock()
	c := s.cron
	s.cron = nil
	s.started = false
	s.mu.Unlock()

	if c != nil {
		// Stop returns a context that is closed when running jobs finish; we intentionally don't wait here.
		c.Stop()
	}
}

// StopGraceful stops the scheduler and waits for any running jobs to finish before returning.
func (s *Scheduler) StopGraceful() {
	s.mu.Lock()
	c := s.cron
	s.cron = nil
	s.started = false
	s.mu.Unlock()

	if c != nil {
		ctx := c.Stop()
		<-ctx.Done()
	}
}

// ActiveJobNames returns a snapshot of the currently scheduled job names.
func (s *Scheduler) ActiveJobNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, len(s.jobNames))
	copy(out, s.jobNames)

	return out
}

// JobResult is a simple summary of the last run for a job.
type JobResult struct {
	JobName   string
	JobType   string
	Success   bool
	LastError error
	Message   string
}

func (r JobResult) Format() string {
	return fmt.Sprintf("%s %s ok? %v\n\n%+v", r.JobName, r.JobType, r.Success, r.LastError)
}

// JobComplete records completion state for a job into the in-memory map.
func JobComplete(result JobResult) {
	log.Printf("Completed job %+v\n", result)

	jobResultsLock.Lock()
	jobResults[result.JobName] = result
	jobResultsLock.Unlock()
}

// writeJobResult writes the job result as JSON to the provided writer.
func writeJobResult(writer http.ResponseWriter, jobName string) {
	writer.Header().Set("Content-Type", "application/json")

	jobResultsLock.Lock()
	jobResult, ok := jobResults[jobName]
	jobResultsLock.Unlock()

	if ok {
		if !jobResult.Success {
			// Set a 503 status code if the last job run was not successful
			writer.WriteHeader(http.StatusServiceUnavailable)
		}

		// Set message from LastError if available
		if jobResult.LastError != nil {
			jobResult.Message = jobResult.LastError.Error()
		}

		// Build a JSON object that maps to the exported JobResult fields (excluding the LastError field,
		// which cannot be marshalled directly). Using the exported field names ensures compatibility
		// with tests that unmarshal into main.JobResult.
		out := map[string]interface{}{
			"JobName": jobResult.JobName,
			"JobType": jobResult.JobType,
			"Success": jobResult.Success,
			"Message": jobResult.Message,
		}

		if err := json.NewEncoder(writer).Encode(out); err != nil {
			http.Error(writer, fmt.Sprintf("failed writing json for %s", jobResult.JobName), http.StatusInternalServerError)
		}

		return
	}

	// Job not found
	writer.WriteHeader(http.StatusNotFound)
	_, _ = writer.Write([]byte("{\"Message\": \"Unknown job\"}"))
}

// HealthHandleFunc handles health check requests.
func HealthHandleFunc(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	if jobName, ok := query["job"]; ok {
		writeJobResult(writer, jobName[0])
		return
	}

	_, _ = writer.Write([]byte("ok"))
}

// ActiveHandleFunc returns the currently scheduled job names. It expects a scheduler
// instance to be provided via closure in RunHTTPHandlers.
func ActiveHandleFunc(writer http.ResponseWriter, request *http.Request, names []string) {
	writer.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(writer).Encode(map[string][]string{"active_jobs": names}); err != nil {
		http.Error(writer, "failed to encode active jobs", http.StatusInternalServerError)
	}
}

// RunHTTPHandlers registers HTTP handlers for /health, /metrics and /active.
// The active handler uses the provided scheduler to get the current job names.
func RunHTTPHandlers(addr string, sched *Scheduler) error {
	http.HandleFunc("/health", HealthHandleFunc)
	http.Handle("/metrics", promhttp.HandlerFor(
		metrics.Metrics.Registry,
		promhttp.HandlerOpts{Registry: metrics.Metrics.Registry}, //nolint:exhaustruct
	))

	// active handler closure
	http.HandleFunc("/active", func(w http.ResponseWriter, r *http.Request) {
		if sched == nil {
			ActiveHandleFunc(w, r, []string{})
			return
		}

		ActiveHandleFunc(w, r, sched.ActiveJobNames())
	})

	return fmt.Errorf("error on http server: %w", http.ListenAndServe(addr, nil)) //#nosec: g114
}
