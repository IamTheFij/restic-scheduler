package main_test

import (
	"os"
	"testing"
	"time"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
	"github.com/go-test/deep"
)

func TestGlobalOptions(t *testing.T) {
	t.Parallel()

	args := main.ResticGlobalOpts{
		CaCertFile:        "file",
		CacheDir:          "directory",
		PasswordFile:      "file",
		TLSClientCertFile: "file",
		LimitDownload:     1,
		LimitUpload:       1,
		VerboseLevel:      1,
		CleanupCache:      true,
		NoCache:           true,
		NoLock:            true,
	}.ToArgs()

	expected := []string{
		"--cacert", "file",
		"--cache-dir", "directory",
		"--password-file", "file",
		"--tls-client-cert", "file",
		"--limit-download", "1",
		"--limit-upload", "1",
		"--verbose", "1",
		"--cleanup-cache",
		"--no-cache",
		"--no-lock",
	}

	if diff := deep.Equal(args, expected); diff != nil {
		t.Errorf("args didn't match %v", diff)
	}
}

func TestBackupOpts(t *testing.T) {
	t.Parallel()

	args := main.BackupOpts{
		Exclude: []string{"file1", "file2"},
		Include: []string{"directory"},
		Tags:    []string{"thing"},
		Host:    "steve",
	}.ToArgs()

	expected := []string{
		"--exclude", "file1",
		"--exclude", "file2",
		"--include", "directory",
		"--tag", "thing",
		"--host", "steve",
	}

	if diff := deep.Equal(args, expected); diff != nil {
		t.Errorf("args didn't match %v", diff)
	}
}

func TestRestoreOpts(t *testing.T) {
	t.Parallel()

	args := main.RestoreOpts{
		Exclude: []string{"file1", "file2"},
		Include: []string{"directory"},
		Host:    []string{"steve"},
		Tags:    []string{"thing"},
		Path:    "directory",
		Target:  "directory",
		Verify:  true,
	}.ToArgs()

	expected := []string{
		"--exclude", "file1",
		"--exclude", "file2",
		"--include", "directory",
		"--host", "steve",
		"--tag", "thing",
		"--path", "directory",
		"--target", "directory",
		"--verify",
	}

	if diff := deep.Equal(args, expected); diff != nil {
		t.Errorf("args didn't match %v", diff)
	}
}

func TestForgetOpts(t *testing.T) {
	t.Parallel()

	args := main.ForgetOpts{
		KeepLast:          1,
		KeepHourly:        1,
		KeepDaily:         1,
		KeepWeekly:        1,
		KeepMonthly:       1,
		KeepYearly:        1,
		KeepWithin:        1 * time.Second,
		KeepWithinHourly:  1 * time.Second,
		KeepWithinDaily:   1 * time.Second,
		KeepWithinWeekly:  1 * time.Second,
		KeepWithinMonthly: 1 * time.Second,
		KeepWithinYearly:  1 * time.Second,
		Tags: []main.TagList{
			{"thing1", "thing2"},
			{"otherthing"},
		},
		KeepTags: []main.TagList{{"thing"}},
		Prune:    true,
	}.ToArgs()

	expected := []string{
		"--keep-last", "1",
		"--keep-hourly", "1",
		"--keep-daily", "1",
		"--keep-weekly", "1",
		"--keep-monthly", "1",
		"--keep-yearly", "1",
		"--keep-within", "1s",
		"--keep-within-hourly", "1s",
		"--keep-within-daily", "1s",
		"--keep-within-weekly", "1s",
		"--keep-within-monthly", "1s",
		"--keep-within-yearly", "1s",
		"--tag", "thing1,thing2",
		"--tag", "otherthing",
		"--keep-tag", "thing",
		"--prune",
	}

	if diff := deep.Equal(args, expected); diff != nil {
		t.Errorf("args didn't match %v", diff)
	}
}

func TestBuildEnv(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		cmd      main.ResticCmd
		expected []string
	}{
		{
			name:     "No Env",
			cmd:      main.ResticCmd{}, // nolint:exhaustivestruct
			expected: os.Environ(),
		},
		{
			name: "SetEnv",
			cmd: main.ResticCmd{ // nolint:exhaustivestruct
				Env: map[string]string{"TestKey": "Value"},
			},
			expected: append(os.Environ(), "TestKey=Value"),
		},
		{
			name: "SetEnv",
			cmd: main.ResticCmd{ // nolint:exhaustivestruct
				Passphrase: "Shhhhhhhh!!",
			},
			expected: append(os.Environ(), "RESTIC_PASSWORD=Shhhhhhhh!!"),
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if diff := deep.Equal(c.expected, c.cmd.BuildEnv()); diff != nil {
				t.Error(diff)
			}
		})
	}
}
