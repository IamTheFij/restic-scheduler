package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const WorkDirPerms = 0o666

type TaskConfig struct {
	JobDir string
	Env    map[string]string
	Logger *log.Logger
	Restic *ResticCmd
}

// ResticConfig is all configuration to be sent to Restic
type ResticConfig struct {
	Repo       string            `hcl:"repo"`
	Passphrase string            `hcl:"passphrase,optional"`
	Env        map[string]string `hcl:"env,optional"`
	GlobalOpts *ResticGlobalOpts `hcl:"options,block"`
}

// ExecutableTask is a task to be run before or after backup/retore
type ExecutableTask interface {
	RunBackup(cfg TaskConfig) error
	RunRestore(cfg TaskConfig) error
	Name() string
}

// JobTaskScript is a sript to be executed as part of a job task
type JobTaskScript struct {
	OnBackup   string `hcl:"on_backup,optional"`
	OnRestore  string `hcl:"on_restore,optional"`
	FromJobDir bool   `hcl:"from_job_dir,optional"`
	env        map[string]string
	name       string
}

// RunBackup runs script on backup
func (t JobTaskScript) RunBackup(cfg TaskConfig) error {
	env := MergeEnv(cfg.Env, t.env)
	if env == nil {
		env = map[string]string{}
	}

	env["RESTIC_JOB_DIR"] = cfg.JobDir

	cwd := ""
	if t.FromJobDir {
		cwd = cfg.JobDir
	}

	if err := RunShell(t.OnBackup, cwd, env, cfg.Logger); err != nil {
		return fmt.Errorf("failed running task script %s: %w", t.Name(), err)
	}

	return nil
}

// RunRestore script on restore
func (t JobTaskScript) RunRestore(cfg TaskConfig) error {
	env := MergeEnv(cfg.Env, t.env)
	if env == nil {
		env = map[string]string{}
	}

	env["RESTIC_JOB_DIR"] = cfg.JobDir

	cwd := ""
	if t.FromJobDir {
		cwd = cfg.JobDir
	}

	if err := RunShell(t.OnRestore, cwd, env, cfg.Logger); err != nil {
		return fmt.Errorf("failed running task script %s: %w", t.Name(), err)
	}

	return nil
}

func (t JobTaskScript) Name() string {
	return t.name
}

func (t *JobTaskScript) SetName(name string) {
	t.name = name
}

// JobTaskMySQL is a sqlite backup task that performs required pre and post tasks
type JobTaskMySQL struct {
	Name     string `hcl:"name,label"`
	Hostname string `hcl:"hostname,optional"`
	Database string `hcl:"database,optional"`
	Username string `hcl:"username,optional"`
	Password string `hcl:"password,optional"`
}

func (t JobTaskMySQL) GetPreTask() ExecutableTask {
	return JobTaskScript{
		name: t.Name,
		OnBackup: fmt.Sprintf(
			"mysqldump -h '%s' -u '%s' -p '%s' '%s' > './%s.sql'",
			t.Hostname,
			t.Username,
			t.Password,
			t.Database,
			t.Name,
		),
		FromJobDir: true,
	}
}

func (t JobTaskMySQL) GetPostTask() ExecutableTask {
	return JobTaskScript{
		name: t.Name,
		OnRestore: fmt.Sprintf(
			"mysql -h '%s' -u '%s' -p '%s' '%s' << './%s.sql'",
			t.Hostname,
			t.Username,
			t.Password,
			t.Database,
			t.Name,
		),
		FromJobDir: true,
	}
}

// JobTaskSqlite is a sqlite backup task that performs required pre and post tasks
type JobTaskSqlite struct {
	Name string `hcl:"name,label"`
	Path string `hcl:"path"`
}

func (t JobTaskSqlite) GetPreTask() ExecutableTask {
	return JobTaskScript{
		name: t.Name,
		OnBackup: fmt.Sprintf(
			"sqlite3 %s '.backup $RESTIC_JOB_DIR/%s.bak'",
			t.Path, t.Name,
		),
	}
}

func (t JobTaskSqlite) GetPostTask() ExecutableTask {
	return JobTaskScript{
		name:      t.Name,
		OnRestore: fmt.Sprintf("cp '$RESTIC_JOB_DIR/%s.bak' '%s'", t.Name, t.Path),
	}
}

type BackupFilesTask struct {
	Files       []string     `hcl:"files"`
	BackupOpts  *BackupOpts  `hcl:"backup_opts,block"`
	RestoreOpts *RestoreOpts `hcl:"restore_opts,block"`
	name        string
}

func (t BackupFilesTask) RunBackup(cfg TaskConfig) error {
	if err := cfg.Restic.Backup(t.Files, t.BackupOpts); err != nil {
		err = fmt.Errorf("failed backing up files: %w", err)
		cfg.Logger.Fatal(err)

		return err
	}

	return nil
}

func (t BackupFilesTask) RunRestore(cfg TaskConfig) error {
	if err := cfg.Restic.Restore("latest", t.RestoreOpts); err != nil {
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

// JobTask represents a single task within a backup job
type JobTask struct {
	Name    string           `hcl:"name,label"`
	Scripts []JobTaskScript  `hcl:"script,block"`
	Backup  *BackupFilesTask `hcl:"backup,block"`
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
// and restore job
type Job struct {
	Name     string       `hcl:"name,label"`
	Schedule string       `hcl:"schedule"`
	Config   ResticConfig `hcl:"config,block"`
	Tasks    []JobTask    `hcl:"task,block"`
	Validate bool         `hcl:"validate,optional"`
	Forget   *ForgetOpts  `hcl:"forget,block"`

	// Meta Tasks
	MySql  []JobTaskMySQL  `hcl:"mysql,block"`
	Sqlite []JobTaskSqlite `hcl:"sqlite,block"`
}

func (j Job) AllTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	// Pre tasks
	for _, mysql := range j.MySql {
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
	for _, mysql := range j.MySql {
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

func (j Job) RunTasks() error {
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
		}

		if err := exTask.RunBackup(taskCfg); err != nil {
			return fmt.Errorf("failed running job %s: %w", j.Name, err)
		}
	}

	if j.Forget != nil {
		restic.Forget(j.Forget)
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
	}
}

type Config struct {
	Jobs []Job `hcl:"job,block"`
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
