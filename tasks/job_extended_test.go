package tasks_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"git.iamthefij.com/iamthefij/restic-scheduler/config"
	"git.iamthefij.com/iamthefij/restic-scheduler/tasks"
	"git.iamthefij.com/iamthefij/restic-scheduler/utils"
	"github.com/stretchr/testify/assert"
)

const (
	TestJobHCL = `
job "TestJob" {
	schedule = "* * * * *"

	config {
		repo = "%s"
		passphrase = "shhh"

		options {
			cache_dir = "%s"
		}
	}


	backup {
		paths = ["%s"]

		restore_opts {
			target = "%s"
		}
	}

	copy {
		target_repo = "%s"
		target_passphrase = "shhh"
	}

	forget {
		keep_last = 2
	}
}
`
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

func TestJobBackups(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skip integration test when running short tests")
	}

	dataDir := t.TempDir()
	repoDir := t.TempDir()
	cacheDir := t.TempDir()
	restoreTarget := t.TempDir()
	repoCopyDir := t.TempDir()

	dataFile := filepath.Join(dataDir, "test.txt")
	restoredDataFile := filepath.Join(restoreTarget, dataFile)

	mainJobHCL := fmt.Sprintf(TestJobHCL, repoDir, cacheDir, dataDir, restoreTarget, repoCopyDir)
	jobs, err := config.ParseConfig("test.hcl", []byte(mainJobHCL))
	utils.AssertEqualFail(t, "unexpected error parsing job hcl", nil, err)

	testJob := jobs[0]

	// Write test file to the data dir
	err = os.WriteFile(dataFile, []byte("testing"), 0o644)
	utils.AssertEqualFail(t, "unexpected error writing to test file", nil, err)

	// Run initial backup, should auto initialize repo
	err = testJob.RunBackup()
	utils.AssertEqualFail(t, "unexpected error running backups", nil, err)

	// Check snapshots
	r := testJob.NewRestic()
	snapshots, err := r.ReadSnapshots()
	utils.AssertEqualFail(t, "unexpected error reading snapshots", nil, err)
	utils.AssertEqual(t, "unexpected number of snapshots", 1, len(snapshots))

	// Backup again
	err = testJob.RunBackup()
	utils.AssertEqualFail(t, "unexpected error running backups", nil, err)

	// Check for second backup
	snapshots, err = r.ReadSnapshots()
	utils.AssertEqualFail(t, "unexpected error reading second snapshots", nil, err)
	utils.AssertEqual(t, "unexpected number of snapshots", 2, len(snapshots))

	// Run third backup to trigger a prune
	err = testJob.RunBackup()
	utils.AssertEqualFail(t, "unexpected error running backups", nil, err)

	// Check only 2 backups
	snapshots, err = r.ReadSnapshots()
	utils.AssertEqualFail(t, "unexpected error reading second snapshots", nil, err)
	utils.AssertEqual(t, "unexpected number of snapshots", 2, len(snapshots))

	// Restore files
	err = testJob.RunRestore("latest")
	utils.AssertEqualFail(t, "unexpected error restoring latest snapshot", nil, err)

	// Check restored values
	value, err := os.ReadFile(restoredDataFile)
	utils.AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	utils.AssertEqualFail(t, "incorrect value in test file", "testing", string(value))

	// Try to unlock the repo (repo shouldn't really be locked, but this should still run without error
	err = testJob.RunUnlock()
	utils.AssertEqualFail(t, "unexpected error unlocking repo", nil, err)

	// Clear out test job so we don't accidentally use it below for restore operations
	testJob = tasks.Job{}

	// Test restore from copy
	copyRepoHCL := fmt.Sprintf(TestJobHCL, repoCopyDir, cacheDir, dataDir, restoreTarget, repoDir)
	jobs, err = config.ParseConfig("test.hcl", []byte(copyRepoHCL))
	utils.AssertEqualFail(t, "unexpected error parsing copy repo hcl", nil, err)

	copyRepoJob := jobs[0]

	// Remove test restored file
	err = os.Remove(restoredDataFile)
	utils.AssertEqualFail(t, "unexpected error removing test data file", nil, err)

	// Restore files
	err = copyRepoJob.RunRestore("latest")
	utils.AssertEqualFail(t, "unexpected error restoring latest snapshot", nil, err)

	// Check restored values
	value, err = os.ReadFile(restoredDataFile)
	utils.AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	utils.AssertEqualFail(t, "incorrect value in test file", "testing", string(value))
}
