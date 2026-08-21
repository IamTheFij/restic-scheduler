package restic_test

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.iamthefij.com/iamthefij/restic-scheduler/restic"
	utils "git.iamthefij.com/iamthefij/restic-scheduler/utils"
)

func TestNoOpts(t *testing.T) {
	t.Parallel()

	args := restic.NoOpts{}.ToArgs()
	expected := []string{}

	utils.AssertEqual(t, "no opts returned some opts", expected, args)
}

func TestGlobalOptions(t *testing.T) {
	t.Parallel()

	args := restic.ResticGlobalOpts{
		CaCertFile:        "file",
		CacheDir:          "directory",
		PasswordFile:      "file",
		TLSClientCertFile: "file",
		LimitDownload:     1,
		LimitUpload:       1,
		VerboseLevel:      1,
		CleanupCache:      true,
		InsecureTLS:       true,
		NoCache:           true,
		NoLock:            true,
		Options: map[string]string{
			"key": "a long value",
		},
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
		"--insecure-tls",
		"--no-cache",
		"--no-lock",
		"--option", "key='a long value'",
	}

	utils.AssertEqual(t, "args didn't match", expected, args)
}

func TestBackupOpts(t *testing.T) {
	t.Parallel()

	args := restic.BackupOpts{
		Exclude: []string{"file1", "file2"},
		Include: []string{"directory"},
		Tags:    []string{"thing"},
		Host:    "steve",
	}.ToArgs()

	expected := []string{
		"--exclude", "file1",
		"--exclude", "file2",
		"--host", "steve",
		"--include", "directory",
		"--tag", "thing",
	}

	utils.AssertEqual(t, "args didn't match", expected, args)
}

func TestRestoreOpts(t *testing.T) {
	t.Parallel()

	args := restic.RestoreOpts{
		Exclude: []string{"file1", "file2"},
		Include: []string{"directory"},
		Hosts:   []string{"steve"},
		Tags:    []string{"thing"},
		Paths:   []string{"directory"},
		Target:  "directory",
		Verify:  true,
	}.ToArgs()

	expected := []string{
		"--exclude", "file1",
		"--exclude", "file2",
		"--host", "steve",
		"--include", "directory",
		"--path", "directory",
		"--tag", "thing",
		"--target", "directory",
		"--verify",
	}

	utils.AssertEqual(t, "args didn't match", expected, args)
}

func TestForgetOpts(t *testing.T) {
	t.Parallel()

	args := restic.ForgetOpts{
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
		Tags:              []string{"thing1,thing2", "otherthing"},
		KeepTags:          []string{"thing"},
		Prune:             true,
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

	utils.AssertEqual(t, "args didn't match", expected, args)
}

func TestUnlockOpts(t *testing.T) {
	t.Parallel()

	args := restic.UnlockOpts{
		RemoveAll: true,
	}.ToArgs()

	expected := []string{
		"--remove-all",
	}

	utils.AssertEqual(t, "args didn't match", expected, args)
}

func TestResticCopyOpts(t *testing.T) {
	t.Parallel()

	args := restic.CopyOpts{
		FromRepo: "s3:bucket",
		Hosts:    []string{"host1", "host2"},
		Paths:    []string{"path1", "path2"},
		Tags:     []string{"tag1", "tag2"},
	}.ToArgs()

	expected := []string{
		"--from-repo", "s3:bucket",
		"--host", "host1",
		"--host", "host2",
		"--path", "path1",
		"--path", "path2",
		"--tag", "tag1",
		"--tag", "tag2",
	}

	utils.AssertEqual(t, "args didn't match", expected, args)
}

func TestResticInitOpts(t *testing.T) {
	t.Parallel()

	args := restic.InitOpts{
		CopyChunkerParams: true,
		FromRepo:          "s3:bucket",
	}.ToArgs()
	expected := []string{
		"--copy-chunker-params",
		"--from-repo", "s3:bucket",
	}

	utils.AssertEqual(t, "init opts returned some opts", expected, args)
}

func TestBuildEnv(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		cmd      restic.Restic
		expected []string
	}{
		{
			name:     "No Env",
			cmd:      restic.Restic{}, //nolint:exhaustruct
			expected: os.Environ(),
		},
		{
			name: "SetEnv",
			cmd: restic.Restic{ //nolint:exhaustruct
				Env: map[string]string{"TestKey": "Value"},
			},
			expected: append(os.Environ(), "TestKey=Value"),
		},
		{
			name: "SetEnv",
			cmd: restic.Restic{ //nolint:exhaustruct
				Passphrase: "Shhhhhhhh!!",
			},
			expected: append(os.Environ(), "RESTIC_PASSWORD=Shhhhhhhh!!"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			utils.AssertEqual(t, "args didn't match", c.expected, c.cmd.BuildEnv())
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

	r := restic.Restic{
		Logger:     log.New(os.Stderr, t.Name()+":", log.Lmsgprefix),
		Repo:       repoDir,
		Env:        map[string]string{},
		Passphrase: "Correct.Horse.Battery.Staple",
		//nolint:exhaustruct
		GlobalOpts: &restic.ResticGlobalOpts{
			CacheDir: cacheDir,
			Options: map[string]string{
				"s3.storage-class": "REDUCED_REDUNDANCY",
			},
		},
		Cwd: dataDir,
	}

	// Write test file to the data dir
	err := os.WriteFile(dataFile, []byte("testing"), 0o644)
	utils.AssertEqualFail(t, "unexpected error writing to test file", nil, err)

	// Make sure no existing repo is found
	_, err = r.ReadSnapshots()
	if err == nil || !errors.Is(err, restic.ErrRepoNotFound) {
		utils.AssertEqualFail(t, "didn't get expected error for snapshots", restic.ErrRepoNotFound.Error(), err.Error())
	}

	// Try to backup when repo is not initialized
	err = r.Backup([]string{dataDir}, restic.BackupOpts{}) //nolint:exhaustruct
	if !errors.Is(err, restic.ErrRepoNotFound) {
		utils.AssertEqualFail(t, "unexpected error creating making backup", nil, err)
	}

	// Init repo
	err = r.EnsureInit(restic.InitOpts{})
	utils.AssertEqualFail(t, "unexpected error initializing repo", nil, err)

	// Verify it can be reinitialized with no issues
	err = r.EnsureInit(restic.InitOpts{})
	utils.AssertEqualFail(t, "unexpected error reinitializing repo", nil, err)

	// Backup for real this time
	err = r.Backup([]string{dataDir}, restic.BackupOpts{Tags: []string{"test"}}) //nolint:exhaustruct
	utils.AssertEqualFail(t, "unexpected error creating making backup", nil, err)

	// Check snapshots
	expectedHostname, _ := os.Hostname()
	snapshots, err := r.ReadSnapshots()
	utils.AssertEqualFail(t, "unexpected error reading snapshots", nil, err)
	utils.AssertEqual(t, "unexpected number of snapshots", 1, len(snapshots))

	utils.AssertEqual(t, "unexpected snapshot value: hostname", expectedHostname, snapshots[0].Hostname)
	utils.AssertEqual(t, "unexpected snapshot value: paths", []string{dataDir}, snapshots[0].Paths)
	utils.AssertEqual(t, "unexpected snapshot value: tags", []string{"test"}, snapshots[0].Tags)

	// Backup again
	err = r.Backup([]string{dataDir}, restic.BackupOpts{}) //nolint:exhaustruct
	utils.AssertEqualFail(t, "unexpected error creating making second backup", nil, err)

	// Check for second backup
	snapshots, err = r.ReadSnapshots()
	utils.AssertEqualFail(t, "unexpected error reading second snapshots", nil, err)
	utils.AssertEqual(t, "unexpected number of snapshots", 2, len(snapshots))

	// Forget one backup
	err = r.Forget(restic.ForgetOpts{KeepLast: 1, Prune: true}) //nolint:exhaustruct
	utils.AssertEqualFail(t, "unexpected error forgetting snapshot", nil, err)

	// Check forgotten snapshot
	snapshots, err = r.ReadSnapshots()
	utils.AssertEqualFail(t, "unexpected error reading post forget snapshots", nil, err)
	utils.AssertEqual(t, "unexpected number of snapshots", 1, len(snapshots))

	// Check restic repo
	err = r.Check()
	utils.AssertEqualFail(t, "unexpected error checking repo", nil, err)

	// Change the data file
	err = os.WriteFile(dataFile, []byte("unexpected"), 0o644)
	utils.AssertEqualFail(t, "unexpected error writing to test file", nil, err)

	// Check that data wrote
	value, err := os.ReadFile(dataFile)
	utils.AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	utils.AssertEqualFail(t, "incorrect value in test file (we expect the unexpected!)", "unexpected", string(value))

	// Restore files
	err = r.Restore("latest", restic.RestoreOpts{Target: restoreTarget}) //nolint:exhaustruct
	utils.AssertEqualFail(t, "unexpected error restoring latest snapshot", nil, err)

	// Check restored values
	value, err = os.ReadFile(restoredDataFile)
	utils.AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	utils.AssertEqualFail(t, "incorrect value in test file", "testing", string(value))

	// Try to unlock the repo (repo shouldn't really be locked, but this should still run without error
	err = r.Unlock(restic.UnlockOpts{}) //nolint:exhaustruct
	utils.AssertEqualFail(t, "unexpected error unlocking repo", nil, err)

	// Test copy functionality
	repoCopyDir := t.TempDir()

	copyOpts := restic.CopyOpts{
		FromRepo:            repoDir,
		FromPasswordCommand: "echo 'Correct.Horse.Battery.Staple'",
	}

	rPrime := restic.Restic{
		Logger:     log.New(os.Stderr, t.Name()+":Prime:", log.Lmsgprefix),
		Repo:       repoCopyDir,
		Env:        map[string]string{},
		Passphrase: "Wrong.Pony.Solar.Tape",
		GlobalOpts: &restic.ResticGlobalOpts{
			CacheDir: cacheDir,
			Options: map[string]string{
				"s3.storage-class": "REDUCED_REDUNDANCY",
			},
		},
		Cwd: dataDir,
	}

	// Make sure no existing repo is found
	_, err = rPrime.ReadSnapshots()
	if err == nil || !errors.Is(err, restic.ErrRepoNotFound) {
		utils.AssertEqualFail(t, "copy repo: didn't get expected error for snapshots", restic.ErrRepoNotFound.Error(), err.Error())
	}

	// Try to copy when repo is not initialized
	err = rPrime.Copy(copyOpts)
	if !errors.Is(err, restic.ErrRepoNotFound) {
		utils.AssertEqualFail(t, "unexpected error creating making copying repo", nil, err)
	}

	// Init repo
	err = rPrime.EnsureInit(restic.InitOpts{
		CopyChunkerParams:   true,
		FromRepo:            repoDir,
		FromPasswordCommand: "echo 'Correct.Horse.Battery.Staple'",
	})
	utils.AssertEqualFail(t, "unexpected error initializing copy repo", nil, err)

	// Verify it can be reinitialized with no issues
	err = rPrime.EnsureInit(restic.InitOpts{
		CopyChunkerParams:   true,
		FromRepo:            repoDir,
		FromPasswordCommand: "echo 'Correct.Horse.Battery.Staple'",
	})
	utils.AssertEqualFail(t, "unexpected error initializing copy repo", nil, err)

	// Try to make copy
	err = rPrime.Copy(copyOpts)
	utils.AssertEqualFail(t, "unexpected error creating making repo copy", nil, err)

	// Check copy snapshots
	snapshots, err = rPrime.ReadSnapshots()
	utils.AssertEqualFail(t, "unexpected error reading copy snapshots", nil, err)
	utils.AssertEqual(t, "unexpected number of copy snapshots", 1, len(snapshots))

	// Change the previously restored data file
	err = os.WriteFile(restoredDataFile, []byte("unexpected"), 0o644)
	utils.AssertEqualFail(t, "unexpected error writing to test file", nil, err)

	// Check that data wrote
	value, err = os.ReadFile(restoredDataFile)
	utils.AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	utils.AssertEqualFail(t, "incorrect value in test file (we expect the unexpected!)", "unexpected", string(value))

	// Restore files from copy repo
	err = rPrime.Restore("latest", restic.RestoreOpts{Target: restoreTarget}) //nolint:exhaustruct
	utils.AssertEqualFail(t, "unexpected error restoring latest snapshot", nil, err)

	// Check restored values
	value, err = os.ReadFile(restoredDataFile)
	utils.AssertEqualFail(t, "unexpected error reading from test file", nil, err)
	utils.AssertEqualFail(t, "incorrect value in test file", "testing", string(value))
}
