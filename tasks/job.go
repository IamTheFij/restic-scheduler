package tasks

import (
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/robfig/cron/v3"

	"git.iamthefij.com/iamthefij/restic-scheduler/restic"
	"git.iamthefij.com/iamthefij/restic-scheduler/utils"
)

var (
	ErrMissingField       = errors.New("missing config field")
	ErrMutuallyExclusive  = errors.New("mutually exclusive values not valid")
	ErrInvalidConfigValue = errors.New("invalid config value")

	// JobBaseDir is the root for the creation of restic job dirs. These will generally
	// house SQL dumps prior to backup and before restoration.
	JobBaseDir = filepath.Join(os.TempDir(), "restic_scheduler")
)

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

type CopyConfig struct {
	TargetRepo       string            `hcl:"target_repo"`
	TargetPassphrase string            `hcl:"target_passphrase"`
	Hosts            []string          `hcl:"hosts,optional"`
	Env              map[string]string `hcl:"env,optional"`
}

type RcloneConfig struct {
	Source      string `hcl:"source"`
	Destination string `hcl:"destination"`
}

// ResticConfig is all configuration to be sent to Restic for the job.
type ResticConfig struct {
	Repo       string                   `hcl:"repo"`
	Passphrase string                   `hcl:"passphrase,optional"`
	Env        map[string]string        `hcl:"env,optional"`
	GlobalOpts *restic.ResticGlobalOpts `hcl:"options,block"`
}

// Validate ensures that the restic configuration is valid and does not contain conflicting values.
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

// Job contains all configuration required to construct and run a backup and restore job.
type Job struct {
	Name     string             `hcl:"name,label"`
	Schedule string             `hcl:"schedule"`
	Config   *ResticConfig      `hcl:"config,block"`
	Tasks    []JobTask          `hcl:"task,block"`
	Backup   BackupFilesTask    `hcl:"backup,block"`
	Forget   *restic.ForgetOpts `hcl:"forget,block"`
	Copy     *CopyConfig        `hcl:"copy,block"`
	Rclone   *RcloneConfig      `hcl:"rclone,block"`

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

	return nil
}

// Validate ensures that a Job config and tasks are all valid.
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

	allBackupPaths := j.BackupPaths()
	if len(allBackupPaths) == 0 {
		return fmt.Errorf("job %s has no backup paths defined: %w", j.Name, ErrMissingField)
	}

	return nil
}

// AllTasks returns an ordered list of ExecutableTasks associated with the Job.
func (j Job) AllTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	// Pre tasks
	for _, jobTask := range j.Tasks {
		allTasks = append(allTasks, jobTask.GetPreTasks()...)
	}

	// Add backup task
	allTasks = append(allTasks, j.Backup)

	// Post tasks
	for _, jobTask := range j.Tasks {
		allTasks = append(allTasks, jobTask.GetPostTasks()...)
	}

	return allTasks
}

// BackupPaths returns all paths to backup defined in any tasks contained with the Job.
func (j Job) BackupPaths() []string {
	paths := j.Backup.Paths()

	for _, jobTask := range j.Tasks {
		paths = append(paths, jobTask.BackupPaths()...)
	}

	return paths
}

// RunBackup executes the backup for this current Job.
func (j *Job) RunBackup() error {
	logger := utils.GetLogger(j.Name)
	r := j.NewRestic()

	if err := r.EnsureInit(restic.InitOpts{}); err != nil {
		j.healthy = false
		j.lastErr = err

		return fmt.Errorf("failed to init restic for job %s: %w", j.Name, err)
	}

	backupPaths := j.BackupPaths()

	for _, exTask := range j.AllTasks() {
		taskCfg := TaskConfig{
			BackupPaths: backupPaths,
			Logger:      utils.GetChildLogger(logger, exTask.Name()),
			Restic:      r,
			Env:         j.Config.Env,
		}

		if err := exTask.RunBackup(taskCfg); err != nil {
			j.healthy = false
			j.lastErr = err

			return fmt.Errorf("failed running job %s: %w", j.Name, err)
		}
	}

	if j.Forget != nil {
		if err := r.Forget(*j.Forget); err != nil {
			j.healthy = false
			j.lastErr = err

			return fmt.Errorf("failed forgetting and pruning job %s: %w", j.Name, err)
		}
	}

	if j.Copy != nil {
		// Make a copy of the base environment
		copyEnv := map[string]string{}
		maps.Copy(copyEnv, j.Config.Env)

		// Move primary repo env variables to source repo variables
		copyEnv["FROM_RESTIC_REPOSITORY"] = utils.MapPop(copyEnv, "RESTIC_REPOSITORY")
		copyEnv["FROM_RESTIC_REPOSITORY_FILE"] = utils.MapPop(copyEnv, "RESTIC_REPOSITORY_FILE")
		copyEnv["FROM_RESTIC_PASSWORD"] = utils.MapPop(copyEnv, "RESTIC_PASSWORD")
		copyEnv["FROM_RESTIC_PASSWORD_COMMAND"] = utils.MapPop(copyEnv, "RESTIC_PASSWORD_COMMAND")
		copyEnv["FROM_RESTIC_PASSWORD_FILE"] = utils.MapPop(copyEnv, "RESTIC_PASSWORD_FILE")

		// Add primary repo passphrase to copyEnv
		if j.Config.GlobalOpts.PasswordFile == "" && j.Config.Passphrase != "" {
			copyEnv["RESTIC_FROM_PASSWORD"] = j.Config.Passphrase
		}

		copyRepo := restic.Restic{
			Logger:     r.Logger,
			Repo:       j.Copy.TargetRepo,
			Passphrase: j.Copy.TargetPassphrase,
			Env:        copyEnv,
			// One bit of undefined behavior is the potential for a global PasswordFile
			// set at the top level and having that pass through to this client.
			// I might move this up to the Restic struct rather than relying on global
			// opts.
			GlobalOpts: j.Config.GlobalOpts,
			Cwd:        "",
		}

		if err := copyRepo.EnsureInit(restic.InitOpts{
			CopyChunkerParams: true,
			FromRepo:          j.Config.Repo,
			FromPasswordFile:  j.Config.GlobalOpts.PasswordFile,
		}); err != nil {
			j.healthy = false
			j.lastErr = err

			return fmt.Errorf("failed to init copy repo for job %s: %w", j.Name, err)
		}

		if err := copyRepo.Copy(restic.CopyOpts{
			Hosts:            j.Copy.Hosts,
			FromRepo:         j.Config.Repo,
			FromPasswordFile: j.Config.GlobalOpts.PasswordFile,
		}); err != nil {
			j.healthy = false
			j.lastErr = err

			return fmt.Errorf("failed copying snapshots for job %s: %w", j.Name, err)
		}
	}

	if j.Rclone != nil {
		cmd := exec.Command("rclone", "sync", j.Rclone.Source, j.Rclone.Destination)

		if err := cmd.Run(); err != nil {
			j.healthy = false
			j.lastErr = err

			return fmt.Errorf("failed syncing repo with rclone for job %s: %w", j.Name, err)
		}
	}

	j.healthy = true
	j.lastErr = nil

	return nil
}

// Logger returns the logger for this job.
func (j Job) Logger() *log.Logger {
	return utils.GetLogger(j.Name)
}

// RunRestore executes a restore of this job for a provided snapshot.
func (j Job) RunRestore(snapshot string) error {
	logger := j.Logger()
	r := j.NewRestic()

	if _, err := r.RunRestic("snapshots", restic.NoOpts{}); errors.Is(err, restic.ErrRepoNotFound) {
		return fmt.Errorf("no repository or snapshots for job %s: %w", j.Name, err)
	}

	for _, exTask := range j.AllTasks() {
		taskCfg := TaskConfig{
			BackupPaths:     nil,
			Logger:          utils.GetChildLogger(logger, exTask.Name()),
			Restic:          r,
			Env:             j.Config.Env,
			RestoreSnapshot: snapshot,
		}

		if err := exTask.RunRestore(taskCfg); err != nil {
			return fmt.Errorf("failed running job %s: %w", j.Name, err)
		}
	}

	return nil
}

func (j Job) RunUnlock() error {
	if err := j.NewRestic().Unlock(restic.UnlockOpts{RemoveAll: true}); err != nil {
		return fmt.Errorf("failed to unlock %s: %w", j.Name, err)
	}

	return nil
}

// Healthy checks if the current job is healthy, returnning a bool and a possible error.
func (j Job) Healthy() (bool, error) {
	return j.healthy, j.lastErr
}

// NewRestic returns a configured Restic command for this job configuration.
func (j Job) NewRestic() *restic.Restic {
	return &restic.Restic{
		Logger:     utils.GetLogger(j.Name),
		Repo:       j.Config.Repo,
		Env:        j.Config.Env,
		Passphrase: j.Config.Passphrase,
		GlobalOpts: j.Config.GlobalOpts,
		Cwd:        "",
	}
}
