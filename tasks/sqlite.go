package tasks

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// JobTaskSqlite is a sqlite backup task that performs required pre and post tasks.
type JobTaskSqlite struct {
	Name       string `hcl:"name,label"`
	Path       string `hcl:"path"`
	DumpToPath string `hcl:"dump_to"`
}

// Paths returns all paths to be backed up from this task.
func (t JobTaskSqlite) Paths() []string {
	return []string{t.DumpToPath}
}

// Validate ensures that this tasks configuration is valid.
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

// GetPreTask returns an ExecutableTask that should be run before backup.
func (t JobTaskSqlite) GetPreTask() ExecutableTask {
	return JobTaskScript{
		name:      t.Name,
		Env:       nil,
		Cwd:       ".",
		OnBackup:  fmt.Sprintf("sqlite3 '%s' '.backup %s'", t.Path, t.DumpToPath),
		OnRestore: "",
	}
}

// GetPostTask returns an ExecutableTask that should be run after backup.
func (t JobTaskSqlite) GetPostTask() ExecutableTask {
	return JobTaskScript{
		name:      t.Name,
		Env:       nil,
		Cwd:       ".",
		OnBackup:  "",
		OnRestore: fmt.Sprintf("cp '%s' '%s'", t.DumpToPath, t.Path),
	}
}
