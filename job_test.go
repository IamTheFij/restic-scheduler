package main_test

import (
	"errors"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
)

func ValidResticConfig() *main.ResticConfig {
	return &main.ResticConfig{
		Passphrase: "shh",
		Repo:       "./data",
		Env:        nil,
		GlobalOpts: nil,
	}
}

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
			config:      main.ResticConfig{}, //nolint:exhaustruct
		},
		{
			name:        "passphrase no file",
			expectedErr: nil,
			//nolint:exhaustruct
			config: main.ResticConfig{
				Passphrase: "shh",
			},
		},
		{
			name:        "file no passphrase",
			expectedErr: nil,
			//nolint:exhaustruct
			config: main.ResticConfig{
				GlobalOpts: &main.ResticGlobalOpts{
					PasswordFile: "file",
				},
			},
		},
		{
			name:        "file and passphrase",
			expectedErr: main.ErrMutuallyExclusive,
			//nolint:exhaustruct
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

func TestJobValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		job         main.Job
		expectedErr error
	}{
		{
			name: "Valid job",
			job: main.Job{
				Name:     "Valid job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []main.JobTask{},
				Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []main.JobTaskMySQL{},
				Postgres: []main.JobTaskPostgres{},
				Sqlite:   []main.JobTaskSqlite{},
			},
			expectedErr: nil,
		},
		{
			name: "Invalid name",
			job: main.Job{
				Name:     "",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []main.JobTask{},
				Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []main.JobTaskMySQL{},
				Postgres: []main.JobTaskPostgres{},
				Sqlite:   []main.JobTaskSqlite{},
			},
			expectedErr: main.ErrMissingField,
		},
		{
			name: "Invalid schedule",
			job: main.Job{
				Name:     "Test job",
				Schedule: "shrug",
				Config:   ValidResticConfig(),
				Tasks:    []main.JobTask{},
				Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []main.JobTaskMySQL{},
				Postgres: []main.JobTaskPostgres{},
				Sqlite:   []main.JobTaskSqlite{},
			},
			expectedErr: main.ErrInvalidConfigValue,
		},
		{
			name: "Invalid config",
			job: main.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   &main.ResticConfig{}, //nolint:exhaustruct
				Tasks:    []main.JobTask{},
				Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []main.JobTaskMySQL{},
				Postgres: []main.JobTaskPostgres{},
				Sqlite:   []main.JobTaskSqlite{},
			},
			expectedErr: main.ErrMutuallyExclusive,
		},
		{
			name: "Invalid task",
			job: main.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks: []main.JobTask{
					{}, //nolint:exhaustruct
				},
				Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []main.JobTaskMySQL{},
				Postgres: []main.JobTaskPostgres{},
				Sqlite:   []main.JobTaskSqlite{},
			},
			expectedErr: main.ErrMissingField,
		},
		{
			name: "Invalid mysql",
			job: main.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []main.JobTask{},
				Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL: []main.JobTaskMySQL{
					{}, //nolint:exhaustruct
				},
				Postgres: []main.JobTaskPostgres{},
				Sqlite:   []main.JobTaskSqlite{},
			},
			expectedErr: main.ErrMissingField,
		},
		{
			name: "Invalid sqlite",
			job: main.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []main.JobTask{},
				Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []main.JobTaskMySQL{},
				Postgres: []main.JobTaskPostgres{},
				Sqlite: []main.JobTaskSqlite{
					{}, //nolint:exhaustruct
				},
			},
			expectedErr: main.ErrMissingField,
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := testCase.job.Validate()

			if !errors.Is(actual, testCase.expectedErr) {
				t.Errorf("expected %v but found %v", testCase.expectedErr, actual)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		config      main.Config
		expectedErr error
	}{
		{
			name: "Valid job",
			config: main.Config{
				DefaultConfig: nil,
				Jobs: []main.Job{{
					Name:     "Valid job",
					Schedule: "@daily",
					Config:   ValidResticConfig(),
					Tasks:    []main.JobTask{},
					Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []main.JobTaskMySQL{},
					Postgres: []main.JobTaskPostgres{},
					Sqlite:   []main.JobTaskSqlite{},
				}},
			},
			expectedErr: nil,
		},
		{
			name: "Valid job with default config",
			config: main.Config{
				DefaultConfig: ValidResticConfig(),
				Jobs: []main.Job{{
					Name:     "Valid job",
					Schedule: "@daily",
					Config:   nil,
					Tasks:    []main.JobTask{},
					Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []main.JobTaskMySQL{},
					Postgres: []main.JobTaskPostgres{},
					Sqlite:   []main.JobTaskSqlite{},
				}},
			},
			expectedErr: nil,
		},
		{
			name: "No jobs",
			config: main.Config{
				DefaultConfig: nil,
				Jobs:          []main.Job{},
			},
			expectedErr: main.ErrNoJobsFound,
		},
		{
			name: "Invalid name",
			config: main.Config{
				DefaultConfig: nil,
				Jobs: []main.Job{{
					Name:     "",
					Schedule: "@daily",
					Config:   ValidResticConfig(),
					Tasks:    []main.JobTask{},
					Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []main.JobTaskMySQL{},
					Postgres: []main.JobTaskPostgres{},
					Sqlite:   []main.JobTaskSqlite{},
				}},
			},
			expectedErr: main.ErrMissingField,
		},
		{
			name: "Missing config",
			config: main.Config{
				DefaultConfig: nil,
				Jobs: []main.Job{{
					Name:     "",
					Schedule: "@daily",
					Config:   nil,
					Tasks:    []main.JobTask{},
					Backup:   main.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []main.JobTaskMySQL{},
					Postgres: []main.JobTaskPostgres{},
					Sqlite:   []main.JobTaskSqlite{},
				}},
			},
			expectedErr: main.ErrMissingField,
		},
	}

	for _, c := range cases {
		testCase := c

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := testCase.config.Validate()

			if !errors.Is(actual, testCase.expectedErr) {
				t.Errorf("expected %v but found %v", testCase.expectedErr, actual)
			}
		})
	}
}
