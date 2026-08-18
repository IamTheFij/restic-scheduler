package main_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
	"git.iamthefij.com/iamthefij/restic-scheduler/tasks"
	"github.com/stretchr/testify/assert"
)

const MinCoverage = 0.0

func ValidResticConfig() *tasks.ResticConfig {
	return &tasks.ResticConfig{
		Passphrase: "shh",
		Repo:       "./data",
		Env:        nil,
		GlobalOpts: nil,
	}
}

func TestMain(m *testing.M) {
	testResult := m.Run()

	if testResult == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < MinCoverage {
			fmt.Printf("WARNING: Tests passed but coverage failed at %0.2f and minimum to pass is %0.2f\n", c, MinCoverage)

			testResult = 0
		}
	}

	os.Exit(testResult)
}

func TestReadJobs(t *testing.T) {
	t.Parallel()

	jobs, err := main.ReadJobs([]string{"./test/sample.hcl"})
	if err != nil {
		t.Errorf("Unexpected error reading jobs: %v", err)
	}

	if len(jobs) == 0 {
		t.Error("Expected read jobs but found none")
	}
}

func TestRunJobs(t *testing.T) {
	t.Parallel()

	validJob := tasks.Job{
		Name:     "Valid job",
		Schedule: "@daily",
		Config:   ValidResticConfig(),
		Tasks:    []tasks.JobTask{},
		Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
		Forget:   nil,
		MySQL:    []tasks.JobTaskMySQL{},
		Postgres: []tasks.JobTaskPostgres{},
		Sqlite:   []tasks.JobTaskSqlite{},
	}

	cases := []struct {
		name          string
		jobs          []tasks.Job
		names         []string
		expected      []tasks.Job
		expectedError error
	}{
		{
			name:          "Found job",
			jobs:          []tasks.Job{validJob},
			names:         []string{"Valid job"},
			expected:      []tasks.Job{validJob},
			expectedError: nil,
		},
		{
			name:          "Run all",
			jobs:          []tasks.Job{validJob},
			names:         []string{"all"},
			expected:      []tasks.Job{validJob},
			expectedError: nil,
		},
		{
			name:          "Extra, missing job",
			jobs:          []tasks.Job{validJob},
			names:         []string{"Valid job", "Not Found"},
			expected:      []tasks.Job{validJob},
			expectedError: main.ErrJobNotFound,
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name+" backup", func(t *testing.T) {
			t.Parallel()

			jobs, err := main.FilterJobs(testCase.jobs, testCase.names)
			_ = assert.Equal(t, jobs, testCase.expected)

			if !errors.Is(err, testCase.expectedError) {
				t.Errorf("expected %v but found %v", testCase.expectedError, err)
			}
		})
	}
}
