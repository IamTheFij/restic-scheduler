package main_test

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
)

const MinCoverage = 0.5

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

	validJob := main.Job{
		Name:     "Valid job",
		Schedule: "@daily",
		Config:   ValidResticConfig(),
		Tasks:    []main.JobTask{},
		Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
		Forget:   nil,
		MySQL:    []main.JobTaskMySQL{},
		Postgres: []main.JobTaskPostgres{},
		Sqlite:   []main.JobTaskSqlite{},
	}

	cases := []struct {
		name          string
		jobs          []main.Job
		names         []string
		expected      []main.Job
		expectedError error
	}{
		{
			name:          "Found job",
			jobs:          []main.Job{validJob},
			names:         []string{"Valid job"},
			expected:      []main.Job{validJob},
			expectedError: nil,
		},
		{
			name:          "Run all",
			jobs:          []main.Job{validJob},
			names:         []string{"all"},
			expected:      []main.Job{validJob},
			expectedError: nil,
		},
		{
			name:          "Extra, missing job",
			jobs:          []main.Job{validJob},
			names:         []string{"Valid job", "Not Found"},
			expected:      []main.Job{validJob},
			expectedError: main.ErrJobNotFound,
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name+" backup", func(t *testing.T) {
			t.Parallel()

			jobs, err := main.FilterJobs(testCase.jobs, testCase.names)
			if !reflect.DeepEqual(jobs, testCase.expected) {
				t.Errorf("expected %v but found %v", testCase.expected, jobs)
			}

			if !errors.Is(err, testCase.expectedError) {
				t.Errorf("expected %v but found %v", testCase.expectedError, err)
			}
		})
	}
}
