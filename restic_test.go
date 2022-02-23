package main_test

import (
	"os"
	"testing"

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
