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
