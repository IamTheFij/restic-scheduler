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
	BackupPaths     []string
	Env             map[string]string
	Logger          *log.Logger
	Restic          *Restic
	RestoreSnapshot string
}

// ExecutableTask is a task to be run before or after backup/retore.
type ExecutableTask interface {
	RunBackup(cfg TaskConfig) error
	RunRestore(cfg TaskConfig) error
	Name() string
}

// JobTaskScript is a sript to be executed as part of a job task.
type JobTaskScript struct {
	OnBackup  string            `hcl:"on_backup,optional"`
	OnRestore string            `hcl:"on_restore,optional"`
	Cwd       string            `hcl:"cwd,optional"`
	Env       map[string]string `hcl:"env,optional"`
	name      string
}

func (t JobTaskScript) run(script string, cfg TaskConfig) error {
	if script == "" {
		return nil
	}

	env := MergeEnvMap(cfg.Env, t.Env)
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

// JobTaskMySQL is a MySQL backup task that performs required pre and post tasks.
type JobTaskMySQL struct {
	Port          int      `hcl:"port,optional"`
	Name          string   `hcl:"name,label"`
	Hostname      string   `hcl:"hostname,optional"`
	Database      string   `hcl:"database,optional"`
	Username      string   `hcl:"username,optional"`
	Password      string   `hcl:"password,optional"`
	Tables        []string `hcl:"tables,optional"`
	NoTablespaces bool     `hcl:"no_tablespaces,optional"`
	DumpToPath    string   `hcl:"dump_to"`
	UseMariaDB    bool     `hcl:"use_mariadb,optional"`
}

func (t JobTaskMySQL) mysqlCommand() string {
	if t.UseMariaDB {
		return "mariadb"
	}

	return "mysql"
}

func (t JobTaskMySQL) mysqldumpCmd() string {
	if t.UseMariaDB {
		return "mariadb-dump"
	}

	return "mysqldump"
}

func (t JobTaskMySQL) Paths() []string {
	return []string{t.DumpToPath}
}

func (t JobTaskMySQL) Validate() error {
	if t.DumpToPath == "" {
		return fmt.Errorf("task %s is missing dump_to path: %w", t.Name, ErrMissingField)
	}

	if stat, err := os.Stat(t.DumpToPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf(
				"task %s: invalid dump_to: could not stat path: %s: %w",
				t.Name,
				t.DumpToPath,
				ErrInvalidConfigValue,
			)
		}
	} else if stat.Mode().IsDir() {
		return fmt.Errorf("task %s: dump_to cannot be a directory: %w", t.Name, ErrInvalidConfigValue)
	}

	if len(t.Tables) > 0 && t.Database == "" {
		return fmt.Errorf(
			"task %s is invalid. Must specify a database to use tables: %w",
			t.Name,
			ErrMissingField,
		)
	}

	return nil
}

func (t JobTaskMySQL) GetPreTask() ExecutableTask {
	command := []string{t.mysqldumpCmd(), "--result-file", t.DumpToPath}

	if t.Hostname != "" {
		command = append(command, "--host", t.Hostname)
	}

	if t.Port != 0 {
		command = append(command, "--port", fmt.Sprintf("%d", t.Port))
	}

	if t.Username != "" {
		command = append(command, "--user", t.Username)
	}

	if t.Password != "" {
		command = append(command, fmt.Sprintf("--password=%s", t.Password))
	}

	if t.NoTablespaces {
		command = append(command, "--no-tablespaces")
	}

	if t.Database != "" {
		command = append(command, t.Database)
	} else {
		command = append(command, "--all-databases")
	}

	command = append(command, t.Tables...)

	return JobTaskScript{
		name:      t.Name,
		Env:       nil,
		Cwd:       ".",
		OnBackup:  strings.Join(command, " "),
		OnRestore: "",
	}
}

func (t JobTaskMySQL) GetPostTask() ExecutableTask {
	command := []string{t.mysqlCommand()}

	if t.Hostname != "" {
		command = append(command, "--host", t.Hostname)
	}

	if t.Port != 0 {
		command = append(command, "--port", fmt.Sprintf("%d", t.Port))
	}

	if t.Username != "" {
		command = append(command, "--user", t.Username)
	}

	if t.Password != "" {
		command = append(command, fmt.Sprintf("--password=%s", t.Password))
	}

	if t.Database != "" {
		command = append(command, t.Database)
	}

	command = append(command, "<", t.DumpToPath)

	return JobTaskScript{
		name:      t.Name,
		Env:       nil,
		Cwd:       ".",
		OnBackup:  "",
		OnRestore: strings.Join(command, " "),
	}
}

// JobTaskPostgres is a postgres backup task that performs required pre and post tasks.
type JobTaskPostgres struct {
	Port          int      `hcl:"port,optional"`
	Name          string   `hcl:"name,label"`
	Hostname      string   `hcl:"hostname,optional"`
	Database      string   `hcl:"database,optional"`
	Username      string   `hcl:"username,optional"`
	Password      string   `hcl:"password,optional"`
	Tables        []string `hcl:"tables,optional"`
	DumpToPath    string   `hcl:"dump_to"`
	NoTablespaces bool     `hcl:"no_tablespaces,optional"`
	Clean         bool     `hcl:"clean,optional"`
	Create        bool     `hcl:"create,optional"`
}

func (t JobTaskPostgres) Paths() []string {
	return []string{t.DumpToPath}
}

func (t JobTaskPostgres) Validate() error {
	if t.DumpToPath == "" {
		return fmt.Errorf("task %s is missing dump_to path: %w", t.Name, ErrMissingField)
	}

	if stat, err := os.Stat(t.DumpToPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf(
				"task %s: invalid dump_to: could not stat path: %s: %w",
				t.Name,
				t.DumpToPath,
				ErrInvalidConfigValue,
			)
		}
	} else if stat.Mode().IsDir() {
		return fmt.Errorf("task %s: dump_to cannot be a directory: %w", t.Name, ErrInvalidConfigValue)
	}

	if len(t.Tables) > 0 && t.Database == "" {
		return fmt.Errorf(
			"task %s is invalid. Must specify a database to use tables: %w",
			t.Name,
			ErrMissingField,
		)
	}

	return nil
}

//nolint:cyclop
func (t JobTaskPostgres) GetPreTask() ExecutableTask {
	command := []string{"pg_dump"}
	if t.Database == "" {
		command = []string{"pg_dumpall"}
	}

	command = append(command, "--file", t.DumpToPath)

	if t.Hostname != "" {
		command = append(command, "--host", t.Hostname)
	}

	if t.Port != 0 {
		command = append(command, "--port", fmt.Sprintf("%d", t.Port))
	}

	if t.Username != "" {
		command = append(command, "--username", t.Username)
	}

	if t.NoTablespaces {
		command = append(command, "--no-tablespaces")
	}

	if t.Clean {
		command = append(command, "--clean")
	}

	if t.Create {
		command = append(command, "--create")
	}

	for _, table := range t.Tables {
		command = append(command, "--table", table)
	}

	if t.Database != "" {
		command = append(command, t.Database)
	}

	env := map[string]string{}
	if t.Password != "" {
		env["PGPASSWORD"] = t.Password
	}

	return JobTaskScript{
		name:      t.Name,
		Env:       env,
		Cwd:       ".",
		OnBackup:  strings.Join(command, " "),
		OnRestore: "",
	}
}

func (t JobTaskPostgres) GetPostTask() ExecutableTask {
	command := []string{"psql"}

	if t.Hostname != "" {
		command = append(command, "--host", t.Hostname)
	}

	if t.Port != 0 {
		command = append(command, "--port", fmt.Sprintf("%d", t.Port))
	}

	if t.Username != "" {
		command = append(command, "--username", t.Username)
	}

	if t.Database != "" {
		command = append(command, t.Database)
	}

	command = append(command, "<", t.DumpToPath)

	env := map[string]string{}
	if t.Password != "" {
		env["PGPASSWORD"] = t.Password
	}

	return JobTaskScript{
		name:      t.Name,
		Env:       env,
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
	if t.DumpToPath == "" {
		return fmt.Errorf("task %s is missing dump_to path: %w", t.Name, ErrMissingField)
	}

	if stat, err := os.Stat(t.DumpToPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf(
				"task %s: invalid dump_to: could not stat path: %s: %w",
				t.Name,
				t.DumpToPath,
				ErrInvalidConfigValue,
			)
		}
	} else if stat.Mode().IsDir() {
		return fmt.Errorf("task %s: dump_to cannot be a directory: %w", t.Name, ErrInvalidConfigValue)
	}

	return nil
}

func (t JobTaskSqlite) GetPreTask() ExecutableTask {
	return JobTaskScript{
		name:      t.Name,
		Env:       nil,
		Cwd:       ".",
		OnBackup:  fmt.Sprintf("sqlite3 '%s' '.backup %s'", t.Path, t.DumpToPath),
		OnRestore: "",
	}
}

func (t JobTaskSqlite) GetPostTask() ExecutableTask {
	return JobTaskScript{
		name:      t.Name,
		Env:       nil,
		Cwd:       ".",
		OnBackup:  "",
		OnRestore: fmt.Sprintf("cp '%s' '%s'", t.DumpToPath, t.Path),
	}
}

type BackupFilesTask struct {
	Paths       []string     `hcl:"paths"`
	BackupOpts  *BackupOpts  `hcl:"backup_opts,block"`
	RestoreOpts *RestoreOpts `hcl:"restore_opts,block"`
	name        string
}

func (t BackupFilesTask) RunBackup(cfg TaskConfig) error {
	if t.BackupOpts == nil {
		t.BackupOpts = &BackupOpts{} //nolint:exhaustruct
	}

	if err := cfg.Restic.Backup(cfg.BackupPaths, *t.BackupOpts); err != nil {
		err = fmt.Errorf("failed backing up paths: %w", err)
		cfg.Logger.Print(err)

		return err
	}

	return nil
}

func (t BackupFilesTask) RunRestore(cfg TaskConfig) error {
	if t.RestoreOpts == nil {
		t.RestoreOpts = &RestoreOpts{} //nolint:exhaustruct
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

func (t BackupFilesTask) Name() string {
	return t.name
}

func (t *BackupFilesTask) SetName(name string) {
	t.name = name
}

func (t *BackupFilesTask) Validate() error {
	if len(t.Paths) == 0 {
		return fmt.Errorf("backup config doesn't include any paths: %w", ErrInvalidConfigValue)
	}

	return nil
}

// JobTask represents a single task within a backup job.
type JobTask struct {
	Name        string            `hcl:"name,label"`
	PreScripts  []JobTaskScript   `hcl:"pre_script,block"`
	PostScripts []JobTaskScript   `hcl:"post_script,block"`
	MySQL       []JobTaskMySQL    `hcl:"mysql,block"`
	Postgres    []JobTaskPostgres `hcl:"postgres,block"`
	Sqlite      []JobTaskSqlite   `hcl:"sqlite,block"`
}

func (t JobTask) Validate() error {
	// NOTE: Might make task types mutually exclusive because order is confusing even if deterministic
	if t.Name == "" {
		return fmt.Errorf("task is missing a name: %w", ErrMissingField)
	}

	return nil
}

func (t JobTask) GetPreTasks() []ExecutableTask {
	allTasks := []ExecutableTask{}

	for _, task := range t.MySQL {
		allTasks = append(allTasks, task.GetPreTask())
	}

	for _, task := range t.Sqlite {
		allTasks = append(allTasks, task.GetPreTask())
	}

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

	for _, task := range t.MySQL {
		allTasks = append(allTasks, task.GetPostTask())
	}

	for _, task := range t.Sqlite {
		allTasks = append(allTasks, task.GetPostTask())
	}

	return allTasks
}
