package tasks

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"git.iamthefij.com/iamthefij/restic-scheduler/utils"
)

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
	SkipSSL       bool     `hcl:"skip_ssl,optional"`
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

// Paths returns all paths to be backed up from this task.
func (t JobTaskMySQL) Paths() []string {
	return []string{t.DumpToPath}
}

// Validate ensures that this tasks configuration is valid.
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

// GetPreTask returns an ExecutableTask that should be run before backup.
func (t JobTaskMySQL) GetPreTask() ExecutableTask {
	command := []string{t.mysqldumpCmd(), "--result-file", t.DumpToPath}

	command = utils.MaybeAddArgBool(command, "--skip-ssl", t.SkipSSL)
	command = utils.MaybeAddArgString(command, "--host", t.Hostname)
	command = utils.MaybeAddArgInt(command, "--port", t.Port)
	command = utils.MaybeAddArgString(command, "--user", t.Username)
	command = utils.MaybeAddArgBool(command, "--no-tablespaces", t.NoTablespaces)

	if t.Password != "" {
		command = append(command, fmt.Sprintf("--password=%s", t.Password))
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

// GetPostTask returns an ExecutableTask that should be run after backup.
func (t JobTaskMySQL) GetPostTask() ExecutableTask {
	command := []string{t.mysqlCommand()}

	command = utils.MaybeAddArgBool(command, "--skip-ssl", t.SkipSSL)
	command = utils.MaybeAddArgString(command, "--host", t.Hostname)
	command = utils.MaybeAddArgInt(command, "--port", t.Port)
	command = utils.MaybeAddArgString(command, "--user", t.Username)

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
