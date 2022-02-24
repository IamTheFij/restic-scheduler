package main_test

import (
	"bytes"
	"errors"
	"log"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
)

func TestResticConfigValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		config      main.ResticConfig
		expectedErr error
	}{
		{
			name:        "missing passphrase",
			expectedErr: main.ErrMutuallyExclusive,
			config:      main.ResticConfig{}, // nolint:exhaustivestruct
		},
		{
			name:        "passphrase no file",
			expectedErr: nil,
			// nolint:exhaustivestruct
			config: main.ResticConfig{
				Passphrase: "shh",
			},
		},
		{
			name:        "file no passphrase",
			expectedErr: nil,
			// nolint:exhaustivestruct
			config: main.ResticConfig{
				GlobalOpts: &main.ResticGlobalOpts{
					PasswordFile: "file",
				},
			},
		},
		{
			name:        "file and passphrase",
			expectedErr: main.ErrMutuallyExclusive,
			// nolint:exhaustivestruct
			config: main.ResticConfig{
				Passphrase: "shh",
				GlobalOpts: &main.ResticGlobalOpts{
					PasswordFile: "file",
				},
			},
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			actual := testCase.config.Validate()

			if !errors.Is(actual, testCase.expectedErr) {
				t.Errorf("expected error to wrap %v but found %v", testCase.expectedErr, actual)
			}
		})
	}
}

func NewBufferedLogger(prefix string) (*bytes.Buffer, *log.Logger) {
	outputBuffer := bytes.Buffer{}
	logger := log.New(&outputBuffer, prefix, 0)

	return &outputBuffer, logger
}

func TestJobTaskScript(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		script         main.JobTaskScript
		config         main.TaskConfig
		expectedErr    error
		expectedOutput string
	}{
		{
			name: "simple",
			config: main.TaskConfig{
				JobDir: "./test",
				Env:    nil,
				Logger: nil,
				Restic: nil,
			},
			script: main.JobTaskScript{
				OnBackup:   "echo yass",
				OnRestore:  "echo yass",
				FromJobDir: false,
			},
			expectedErr:    nil,
			expectedOutput: "t yass\nt \n",
		},
		{
			name: "check job dir",
			config: main.TaskConfig{
				JobDir: "./test",
				Env:    nil,
				Logger: nil,
				Restic: nil,
			},
			script: main.JobTaskScript{
				OnBackup:   "echo $RESTIC_JOB_DIR",
				OnRestore:  "echo $RESTIC_JOB_DIR",
				FromJobDir: false,
			},
			expectedErr:    nil,
			expectedOutput: "t ./test\nt \n",
		},
		{
			name: "check from job dir",
			config: main.TaskConfig{
				JobDir: "./test",
				Env:    nil,
				Logger: nil,
				Restic: nil,
			},
			script: main.JobTaskScript{
				OnBackup:   "basename `pwd`",
				OnRestore:  "basename `pwd`",
				FromJobDir: true,
			},
			expectedErr:    nil,
			expectedOutput: "t test\nt \n",
		},
		{
			name: "check env",
			config: main.TaskConfig{
				JobDir: "./test",
				Env:    map[string]string{"TEST": "OK"},
				Logger: nil,
				Restic: nil,
			},
			script: main.JobTaskScript{
				OnBackup:   "echo $TEST",
				OnRestore:  "echo $TEST",
				FromJobDir: false,
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

func TestJobTaskMySQL(t *testing.T) {
	t.Parallel()

	type TaskGenerator interface {
		Validate() error
		GetPreTask() main.ExecutableTask
		GetPostTask() main.ExecutableTask
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
			// nolint:exhaustivestruct
			task:          main.JobTaskMySQL{Name: "simple"},
			validationErr: nil,
			preBackup:     "mysqldump --result-file './simple.sql'",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "mysql < './simple.sql'",
		},
		{
			name: "mysql invalid name",
			// nolint:exhaustivestruct
			task:          main.JobTaskMySQL{Name: "it's invalid;"},
			validationErr: main.ErrInvalidConfigValue,
			preBackup:     "",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "",
		},
		{
			name: "mysql tables no database",
			// nolint:exhaustivestruct
			task: main.JobTaskMySQL{
				Name:   "name",
				Tables: []string{"table1", "table2"},
			},
			validationErr: main.ErrMissingField,
			preBackup:     "",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "",
		},
		{
			name: "mysql all options",
			task: main.JobTaskMySQL{
				Name:     "simple",
				Hostname: "host",
				Username: "user",
				Password: "pass",
				Database: "db",
				Tables:   []string{"table1", "table2"},
			},
			validationErr: nil,
			preBackup:     "mysqldump --result-file './simple.sql' --host host --user user --password pass db table1 table2",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "mysql --host host --user user --password pass < './simple.sql'",
		},
		// Sqlite
		{
			name: "sqlite simple",

			task:          main.JobTaskSqlite{Name: "simple", Path: "database.db"},
			validationErr: nil,
			preBackup:     "sqlite3 'database.db' '.backup $RESTIC_JOB_DIR/simple.db.bak'",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "cp '$RESTIC_JOB_DIR/simple.db.bak' 'database.db'",
		},
		{
			name: "sqlite invalid name",

			task:          main.JobTaskSqlite{Name: "it's invalid;", Path: "database.db"},
			validationErr: main.ErrInvalidConfigValue,
			preBackup:     "",
			postBackup:    "",
			preRestore:    "",
			postRestore:   "",
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

			if preTask, ok := testCase.task.GetPreTask().(main.JobTaskScript); ok {
				AssertEqual(t, "incorrect pre-backup", testCase.preBackup, preTask.OnBackup)
				AssertEqual(t, "incorrect pre-restore", testCase.preRestore, preTask.OnRestore)
			} else {
				t.Error("pre task was not a JobTaskScript")
			}

			if postTask, ok := testCase.task.GetPostTask().(main.JobTaskScript); ok {
				AssertEqual(t, "incorrect post-backup", testCase.postBackup, postTask.OnBackup)
				AssertEqual(t, "incorrect post-restore", testCase.postRestore, postTask.OnRestore)
			} else {
				t.Error("post task was not a JobTaskScript")
			}
		})
	}
}
