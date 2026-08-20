package tasks_test

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"git.iamthefij.com/iamthefij/restic-scheduler/tasks"
	"git.iamthefij.com/iamthefij/restic-scheduler/utils"
)

func NewBufferedLogger(prefix string) (*bytes.Buffer, *log.Logger) {
	outputBuffer := bytes.Buffer{}
	logger := log.New(&outputBuffer, prefix, 0)

	return &outputBuffer, logger
}

func TestJobTaskScript(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		script         tasks.JobTaskScript
		config         tasks.TaskConfig
		expectedErr    error
		expectedOutput string
	}{
		{
			name: "simple",
			config: tasks.TaskConfig{
				BackupPaths: nil,
				Env:         nil,
				Logger:      nil,
				Restic:      nil,
			},
			script: tasks.JobTaskScript{
				Cwd:       "../test",
				OnBackup:  "echo yass",
				OnRestore: "echo yass",
			},
			expectedErr:    nil,
			expectedOutput: "t yass\nt \n",
		},
		{
			name: "check from job dir",
			config: tasks.TaskConfig{
				BackupPaths: nil,
				Env:         nil,
				Logger:      nil,
				Restic:      nil,
			},
			script: tasks.JobTaskScript{
				Cwd:       "../test",
				OnBackup:  "basename `pwd`",
				OnRestore: "basename `pwd`",
			},
			expectedErr:    nil,
			expectedOutput: "t test\nt \n",
		},
		{
			name: "check env",
			config: tasks.TaskConfig{
				BackupPaths: nil,
				Env:         map[string]string{"TEST": "OK"},
				Logger:      nil,
				Restic:      nil,
			},
			script: tasks.JobTaskScript{
				Cwd:       "../test",
				OnBackup:  "echo $TEST",
				OnRestore: "echo $TEST",
			},
			expectedErr:    nil,
			expectedOutput: "t OK\nt \n",
		},
	}

	for _, c := range cases {
		testCase := c

		buf, logger := NewBufferedLogger("t")
		testCase.config.Logger = logger

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := testCase.script.RunBackup(testCase.config)

			if !errors.Is(actual, testCase.expectedErr) {
				t.Errorf("expected error to wrap %v but found %v", testCase.expectedErr, actual)
			}

			output := buf.String()

			if testCase.expectedOutput != output {
				t.Errorf("Unexpected output. expected: %s actual: %s", testCase.expectedOutput, output)
			}
		})
	}
}

func TestJobTaskSql(t *testing.T) {
	t.Parallel()

	type TaskGenerator interface {
		Validate() error
		GetPreTask() tasks.ExecutableTask
		GetPostTask() tasks.ExecutableTask
	}

	cases := []struct {
		name          string
		task          TaskGenerator
		validationErr error
		preBackup     string
		postBackup    string
		preRestore    string
		postRestore   string
	}{
		{
			name: "mysql simple",
			//nolint:exhaustruct
			task: tasks.JobTaskMySQL{
				Name:       "simple",
				DumpToPath: "./simple.sql",
			},
			validationErr: nil,
			preBackup:     "mysqldump --result-file ./simple.sql --all-databases",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "mysql < ./simple.sql",
		},
		{
			name: "mariadb simple",
			//nolint:exhaustruct
			task: tasks.JobTaskMariaDB{
				//nolint:exhaustruct
				tasks.JobTaskMySQL{
					DumpToPath: "./simple.sql",
				},
				"simple",
			},
			validationErr: nil,
			preBackup:     "mariadb-dump --result-file ./simple.sql --all-databases",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "mariadb < ./simple.sql",
		},
		{
			name: "mysql using mariadb",
			//nolint:exhaustruct
			task: tasks.JobTaskMySQL{
				Name:       "simple",
				DumpToPath: "./simple.sql",
				UseMariaDB: true,
			},
			validationErr: nil,
			preBackup:     "mariadb-dump --result-file ./simple.sql --all-databases",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "mariadb < ./simple.sql",
		},
		{
			name: "mysql tables no database",
			//nolint:exhaustruct
			task: tasks.JobTaskMySQL{
				Name:       "name",
				Tables:     []string{"table1", "table2"},
				DumpToPath: "./simple.sql",
			},
			validationErr: tasks.ErrMissingField,
			preBackup:     "",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "",
		},
		{
			name: "mysql all options",
			task: tasks.JobTaskMySQL{
				Name:          "simple",
				Hostname:      "host",
				Port:          3306,
				Username:      "user",
				Password:      "pass",
				Database:      "db",
				NoTablespaces: true,
				Tables:        []string{"table1", "table2"},
				DumpToPath:    "./simple.sql",
			},
			validationErr: nil,
			preBackup: "mysqldump --result-file ./simple.sql --host host --port 3306" +
				" --user user --no-tablespaces --password=pass db table1 table2",
			postBackup:  "",
			preRestore:  "",
			postRestore: "mysql --host host --port 3306 --user user --password=pass db < ./simple.sql",
		},
		{
			name: "psql all",
			task: tasks.JobTaskPostgres{
				Name:          "simple",
				Hostname:      "host",
				Port:          6543,
				Username:      "user",
				Password:      "pass",
				Database:      "db",
				NoTablespaces: true,
				Create:        true,
				Clean:         true,
				Tables:        []string{"table1", "table2"},
				DumpToPath:    "./simple.sql",
			},
			validationErr: nil,
			preBackup: "pg_dump --file ./simple.sql --host host --port 6543 --username user --no-tablespaces" +
				" --clean --create --table table1 --table table2 db",
			postBackup:  "",
			preRestore:  "",
			postRestore: "psql --host host --port 6543 --username user db < ./simple.sql",
		},
		// Sqlite
		{
			name: "sqlite simple",

			task: tasks.JobTaskSqlite{
				Name:       "simple",
				Path:       "database.db",
				DumpToPath: "./simple.db.bak",
			},
			validationErr: nil,
			preBackup:     "sqlite3 'database.db' '.backup ./simple.db.bak'",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "cp './simple.db.bak' 'database.db'",
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			validateErr := testCase.task.Validate()
			if !errors.Is(validateErr, testCase.validationErr) {
				t.Errorf("unexpected validation result. expected: %v, actual: %v", testCase.validationErr, validateErr)
			}

			if validateErr != nil {
				return
			}

			if preTask, ok := testCase.task.GetPreTask().(tasks.JobTaskScript); ok {
				utils.AssertEqual(t, "incorrect pre-backup", testCase.preBackup, preTask.OnBackup)
				utils.AssertEqual(t, "incorrect pre-restore", testCase.preRestore, preTask.OnRestore)
			} else {
				t.Error("pre task was not a JobTaskScript")
			}

			if postTask, ok := testCase.task.GetPostTask().(tasks.JobTaskScript); ok {
				utils.AssertEqual(t, "incorrect post-backup", testCase.postBackup, postTask.OnBackup)
				utils.AssertEqual(t, "incorrect post-restore", testCase.postRestore, postTask.OnRestore)
			} else {
				t.Error("post task was not a JobTaskScript")
			}
		})
	}
}
