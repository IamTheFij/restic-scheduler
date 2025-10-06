package main_test

import (
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
	"github.com/stretchr/testify/assert"
)

func TestJobTaskScriptName(t *testing.T) {
	t.Parallel()

	// Create a script task
	script := &main.JobTaskScript{}

	// Initial name should be empty
	assert.Empty(t, script.Name())

	// Test SetName
	script.SetName("test-script")
	assert.Equal(t, "test-script", script.Name())
}

func TestMySQLTaskPaths(t *testing.T) {
	t.Parallel()

	mysqlTask := main.JobTaskMySQL{
		Name:       "test-mysql",
		DumpToPath: "/path/to/dump.sql",
	}

	paths := mysqlTask.Paths()
	assert.Equal(t, []string{"/path/to/dump.sql"}, paths)
}

func TestPostgresTaskPaths(t *testing.T) {
	t.Parallel()

	pgTask := main.JobTaskPostgres{
		Name:       "test-postgres",
		DumpToPath: "/path/to/dump.sql",
	}

	paths := pgTask.Paths()
	assert.Equal(t, []string{"/path/to/dump.sql"}, paths)
}

func TestSqliteTaskPaths(t *testing.T) {
	t.Parallel()

	sqliteTask := main.JobTaskSqlite{
		Name:       "test-sqlite",
		DumpToPath: "/path/to/dump.sql",
	}

	paths := sqliteTask.Paths()
	assert.Equal(t, []string{"/path/to/dump.sql"}, paths)
}

func TestMySQLGetPreTask(t *testing.T) {
	t.Parallel()

	mysqlTask := main.JobTaskMySQL{
		Name:       "test-mysql",
		Hostname:   "localhost",
		Port:       3306,
		Username:   "user",
		Password:   "pass",
		Database:   "testdb",
		Tables:     []string{"table1", "table2"},
		DumpToPath: "/path/to/dump.sql",
	}

	preTask := mysqlTask.GetPreTask()
	assert.NotNil(t, preTask)
	assert.Equal(t, "test-mysql", preTask.Name())
}

func TestMySQLGetPostTask(t *testing.T) {
	t.Parallel()

	mysqlTask := main.JobTaskMySQL{
		Name:       "test-mysql",
		Hostname:   "localhost",
		Port:       3306,
		Username:   "user",
		Password:   "pass",
		Database:   "testdb",
		DumpToPath: "/path/to/dump.sql",
	}

	postTask := mysqlTask.GetPostTask()
	assert.NotNil(t, postTask)
	assert.Equal(t, "test-mysql", postTask.Name())
}
