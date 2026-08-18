package tasks

import (
	"fmt"
	"log"

	"git.iamthefij.com/iamthefij/restic-scheduler/restic"
	"git.iamthefij.com/iamthefij/restic-scheduler/utils"
)

type TaskConfig struct {
	BackupPaths     []string
	Env             map[string]string
	Logger          *log.Logger
	Restic          *restic.Restic
	RestoreSnapshot string
}

// ExecutableTask is a task to be run before or after backup/retore.
type ExecutableTask interface {
	RunBackup(cfg TaskConfig) error
	RunRestore(cfg TaskConfig) error
	Name() string
	Paths() []string
}

// JobTaskScript is a sript to be executed as part of a job task.
type JobTaskScript struct {
	OnBackup    string            `hcl:"on_backup,optional"`
	OnRestore   string            `hcl:"on_restore,optional"`
	Cwd         string            `hcl:"cwd,optional"`
	Env         map[string]string `hcl:"env,optional"`
	BackupPaths []string          `hcl:"backup_paths,optional"`
	name        string
}

func (t JobTaskScript) run(script string, cfg TaskConfig) error {
	if script == "" {
		return nil
	}

	env := utils.MergeEnvMap(cfg.Env, t.Env)
	if env == nil {
		env = map[string]string{}
	}

	if err := utils.RunShell(script, t.Cwd, env, cfg.Logger); err != nil {
		return fmt.Errorf("failed running task script %s: %w", t.Name(), err)
	}

	return nil
}

// RunBackup runs script on backup.
func (t JobTaskScript) RunBackup(cfg TaskConfig) error {
	return t.run(t.OnBackup, cfg)
}

// RunRestore script on restore.
func (t JobTaskScript) RunRestore(cfg TaskConfig) error {
	return t.run(t.OnRestore, cfg)
}

// Name returns the name of this task.
func (t JobTaskScript) Name() string {
	return t.name
}

// SetName sets the name for the task.
func (t *JobTaskScript) SetName(name string) {
	t.name = name
}

// Paths returns all paths to be backed up from this task.
func (t JobTaskScript) Paths() []string {
	return t.BackupPaths
}

type DatabaseTask interface {
	Validate() error
	GetPreTask() ExecutableTask
	GetPostTask() ExecutableTask
	Paths() []string
}

// BackupFilesTask is the main task for executing a backup to a remote.
type BackupFilesTask struct {
	BackupPaths []string            `hcl:"paths"`
	BackupOpts  *restic.BackupOpts  `hcl:"backup_opts,block"`
	RestoreOpts *restic.RestoreOpts `hcl:"restore_opts,block"`
	name        string
}

// RunBackup runs the backup task sending data to the repository.
func (t BackupFilesTask) RunBackup(cfg TaskConfig) error {
	if t.BackupOpts == nil {
		t.BackupOpts = &restic.BackupOpts{} //nolint:exhaustruct
	}

	if err := cfg.Restic.Backup(cfg.BackupPaths, *t.BackupOpts); err != nil {
		err = fmt.Errorf("failed backing up paths: %w", err)
		cfg.Logger.Print(err)

		return err
	}

	return nil
}

// RunRestore runs the restore task for the backup, pulling the data from the repository.
func (t BackupFilesTask) RunRestore(cfg TaskConfig) error {
	if t.RestoreOpts == nil {
		t.RestoreOpts = &restic.RestoreOpts{} //nolint:exhaustruct
	}

	if cfg.RestoreSnapshot == "" {
		cfg.RestoreSnapshot = "latest"
	}

	if err := cfg.Restic.Restore(cfg.RestoreSnapshot, *t.RestoreOpts); err != nil {
		err = fmt.Errorf("failed restoring paths: %w", err)
		cfg.Logger.Print(err)

		return err
	}

	return nil
}

// Name returns the name of this task.
func (t BackupFilesTask) Name() string {
	return t.name
}

// SetName sets the name for the task.
func (t *BackupFilesTask) SetName(name string) {
	t.name = name
}

// Validate ensures that this tasks configuration is valid.
func (t *BackupFilesTask) Validate() error {
	// We don't need to validate paths because paths can be added by other tasks dynamically. Instead, we rely on typing here.
	return nil
}

// Paths returns all paths to be backed up from this task.
func (t BackupFilesTask) Paths() []string {
	return t.BackupPaths
}

// JobTask represents a single task within a backup job.
type JobTask struct {
	Name        string            `hcl:"name,label"`
	PreScripts  []JobTaskScript   `hcl:"pre_script,block"`
	PostScripts []JobTaskScript   `hcl:"post_script,block"`
	MySQL       []JobTaskMySQL    `hcl:"mysql,block"`
	MariaDB     []JobTaskMariaDB  `hcl:"mariadb,block"`
	Postgres    []JobTaskPostgres `hcl:"postgres,block"`
	Sqlite      []JobTaskSqlite   `hcl:"sqlite,block"`
}

// // DatabaseTasks returns a slice of all DatabaseTasks in this JobTask.
func (t JobTask) DatabaseTasks() []DatabaseTask {
	tasks := []DatabaseTask{}

	for _, task := range t.MySQL {
		tasks = append(tasks, task)
	}

	for _, task := range t.MariaDB {
		tasks = append(tasks, task)
	}

	for _, task := range t.Postgres {
		tasks = append(tasks, task)
	}

	for _, task := range t.Sqlite {
		tasks = append(tasks, task)
	}

	return tasks
}

// Validate ensures that this tasks configuration is valid.
func (t JobTask) Validate() error {
	// NOTE: Might make task types mutually exclusive because order is confusing even if deterministic
	if t.Name == "" {
		return fmt.Errorf("task is missing a name: %w", ErrMissingField)
	}

	return nil
}

// GetPreTasks returns all ExecutableTasks that should be run before backup or restore.
func (t JobTask) GetPreTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	for _, task := range t.DatabaseTasks() {
		allTasks = append(allTasks, task.GetPreTask())
	}

	for _, exTask := range t.PreScripts {
		exTask.SetName(t.Name)
		allTasks = append(allTasks, exTask)
	}

	return allTasks
}

// GetPostTasks returns all ExecutableTasks that should be run after backup or restore.
func (t JobTask) GetPostTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	for _, task := range t.DatabaseTasks() {
		allTasks = append(allTasks, task.GetPostTask())
	}

	for _, exTask := range t.PostScripts {
		exTask.SetName(t.Name)
		allTasks = append(allTasks, exTask)
	}

	return allTasks
}

// BackupPaths returns a slice of all paths this task asks to back up
func (t JobTask) BackupPaths() []string {
	paths := []string{}

	for _, task := range t.DatabaseTasks() {
		paths = append(paths, task.Paths()...)
	}

	for _, task := range t.PostScripts {
		paths = append(paths, task.Paths()...)
	}

	for _, task := range t.PreScripts {
		paths = append(paths, task.Paths()...)
	}

	return paths
}
