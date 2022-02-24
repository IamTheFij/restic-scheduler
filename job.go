package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/robfig/cron/v3"
)

const WorkDirPerms = 0o666

var (
	ErrNoJobsFound       = errors.New("no jobs found and at least one job is required")
	ErrMissingField      = errors.New("missing config field")
	ErrMissingBlock      = errors.New("missing config block")
	ErrMutuallyExclusive = errors.New("mutually exclusive values not valid")
)

type TaskConfig struct {
	JobDir string
	Env    map[string]string
	Logger *log.Logger
	Restic *ResticCmd
}

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

// ExecutableTask is a task to be run before or after backup/retore.
type ExecutableTask interface {
	RunBackup(cfg TaskConfig) error
	RunRestore(cfg TaskConfig) error
	Name() string
}

// JobTaskScript is a sript to be executed as part of a job task.
type JobTaskScript struct {
	OnBackup   string `hcl:"on_backup,optional"`
	OnRestore  string `hcl:"on_restore,optional"`
	FromJobDir bool   `hcl:"from_job_dir,optional"`
	env        map[string]string
	name       string
}

func (t JobTaskScript) run(script string, cfg TaskConfig) error {
	if script == "" {
		return nil
	}

	env := MergeEnvMap(cfg.Env, t.env)
	if env == nil {
		env = map[string]string{}
	}

	// Inject the job directory to the running task
	env["RESTIC_JOB_DIR"] = cfg.JobDir

	cwd := ""
	if t.FromJobDir {
		cwd = cfg.JobDir
	}

	if err := RunShell(script, cwd, env, cfg.Logger); err != nil {
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

func (t JobTaskScript) Name() string {
	return t.name
}

func (t *JobTaskScript) SetName(name string) {
	t.name = name
}

// JobTaskMySQL is a sqlite backup task that performs required pre and post tasks.
type JobTaskMySQL struct {
	Name     string   `hcl:"name,label"`
	Hostname string   `hcl:"hostname,optional"`
	Database string   `hcl:"database,optional"`
	Username string   `hcl:"username,optional"`
	Password string   `hcl:"password,optional"`
	Tables   []string `hcl:"tables,optional"`
}

func (t JobTaskMySQL) Filename() string {
	return fmt.Sprintf("%s.sql", t.Name)
}

func (t JobTaskMySQL) Validate() error {
	if invalidChars := "'\";"; strings.ContainsAny(t.Name, invalidChars) {
		return fmt.Errorf("mysql task %s has an invalid name. The name may not contain %s", t.Name, invalidChars)
	}

	if len(t.Tables) > 0 && t.Database == "" {
		return fmt.Errorf("mysql task %s is invalid. Must specify a database to use tables: %w", t.Name, ErrMissingField)
	}

	return nil
}

func (t JobTaskMySQL) GetPreTask() ExecutableTask {
	command := []string{"mysqldump", "--result-file", fmt.Sprintf("'./%s'", t.Filename())}

	if t.Hostname != "" {
		command = append(command, "--host", t.Hostname)
	}

	if t.Username != "" {
		command = append(command, "--user", t.Username)
	}

	if t.Password != "" {
		command = append(command, "--password", t.Password)
	}

	if t.Database != "" {
		command = append(command, t.Database)
	}

	command = append(command, t.Tables...)

	return JobTaskScript{
		name:       t.Name,
		env:        nil,
		OnBackup:   strings.Join(command, " "),
		OnRestore:  "",
		FromJobDir: true,
	}
}

func (t JobTaskMySQL) GetPostTask() ExecutableTask {
	command := []string{"mysql"}

	if t.Hostname != "" {
		command = append(command, "--host", t.Hostname)
	}

	if t.Username != "" {
		command = append(command, "--user", t.Username)
	}

	if t.Password != "" {
		command = append(command, "--password", t.Password)
	}

	command = append(command, "<", fmt.Sprintf("'./%s'", t.Filename()))

	return JobTaskScript{
		name:       t.Name,
		env:        nil,
		OnBackup:   "",
		OnRestore:  strings.Join(command, " "),
		FromJobDir: true,
	}
}

// JobTaskSqlite is a sqlite backup task that performs required pre and post tasks.
type JobTaskSqlite struct {
	Name string `hcl:"name,label"`
	Path string `hcl:"path"`
}

func (t JobTaskSqlite) Filename() string {
	return fmt.Sprintf("%s.db.bak", t.Name)
}

func (t JobTaskSqlite) Validate() error {
	if invalidChars := "'\";"; strings.ContainsAny(t.Name, invalidChars) {
		return fmt.Errorf("sqlite task %s has an invalid name. The name may not contain %s", t.Name, invalidChars)
	}

	return nil
}

func (t JobTaskSqlite) GetPreTask() ExecutableTask {
	return JobTaskScript{
		name: t.Name,
		env:  nil,
		OnBackup: fmt.Sprintf(
			"sqlite3 %s '.backup $RESTIC_JOB_DIR/%s'",
			t.Path, t.Filename(),
		),
		OnRestore:  "",
		FromJobDir: false,
	}
}

func (t JobTaskSqlite) GetPostTask() ExecutableTask {
	return JobTaskScript{
		name:       t.Name,
		env:        nil,
		OnBackup:   "",
		OnRestore:  fmt.Sprintf("cp '$RESTIC_JOB_DIR/%s' '%s'", t.Filename(), t.Path),
		FromJobDir: false,
	}
}

type BackupFilesTask struct {
	Files       []string     `hcl:"files"`
	BackupOpts  *BackupOpts  `hcl:"backup_opts,block"`
	RestoreOpts *RestoreOpts `hcl:"restore_opts,block"`
	name        string
}

func (t BackupFilesTask) RunBackup(cfg TaskConfig) error {
	if t.BackupOpts == nil {
		t.BackupOpts = &BackupOpts{} // nolint:exhaustivestruct
	}

	if err := cfg.Restic.Backup(append(t.Files, cfg.JobDir), *t.BackupOpts); err != nil {
		err = fmt.Errorf("failed backing up files: %w", err)
		cfg.Logger.Fatal(err)

		return err
	}

	return nil
}

func (t BackupFilesTask) RunRestore(cfg TaskConfig) error {
	if t.RestoreOpts == nil {
		t.RestoreOpts = &RestoreOpts{} // nolint:exhaustivestruct
	}

	if err := cfg.Restic.Restore("latest", *t.RestoreOpts); err != nil {
		err = fmt.Errorf("failed restoring files: %w", err)
		cfg.Logger.Fatal(err)

		return err
	}

	return nil
}

func (t BackupFilesTask) Name() string {
	return t.name
}

func (t *BackupFilesTask) SetName(name string) {
	t.name = name
}

// JobTask represents a single task within a backup job.
type JobTask struct {
	Name    string           `hcl:"name,label"`
	Scripts []JobTaskScript  `hcl:"script,block"`
	Backup  *BackupFilesTask `hcl:"backup,block"`
}

func (t JobTask) Validate() error {
	if len(t.Scripts) > 0 && t.Backup != nil {
		return fmt.Errorf(
			"task %s is invalid. script and backup blocks are mutually exclusive: %w",
			t.Name,
			ErrMutuallyExclusive,
		)
	}

	if len(t.Scripts) == 0 && t.Backup == nil {
		return fmt.Errorf(
			"task %s is invalid. Ether script or backup blocks must be provided: %w",
			t.Name,
			ErrMutuallyExclusive,
		)
	}

	return nil
}

func (t JobTask) GetTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	for _, exTask := range t.Scripts {
		exTask.SetName(t.Name)
		allTasks = append(allTasks, exTask)
	}

	if t.Backup != nil {
		t.Backup.SetName(t.Name)
		allTasks = append(allTasks, t.Backup)
	}

	return allTasks
}

// Job contains all configuration required to construct and run a backup
// and restore job.
type Job struct {
	Name     string       `hcl:"name,label"`
	Schedule string       `hcl:"schedule"`
	Config   ResticConfig `hcl:"config,block"`
	Tasks    []JobTask    `hcl:"task,block"`
	Forget   *ForgetOpts  `hcl:"forget,block"`

	// Meta Tasks
	MySQL  []JobTaskMySQL  `hcl:"mysql,block"`
	Sqlite []JobTaskSqlite `hcl:"sqlite,block"`
}

func (j Job) validateTasks() error {
	if len(j.Tasks) == 0 {
		return fmt.Errorf("job %s is missing tasks: %w", j.Name, ErrMissingBlock)
	}

	foundBackup := false

	for _, task := range j.Tasks {
		if task.Backup != nil {
			foundBackup = true
		}

		if err := task.Validate(); err != nil {
			return fmt.Errorf("job %s has an inavalid task: %w", j.Name, err)
		}
	}

	if !foundBackup {
		return fmt.Errorf("job %s is missing a backup task: %w", j.Name, ErrMissingBlock)
	}

	return nil
}

func (j Job) Validate() error {
	if j.Name == "" {
		return fmt.Errorf("job is missing name: %w", ErrMissingField)
	}

	if _, err := cron.ParseStandard(j.Schedule); err != nil {
		return fmt.Errorf("job %s has an invalid schedule: %w", j.Name, err)
	}

	if err := j.Config.Validate(); err != nil {
		return fmt.Errorf("job %s has invalid config: %w", j.Name, err)
	}

	if err := j.validateTasks(); err != nil {
		return err
	}

	for _, mysql := range j.MySQL {
		if err := mysql.Validate(); err != nil {
			return fmt.Errorf("job %s has an inavalid task: %w", j.Name, err)
		}
	}

	for _, sqlite := range j.Sqlite {
		if err := sqlite.Validate(); err != nil {
			return fmt.Errorf("job %s has an inavalid task: %w", j.Name, err)
		}
	}

	return nil
}

func (j Job) AllTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	// Pre tasks
	for _, mysql := range j.MySQL {
		allTasks = append(allTasks, mysql.GetPreTask())
	}

	for _, sqlite := range j.Sqlite {
		allTasks = append(allTasks, sqlite.GetPreTask())
	}

	// Get ordered tasks
	for _, jobTask := range j.Tasks {
		allTasks = append(allTasks, jobTask.GetTasks()...)
	}

	// Post tasks
	for _, mysql := range j.MySQL {
		allTasks = append(allTasks, mysql.GetPreTask())
	}

	for _, sqlite := range j.Sqlite {
		allTasks = append(allTasks, sqlite.GetPreTask())
	}

	return allTasks
}

func (j Job) JobDir() string {
	cwd := filepath.Join("/restic_backup", j.Name)
	_ = os.MkdirAll(cwd, WorkDirPerms)

	return cwd
}

/*
 * func NewTaskConfig(jobDir string, jobLogger *log.Logger, restic *ResticCmd, taskName string) TaskConfig {
 * 	return TaskConfig{
 * 		JobDir: jobDir,
 * 		Logger: GetChildLogger(jobLogger, taskName),
 * 		Restic: restic,
 * 		Env:    nil,
 * 	}
 * }
 */

func (j Job) RunBackup() error {
	logger := GetLogger(j.Name)
	restic := j.NewRestic()
	jobDir := j.JobDir()

	if err := restic.EnsureInit(); err != nil {
		return fmt.Errorf("failed to init restic for job %s: %w", j.Name, err)
	}

	for _, exTask := range j.AllTasks() {
		taskCfg := TaskConfig{
			JobDir: jobDir,
			Logger: GetChildLogger(logger, exTask.Name()),
			Restic: restic,
			Env:    nil,
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

func (j Job) RunRestore() error {
	logger := GetLogger(j.Name)
	restic := j.NewRestic()
	jobDir := j.JobDir()

	if err := restic.RunRestic("snapshots", NoOpts{}); err != nil {
		return fmt.Errorf("no repository or snapshots for job %s: %w", j.Name, err)
	}

	for _, exTask := range j.AllTasks() {
		taskCfg := TaskConfig{
			JobDir: jobDir,
			Logger: GetChildLogger(logger, exTask.Name()),
			Restic: restic,
			Env:    nil,
		}

		if err := exTask.RunRestore(taskCfg); err != nil {
			return fmt.Errorf("failed running job %s: %w", j.Name, err)
		}
	}

	return nil
}

func (j Job) NewRestic() *ResticCmd {
	return &ResticCmd{
		Logger:     GetLogger(j.Name),
		Repo:       j.Config.Repo,
		Env:        j.Config.Env,
		Passphrase: j.Config.Passphrase,
		GlobalOpts: j.Config.GlobalOpts,
		Cwd:        "",
	}
}

type Config struct {
	Jobs []Job `hcl:"job,block"`
}

func (c Config) Validate() error {
	if len(c.Jobs) == 0 {
		return ErrNoJobsFound
	}

	for _, job := range c.Jobs {
		if err := job.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/***

job "My App" {
	schedule = "* * * * *"
	config {
		repo = "s3://..."
		passphrase = "foo"
	}

	task "Dump mysql" {
		mysql {
			hostname = "foo"
			username = "bar"
		}
	}

	task "Create biz file" {
		on_backup {
			body = <<EOF
			echo foo > /biz.txt
			EOF
		}
	}

	task "Backup data files" {
		files = [
		"/foo/bar",
		"/biz.txt",
		]
	}
}

***/
