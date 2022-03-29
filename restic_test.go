package main_test

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
)

func TestNoOpts(t *testing.T) {
	t.Parallel()

	args := main.NoOpts{}.ToArgs()
	expected := []string{}

	AssertEqual(t, "no opts returned some opts", expected, args)
}

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

	AssertEqual(t, "args didn't match", expected, args)
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

	AssertEqual(t, "args didn't match", expected, args)
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

	AssertEqual(t, "args didn't match", expected, args)
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

	AssertEqual(t, "args didn't match", expected, args)
}

func TestBuildEnv(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		cmd      main.Restic
		expected []string
	}{
		{
			name:     "No Env",
			cmd:      main.Restic{}, // nolint:exhaustivestruct
			expected: os.Environ(),
		},
		{
			name: "SetEnv",
			cmd: main.Restic{ // nolint:exhaustivestruct
				Env: map[string]string{"TestKey": "Value"},
			},
			expected: append(os.Environ(), "TestKey=Value"),
		},
		{
			name: "SetEnv",
			cmd: main.Restic{ // nolint:exhaustivestruct
				Passphrase: "Shhhhhhhh!!",
			},
			expected: append(os.Environ(), "RESTIC_PASSWORD=Shhhhhhhh!!"),
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			AssertEqual(t, "args didn't match", c.expected, c.cmd.BuildEnv())
		})
	}
}

func TestResticInterface(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skip integration test when running short tests")
	}

	dataDir := t.TempDir()
	repoDir := t.TempDir()
	cacheDir := t.TempDir()
	restoreTarget := t.TempDir()

	dataFile := filepath.Join(dataDir, "test.txt")
	restoredDataFile := filepath.Join(restoreTarget, dataFile)

	restic := main.Restic{
		Logger:     log.New(os.Stderr, t.Name()+":", log.Lmsgprefix),
		Repo:       repoDir,
		Env:        map[string]string{},
		Passphrase: "Correct.Horse.Battery.Staple",
		// nolint:exhaustivestruct
		GlobalOpts: &main.ResticGlobalOpts{
			CacheDir: cacheDir,
		},
		Cwd: dataDir,
	}

	// Write test file to the data dir
	err := os.WriteFile(dataFile, []byte("testing"), 0644)
	AssertEqualFail(t, "unexpected error writing to test file", nil, err)

	// Make sure no existing repo is found
	_, err = restic.ReadSnapshots()
	if err == nil || !errors.Is(err, main.ErrRepoNotFound) {
		AssertEqualFail(t, "didn't get expected error for backup", main.ErrRepoNotFound, err)
	}

	// Try to backup when repo is not initialized
	err = restic.Backup([]string{dataDir}, main.BackupOpts{}) // nolint:exhaustivestruct
	if !errors.Is(err, main.ErrRepoNotFound) {
		AssertEqualFail(t, "unexpected error creating making backup", nil, err)
	}

	// Init repo
	err = restic.EnsureInit()
	AssertEqualFail(t, "unexpected error initializing repo", nil, err)

	// Verify it can be reinitialized with no issues
	err = restic.EnsureInit()
	AssertEqualFail(t, "unexpected error reinitializing repo", nil, err)

	// Backup for real this time
	err = restic.Backup([]string{dataDir}, main.BackupOpts{Tags: []string{"test"}}) // nolint:exhaustivestruct
	AssertEqualFail(t, "unexpected error creating making backup", nil, err)

	// Check snapshots
	expectedHostname, _ := os.Hostname()
	snapshots, err := restic.ReadSnapshots()
	AssertEqualFail(t, "unexpected error reading snapshots", nil, err)
	AssertEqual(t, "unexpected number of snapshots", 1, len(snapshots))

	AssertEqual(t, "unexpected snapshot value: hostname", expectedHostname, snapshots[0].Hostname)
	AssertEqual(t, "unexpected snapshot value: paths", []string{dataDir}, snapshots[0].Paths)
	AssertEqual(t, "unexpected snapshot value: tags", []string{"test"}, snapshots[0].Tags)

	// Backup again
	err = restic.Backup([]string{dataDir}, main.BackupOpts{}) // nolint:exhaustivestruct
	AssertEqualFail(t, "unexpected error creating making second backup", nil, err)

	// Check for second backup
	snapshots, err = restic.ReadSnapshots()
	AssertEqualFail(t, "unexpected error reading second snapshots", nil, err)
	AssertEqual(t, "unexpected number of snapshots", 2, len(snapshots))

	// Forget one backup
	err = restic.Forget(main.ForgetOpts{KeepLast: 1, Prune: true}) // nolint:exhaustivestruct
	AssertEqualFail(t, "unexpected error forgetting snapshot", nil, err)

	// Check forgotten snapshot
	snapshots, err = restic.ReadSnapshots()
	AssertEqualFail(t, "unexpected error reading post forget snapshots", nil, err)
	AssertEqual(t, "unexpected number of snapshots", 1, len(snapshots))

	// Check restic repo
	err = restic.Check()
	AssertEqualFail(t, "unexpected error checking repo", nil, err)

	// Change the data file
	err = os.WriteFile(dataFile, []byte("unexpected"), 0644)
	AssertEqualFail(t, "unexpected error writing to test file", nil, err)

	// Check that data wrote
	value, err := os.ReadFile(dataFile)
	AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	AssertEqualFail(t, "incorrect value in test file (we expect the unexpected!)", "unexpected", string(value))

	// Restore files
	err = restic.Restore("latest", main.RestoreOpts{Target: restoreTarget}) // nolint:exhaustivestruct
	AssertEqualFail(t, "unexpected error restoring latest snapshot", nil, err)

	// Check restored values
	value, err = os.ReadFile(restoredDataFile)
	AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	AssertEqualFail(t, "incorrect value in test file", "testing", string(value))
}
