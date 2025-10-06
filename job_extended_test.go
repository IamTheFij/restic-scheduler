package main_test

import (
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
	"github.com/stretchr/testify/assert"
)

func TestAllTasks(t *testing.T) {
	t.Parallel()

	// Create a job with multiple task types
	job := main.Job{
		Name:     "TestJob",
		Schedule: "@daily",
		Config:   ValidResticConfig(),
		Tasks: []main.JobTask{
			{
				Name: "test-task",
			},
		},
		Backup: main.BackupFilesTask{Paths: []string{"/test"}},
		MySQL: []main.JobTaskMySQL{
			{
				Name:       "test-mysql",
				Hostname:   "localhost",
				DumpToPath: "/tmp/mysql",
			},
		},
		Postgres: []main.JobTaskPostgres{
			{
				Name:       "test-postgres",
				Hostname:   "localhost",
				DumpToPath: "/tmp/postgres",
			},
		},
		Sqlite: []main.JobTaskSqlite{
			{
				Name:       "test-sqlite",
				Path:       "/path/to/db.sqlite",
				DumpToPath: "/tmp/sqlite",
			},
		},
	}

	tasks := job.AllTasks()

	// We should have at least 5 tasks:
	// - MySQL pre task
	// - Postgres pre task
	// - Sqlite pre task
	// - Task pre task
	// - Backup task
	// - Task post task (since RunAfter is false)
	// - MySQL post task
	// - Postgres post task
	// - Sqlite post task
	assert.GreaterOrEqual(t, len(tasks), 5, "Should have at least 5 tasks")

	// Make sure the backup task is included in the list
	var foundBackup bool

	for _, task := range tasks {
		if bt, ok := task.(main.BackupFilesTask); ok && len(bt.Paths) > 0 {
			foundBackup = true
			break
		}
	}

	assert.True(t, foundBackup, "Backup task should be included in AllTasks")
}

func TestBackupPaths(t *testing.T) {
	t.Parallel()

	job := main.Job{
		Name:     "TestJob",
		Schedule: "@daily",
		Config:   ValidResticConfig(),
		Backup:   main.BackupFilesTask{Paths: []string{"/path1", "/path2"}},
		MySQL: []main.JobTaskMySQL{
			{
				Name:       "test-mysql",
				Hostname:   "localhost",
				DumpToPath: "/tmp/mysql",
			},
		},
		Postgres: []main.JobTaskPostgres{
			{
				Name:       "test-postgres",
				Hostname:   "localhost",
				DumpToPath: "/tmp/postgres",
			},
		},
		Sqlite: []main.JobTaskSqlite{
			{
				Name:       "test-sqlite",
				Path:       "/path/to/db.sqlite",
				DumpToPath: "/tmp/sqlite",
			},
		},
	}

	paths := job.BackupPaths()

	// Should include both the backup paths and the database dump paths
	expectedPaths := []string{
		"/path1",
		"/path2",
		"/tmp/mysql",
		"/tmp/postgres",
		"/tmp/sqlite",
	}

	assert.ElementsMatch(t, expectedPaths, paths)
}

func TestLogger(t *testing.T) {
	t.Parallel()

	job := main.Job{
		Name:     "TestLoggerJob",
		Schedule: "@daily",
		Config:   ValidResticConfig(),
	}

	logger := job.Logger()
	assert.NotNil(t, logger, "Logger should not be nil")
}

func TestNewRestic(t *testing.T) {
	t.Parallel()

	resticCfg := ValidResticConfig()
	resticCfg.Repo = "./test-repo"
	resticCfg.Passphrase = "test-passphrase"
	resticCfg.Env = map[string]string{"TEST_ENV": "value"}

	job := main.Job{
		Name:     "TestResticJob",
		Schedule: "@daily",
		Config:   resticCfg,
	}

	restic := job.NewRestic()

	assert.NotNil(t, restic, "Restic should not be nil")
	assert.Equal(t, resticCfg.Repo, restic.Repo)
	assert.Equal(t, resticCfg.Passphrase, restic.Passphrase)
	assert.Equal(t, resticCfg.Env, restic.Env)
}
