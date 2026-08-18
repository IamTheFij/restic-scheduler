package main_test

import (
	"testing"

	"git.iamthefij.com/iamthefij/restic-scheduler/tasks"
	"github.com/stretchr/testify/assert"
)

func TestAllTasks(t *testing.T) {
	t.Parallel()

	// Create a job with multiple task types
	job := tasks.Job{
		Name:     "TestJob",
		Schedule: "@daily",
		Config:   ValidResticConfig(),
		Tasks: []tasks.JobTask{
			{
				Name: "test-task",
				MySQL: []tasks.JobTaskMySQL{
					{
						Name:       "test-mysql",
						Hostname:   "localhost",
						DumpToPath: "/tmp/mysql",
					},
				},
				Postgres: []tasks.JobTaskPostgres{
					{
						Name:       "test-postgres",
						Hostname:   "localhost",
						DumpToPath: "/tmp/postgres",
					},
				},
				Sqlite: []tasks.JobTaskSqlite{
					{
						Name:       "test-sqlite",
						Path:       "/path/to/db.sqlite",
						DumpToPath: "/tmp/sqlite",
					},
				},
			},
		},
		Backup: tasks.BackupFilesTask{BackupPaths: []string{"/test"}},
	}

	allTasks := job.AllTasks()

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
	assert.GreaterOrEqual(t, len(allTasks), 5, "Should have at least 5 tasks")

	// Make sure the backup task is included in the list
	var foundBackup bool

	for _, task := range allTasks {
		if bt, ok := task.(tasks.BackupFilesTask); ok && len(bt.BackupPaths) > 0 {
			foundBackup = true
			break
		}
	}

	assert.True(t, foundBackup, "Backup task should be included in AllTasks")
}

func TestBackupPaths(t *testing.T) {
	t.Parallel()

	job := tasks.Job{
		Name:     "TestJob",
		Schedule: "@daily",
		Config:   ValidResticConfig(),
		Backup:   tasks.BackupFilesTask{BackupPaths: []string{"/path1", "/path2"}},
		Tasks: []tasks.JobTask{
			{
				MySQL: []tasks.JobTaskMySQL{
					{
						Name:       "test-mysql",
						Hostname:   "localhost",
						DumpToPath: "/tmp/mysql",
					},
				},
				Postgres: []tasks.JobTaskPostgres{
					{
						Name:       "test-postgres",
						Hostname:   "localhost",
						DumpToPath: "/tmp/postgres",
					},
				},
				Sqlite: []tasks.JobTaskSqlite{
					{
						Name:       "test-sqlite",
						Path:       "/path/to/db.sqlite",
						DumpToPath: "/tmp/sqlite",
					},
				},
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

	job := tasks.Job{
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

	job := tasks.Job{
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
