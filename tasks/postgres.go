package tasks

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"git.iamthefij.com/iamthefij/restic-scheduler/utils"
)

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

// Paths returns all paths to be backed up from this task.
func (t JobTaskPostgres) Paths() []string {
	return []string{t.DumpToPath}
}

// Validate ensures that this tasks configuration is valid.
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

// GetPreTask returns an ExecutableTask that should be run before backup.
func (t JobTaskPostgres) GetPreTask() ExecutableTask {
	command := []string{"pg_dump"}
	if t.Database == "" {
		command = []string{"pg_dumpall"}
	}

	command = append(command, "--file", t.DumpToPath)
	command = utils.MaybeAddArgString(command, "--host", t.Hostname)
	command = utils.MaybeAddArgInt(command, "--port", t.Port)
	command = utils.MaybeAddArgString(command, "--username", t.Username)
	command = utils.MaybeAddArgBool(command, "--no-tablespaces", t.NoTablespaces)
	command = utils.MaybeAddArgBool(command, "--clean", t.Clean)
	command = utils.MaybeAddArgBool(command, "--create", t.Create)
	command = utils.MaybeAddArgsList(command, "--table", t.Tables)

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

// GetPostTask returns an ExecutableTask that should be run after backup.
func (t JobTaskPostgres) GetPostTask() ExecutableTask {
	command := []string{"psql"}

	command = utils.MaybeAddArgString(command, "--host", t.Hostname)
	command = utils.MaybeAddArgInt(command, "--port", t.Port)
	command = utils.MaybeAddArgString(command, "--username", t.Username)

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
