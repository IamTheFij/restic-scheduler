package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/robfig/cron/v3"
)

var (
	ErrNoJobsFound        = errors.New("no jobs found and at least one job is required")
	ErrMissingField       = errors.New("missing config field")
	ErrMutuallyExclusive  = errors.New("mutually exclusive values not valid")
	ErrInvalidConfigValue = errors.New("invalid config value")

	// JobBaseDir is the root for the creation of restic job dirs. These will generally
	// house SQL dumps prior to backup and before restoration.
	JobBaseDir = filepath.Join(os.TempDir(), "restic_scheduler")
)

// ResticConfig is all configuration to be sent to Restic.
type ResticConfig struct {
	Repo       string            `hcl:"repo"`
	Passphrase string            `hcl:"passphrase,optional"`
	Env        map[string]string `hcl:"env,optional"`
	GlobalOpts *ResticGlobalOpts `hcl:"options,block"`
}

func (r ResticConfig) Validate() error {
	if r.Passphrase == "" && (r.GlobalOpts == nil || r.GlobalOpts.PasswordFile == "") {
		return fmt.Errorf(
			"either config { Passphrase = string } or config { options { PasswordFile = string } } must be set: %w",
			ErrMutuallyExclusive,
		)
	}

	if r.Passphrase != "" && r.GlobalOpts != nil && r.GlobalOpts.PasswordFile != "" {
		return fmt.Errorf(
			"only one of config { Passphrase = string } or config { options { PasswordFile = string } } may be set: %w",
			ErrMutuallyExclusive,
		)
	}

	return nil
}

// Job contains all configuration required to construct and run a backup
// and restore job.
type Job struct {
	Name     string          `hcl:"name,label"`
	Schedule string          `hcl:"schedule"`
	Config   *ResticConfig   `hcl:"config,block"`
	Tasks    []JobTask       `hcl:"task,block"`
	Backup   BackupFilesTask `hcl:"backup,block"`
	Forget   *ForgetOpts     `hcl:"forget,block"`

	// Meta Tasks
	// NOTE: Now that these are also available within a task
	// these could be removed to make task order more obvious
	MySQL    []JobTaskMySQL    `hcl:"mysql,block"`
	Postgres []JobTaskPostgres `hcl:"postgres,block"`
	Sqlite   []JobTaskSqlite   `hcl:"sqlite,block"`

	// Metrics and health
	healthy bool
	lastErr error
}

func (j Job) validateTasks() error {
	for _, task := range j.Tasks {
		if err := task.Validate(); err != nil {
			return fmt.Errorf("job %s has an invalid task: %w", j.Name, err)
		}
	}

	for _, mysql := range j.MySQL {
		if err := mysql.Validate(); err != nil {
			return fmt.Errorf("job %s has an invalid task: %w", j.Name, err)
		}
	}

	for _, pg := range j.Postgres {
		if err := pg.Validate(); err != nil {
			return fmt.Errorf("job %s has an invalid task: %w", j.Name, err)
		}
	}

	for _, sqlite := range j.Sqlite {
		if err := sqlite.Validate(); err != nil {
			return fmt.Errorf("job %s has an invalid task: %w", j.Name, err)
		}
	}

	return nil
}

func (j Job) Validate() error {
	if j.Name == "" {
		return fmt.Errorf("job is missing name: %w", ErrMissingField)
	}

	if _, err := cron.ParseStandard(j.Schedule); err != nil {
		return fmt.Errorf("job %s has an invalid schedule: %w: %w", j.Name, err, ErrInvalidConfigValue)
	}

	if j.Config == nil {
		return fmt.Errorf("job %s is missing restic config: %w", j.Name, ErrMissingField)
	}

	if err := j.Config.Validate(); err != nil {
		return fmt.Errorf("job %s has invalid config: %w", j.Name, err)
	}

	if err := j.validateTasks(); err != nil {
		return err
	}

	if err := j.Backup.Validate(); err != nil {
		return fmt.Errorf("job %s has an invalid backup config: %w", j.Name, err)
	}

	return nil
}

func (j Job) AllTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	// Pre tasks
	for _, mysql := range j.MySQL {
		allTasks = append(allTasks, mysql.GetPreTask())
	}

	for _, pg := range j.Postgres {
		allTasks = append(allTasks, pg.GetPreTask())
	}

	for _, sqlite := range j.Sqlite {
		allTasks = append(allTasks, sqlite.GetPreTask())
	}

	for _, jobTask := range j.Tasks {
		allTasks = append(allTasks, jobTask.GetPreTasks()...)
	}

	// Add backup task
	allTasks = append(allTasks, j.Backup)

	// Post tasks
	for _, jobTask := range j.Tasks {
		allTasks = append(allTasks, jobTask.GetPostTasks()...)
	}

	for _, mysql := range j.MySQL {
		allTasks = append(allTasks, mysql.GetPostTask())
	}

	for _, pg := range j.Postgres {
		allTasks = append(allTasks, pg.GetPostTask())
	}

	for _, sqlite := range j.Sqlite {
		allTasks = append(allTasks, sqlite.GetPostTask())
	}

	return allTasks
}

func (j Job) BackupPaths() []string {
	paths := j.Backup.Paths

	for _, t := range j.MySQL {
		paths = append(paths, t.DumpToPath)
	}

	for _, t := range j.Postgres {
		paths = append(paths, t.DumpToPath)
	}

	for _, t := range j.Sqlite {
		paths = append(paths, t.DumpToPath)
	}

	return paths
}

func (j Job) RunBackup() error {
	logger := GetLogger(j.Name)
	restic := j.NewRestic()

	if err := restic.EnsureInit(); err != nil {
		return fmt.Errorf("failed to init restic for job %s: %w", j.Name, err)
	}

	backupPaths := j.BackupPaths()

	for _, exTask := range j.AllTasks() {
		taskCfg := TaskConfig{
			BackupPaths: backupPaths,
			Logger:      GetChildLogger(logger, exTask.Name()),
			Restic:      restic,
			Env:         j.Config.Env,
		}

		if err := exTask.RunBackup(taskCfg); err != nil {
			return fmt.Errorf("failed running job %s: %w", j.Name, err)
		}
	}

	if j.Forget != nil {
		if err := restic.Forget(*j.Forget); err != nil {
			return fmt.Errorf("failed forgetting and pruning job %s: %w", j.Name, err)
		}
	}

	return nil
}

func (j Job) Logger() *log.Logger {
	return GetLogger(j.Name)
}

func (j Job) RunRestore(snapshot string) error {
	logger := j.Logger()
	restic := j.NewRestic()

	if _, err := restic.RunRestic("snapshots", NoOpts{}); errors.Is(err, ErrRepoNotFound) {
		return fmt.Errorf("no repository or snapshots for job %s: %w", j.Name, err)
	}

	for _, exTask := range j.AllTasks() {
		taskCfg := TaskConfig{
			BackupPaths: nil,
			Logger:      GetChildLogger(logger, exTask.Name()),
			Restic:      restic,
			Env:         j.Config.Env,
		}

		if backupTask, ok := exTask.(BackupFilesTask); ok {
			backupTask.snapshot = snapshot
		}

		if err := exTask.RunRestore(taskCfg); err != nil {
			return fmt.Errorf("failed running job %s: %w", j.Name, err)
		}
	}

	return nil
}

func (j Job) Healthy() (bool, error) {
	return j.healthy, j.lastErr
}

func (j Job) Run() {
	result := JobResult{
		JobName:   j.Name,
		JobType:   "backup",
		Success:   true,
		LastError: nil,
		Message:   "",
	}

	Metrics.JobStartTime.WithLabelValues(j.Name).SetToCurrentTime()

	if err := j.RunBackup(); err != nil {
		j.healthy = false
		j.lastErr = err

		j.Logger().Printf("ERROR: Backup failed: %s", err.Error())

		result.Success = false
		result.LastError = err
	}

	snapshots, err := j.NewRestic().ReadSnapshots()
	if err != nil {
		result.LastError = err
	} else {
		Metrics.SnapshotCurrentCount.WithLabelValues(j.Name).Set(float64(len(snapshots)))

		if len(snapshots) > 0 {
			latestSnapshot := snapshots[len(snapshots)-1]
			Metrics.SnapshotLatestTime.WithLabelValues(j.Name).Set(float64(latestSnapshot.Time.Unix()))
		}
	}

	if result.Success {
		Metrics.JobFailureCount.WithLabelValues(j.Name).Set(0.0)
	} else {
		Metrics.JobFailureCount.WithLabelValues(j.Name).Inc()
	}

	JobComplete(result)
}

func (j Job) NewRestic() *Restic {
	return &Restic{
		Logger:     GetLogger(j.Name),
		Repo:       j.Config.Repo,
		Env:        j.Config.Env,
		Passphrase: j.Config.Passphrase,
		GlobalOpts: j.Config.GlobalOpts,
		Cwd:        "",
	}
}

type Config struct {
	DefaultConfig *ResticConfig `hcl:"default_config,block"`
	Jobs          []Job         `hcl:"job,block"`
}

func (c Config) Validate() error {
	if len(c.Jobs) == 0 {
		return ErrNoJobsFound
	}

	for _, job := range c.Jobs {
		// Use default restic config if no job config is provided
		// TODO: Maybe merge values here
		if job.Config == nil {
			job.Config = c.DefaultConfig
		}

		if err := job.Validate(); err != nil {
			return err
		}
	}

	return nil
}
