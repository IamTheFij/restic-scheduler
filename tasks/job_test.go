package tasks_test

import (
	"errors"
	"testing"

	config "git.iamthefij.com/iamthefij/restic-scheduler/config"
	restic "git.iamthefij.com/iamthefij/restic-scheduler/restic"
	tasks "git.iamthefij.com/iamthefij/restic-scheduler/tasks"
)

func ValidResticConfig() *tasks.ResticConfig {
	return &tasks.ResticConfig{
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
		config      tasks.ResticConfig
		expectedErr error
	}{
		{
			name:        "missing passphrase",
			expectedErr: tasks.ErrMutuallyExclusive,
			config:      tasks.ResticConfig{}, //nolint:exhaustruct
		},
		{
			name:        "passphrase no file",
			expectedErr: nil,
			//nolint:exhaustruct
			config: tasks.ResticConfig{
				Passphrase: "shh",
			},
		},
		{
			name:        "file no passphrase",
			expectedErr: nil,
			//nolint:exhaustruct
			config: tasks.ResticConfig{
				GlobalOpts: &restic.ResticGlobalOpts{
					PasswordFile: "file",
				},
			},
		},
		{
			name:        "file and passphrase",
			expectedErr: tasks.ErrMutuallyExclusive,
			//nolint:exhaustruct
			config: tasks.ResticConfig{
				Passphrase: "shh",
				GlobalOpts: &restic.ResticGlobalOpts{
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
		job         tasks.Job
		expectedErr error
	}{
		{
			name: "Valid job",
			job: tasks.Job{
				Name:     "Valid job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []tasks.JobTask{},
				Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []tasks.JobTaskMySQL{},
				Postgres: []tasks.JobTaskPostgres{},
				Sqlite:   []tasks.JobTaskSqlite{},
			},
			expectedErr: nil,
		},
		{
			name: "Invalid name",
			job: tasks.Job{
				Name:     "",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []tasks.JobTask{},
				Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []tasks.JobTaskMySQL{},
				Postgres: []tasks.JobTaskPostgres{},
				Sqlite:   []tasks.JobTaskSqlite{},
			},
			expectedErr: tasks.ErrMissingField,
		},
		{
			name: "Invalid schedule",
			job: tasks.Job{
				Name:     "Test job",
				Schedule: "shrug",
				Config:   ValidResticConfig(),
				Tasks:    []tasks.JobTask{},
				Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []tasks.JobTaskMySQL{},
				Postgres: []tasks.JobTaskPostgres{},
				Sqlite:   []tasks.JobTaskSqlite{},
			},
			expectedErr: tasks.ErrInvalidConfigValue,
		},
		{
			name: "Invalid config",
			job: tasks.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   &tasks.ResticConfig{}, //nolint:exhaustruct
				Tasks:    []tasks.JobTask{},
				Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []tasks.JobTaskMySQL{},
				Postgres: []tasks.JobTaskPostgres{},
				Sqlite:   []tasks.JobTaskSqlite{},
			},
			expectedErr: tasks.ErrMutuallyExclusive,
		},
		{
			name: "Invalid task",
			job: tasks.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks: []tasks.JobTask{
					{}, //nolint:exhaustruct
				},
				Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []tasks.JobTaskMySQL{},
				Postgres: []tasks.JobTaskPostgres{},
				Sqlite:   []tasks.JobTaskSqlite{},
			},
			expectedErr: tasks.ErrMissingField,
		},
		{
			name: "Invalid mysql",
			job: tasks.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []tasks.JobTask{},
				Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL: []tasks.JobTaskMySQL{
					{}, //nolint:exhaustruct
				},
				Postgres: []tasks.JobTaskPostgres{},
				Sqlite:   []tasks.JobTaskSqlite{},
			},
			expectedErr: tasks.ErrMissingField,
		},
		{
			name: "Invalid sqlite",
			job: tasks.Job{
				Name:     "Test job",
				Schedule: "@daily",
				Config:   ValidResticConfig(),
				Tasks:    []tasks.JobTask{},
				Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
				Forget:   nil,
				MySQL:    []tasks.JobTaskMySQL{},
				Postgres: []tasks.JobTaskPostgres{},
				Sqlite: []tasks.JobTaskSqlite{
					{}, //nolint:exhaustruct
				},
			},
			expectedErr: tasks.ErrMissingField,
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
		config      config.Config
		expectedErr error
	}{
		{
			name: "Valid job",
			config: config.Config{
				DefaultConfig: nil,
				Jobs: []tasks.Job{{
					Name:     "Valid job",
					Schedule: "@daily",
					Config:   ValidResticConfig(),
					Tasks:    []tasks.JobTask{},
					Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []tasks.JobTaskMySQL{},
					Postgres: []tasks.JobTaskPostgres{},
					Sqlite:   []tasks.JobTaskSqlite{},
				}},
			},
			expectedErr: nil,
		},
		{
			name: "Valid job with default config",
			config: config.Config{
				DefaultConfig: ValidResticConfig(),
				Jobs: []tasks.Job{{
					Name:     "Valid job",
					Schedule: "@daily",
					Config:   nil,
					Tasks:    []tasks.JobTask{},
					Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []tasks.JobTaskMySQL{},
					Postgres: []tasks.JobTaskPostgres{},
					Sqlite:   []tasks.JobTaskSqlite{},
				}},
			},
			expectedErr: nil,
		},
		{
			name: "No jobs",
			config: config.Config{
				DefaultConfig: nil,
				Jobs:          []tasks.Job{},
			},
			expectedErr: config.ErrNoJobsFound,
		},
		{
			name: "Invalid name",
			config: config.Config{
				DefaultConfig: nil,
				Jobs: []tasks.Job{{
					Name:     "",
					Schedule: "@daily",
					Config:   ValidResticConfig(),
					Tasks:    []tasks.JobTask{},
					Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []tasks.JobTaskMySQL{},
					Postgres: []tasks.JobTaskPostgres{},
					Sqlite:   []tasks.JobTaskSqlite{},
				}},
			},
			expectedErr: tasks.ErrMissingField,
		},
		{
			name: "Missing config",
			config: config.Config{
				DefaultConfig: nil,
				Jobs: []tasks.Job{{
					Name:     "",
					Schedule: "@daily",
					Config:   nil,
					Tasks:    []tasks.JobTask{},
					Backup:   tasks.BackupFilesTask{Paths: []string{"/test"}}, //nolint:exhaustruct
					Forget:   nil,
					MySQL:    []tasks.JobTaskMySQL{},
					Postgres: []tasks.JobTaskPostgres{},
					Sqlite:   []tasks.JobTaskSqlite{},
				}},
			},
			expectedErr: tasks.ErrMissingField,
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
