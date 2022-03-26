package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
)

type TaskConfig struct {
	BackupPaths []string
	Env         map[string]string
	Logger      *log.Logger
	Restic      *Restic
}

// ExecutableTask is a task to be run before or after backup/retore.
type ExecutableTask interface {
	RunBackup(cfg TaskConfig) error
	RunRestore(cfg TaskConfig) error
	Name() string
}

// JobTaskScript is a sript to be executed as part of a job task.
type JobTaskScript struct {
	OnBackup  string `hcl:"on_backup,optional"`
	OnRestore string `hcl:"on_restore,optional"`
	Cwd       string `hcl:"cwd,optional"`
	env       map[string]string
	name      string
}

func (t JobTaskScript) run(script string, cfg TaskConfig) error {
	if script == "" {
		return nil
	}

	env := MergeEnvMap(cfg.Env, t.env)
	if env == nil {
		env = map[string]string{}
	}

	if err := RunShell(script, t.Cwd, env, cfg.Logger); err != nil {
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
	Name       string   `hcl:"name,label"`
	Hostname   string   `hcl:"hostname,optional"`
	Database   string   `hcl:"database,optional"`
	Username   string   `hcl:"username,optional"`
	Password   string   `hcl:"password,optional"`
	Tables     []string `hcl:"tables,optional"`
	DumpToPath string   `hcl:"dump_to"`
}

func (t JobTaskMySQL) Paths() []string {
	return []string{t.DumpToPath}
}

func (t JobTaskMySQL) Validate() error {
	if s, err := os.Stat(t.DumpToPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("Could not stat dump file path: %w", err)
		}
	} else if s.Mode().IsDir() {
		return fmt.Errorf("dump_to cannot be a directory: %w", ErrInvalidConfigValue)
	}

	if len(t.Tables) > 0 && t.Database == "" {
		return fmt.Errorf(
			"mysql task %s is invalid. Must specify a database to use tables: %w",
			t.Name,
			ErrMissingField,
		)
	}

	return nil
}

func (t JobTaskMySQL) GetPreTask() ExecutableTask {
	command := []string{"mysqldump", "--result-file", t.DumpToPath}

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
		name:      t.Name,
		env:       nil,
		Cwd:       ".",
		OnBackup:  strings.Join(command, " "),
		OnRestore: "",
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

	command = append(command, "<", t.DumpToPath)

	return JobTaskScript{
		name:      t.Name,
		env:       nil,
		Cwd:       ".",
		OnBackup:  "",
		OnRestore: strings.Join(command, " "),
	}
}

// JobTaskSqlite is a sqlite backup task that performs required pre and post tasks.
type JobTaskSqlite struct {
	Name       string `hcl:"name,label"`
	Path       string `hcl:"path"`
	DumpToPath string `hcl:"dump_to"`
}

func (t JobTaskSqlite) Paths() []string {
	return []string{t.DumpToPath}
}

func (t JobTaskSqlite) Validate() error {
	if s, err := os.Stat(t.DumpToPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("Could not stat dump file path: %w", err)
		}
	} else if s.Mode().IsDir() {
		return fmt.Errorf("dump_to cannot be a directory: %w", ErrInvalidConfigValue)
	}

	return nil
}

func (t JobTaskSqlite) GetPreTask() ExecutableTask {
	return JobTaskScript{
		name:      t.Name,
		env:       nil,
		Cwd:       ".",
		OnBackup:  fmt.Sprintf("sqlite3 '%s' '.backup %s'", t.Path, t.DumpToPath),
		OnRestore: "",
	}
}

func (t JobTaskSqlite) GetPostTask() ExecutableTask {
	return JobTaskScript{
		name:      t.Name,
		env:       nil,
		Cwd:       ".",
		OnBackup:  "",
		OnRestore: fmt.Sprintf("cp '%s' '%s'", t.DumpToPath, t.Path),
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

	if err := cfg.Restic.Backup(cfg.BackupPaths, *t.BackupOpts); err != nil {
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

	// TODO: Make the snapshot configurable
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
	Name        string          `hcl:"name,label"`
	PreScripts  []JobTaskScript `hcl:"pre_script,block"`
	PostScripts []JobTaskScript `hcl:"post_script,block"`
}

func (t JobTask) Validate() error {
	return nil
}

func (t JobTask) GetPreTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	for _, exTask := range t.PreScripts {
		exTask.SetName(t.Name)
		allTasks = append(allTasks, exTask)
	}

	return allTasks
}

func (t JobTask) GetPostTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	for _, exTask := range t.PostScripts {
		exTask.SetName(t.Name)
		allTasks = append(allTasks, exTask)
	}

	return allTasks
}
