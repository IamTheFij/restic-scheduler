package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var ErrRestic = errors.New("restic error")
var ErrRepoNotFound = errors.New("repository not found or uninitialized")

func lineIn(needle string, haystack []string) bool {
	for _, line := range haystack {
		if line == needle {
			return true
		}
	}

	return false
}

func maybeAddArgString(args []string, name, value string) []string {
	if value != "" {
		return append(args, name, value)
	}

	return args
}

func maybeAddArgInt(args []string, name string, value int) []string {
	if value > 0 {
		return append(args, name, fmt.Sprint(value))
	}

	return args
}

func maybeAddArgBool(args []string, name string, value bool) []string {
	if value {
		return append(args, name)
	}

	return args
}

func maybeAddArgsList(args []string, name string, value []string) []string {
	for _, v := range value {
		args = append(args, name, v)
	}

	return args
}

type CommandOptions interface {
	ToArgs() []string
}

type GenericOpts []string

func (o GenericOpts) ToArgs() []string {
	return o
}

type NoOpts struct{}

func (NoOpts) ToArgs() []string {
	return []string{}
}

type UnlockOpts struct {
	RemoveAll bool `hcl:"RemoveAll,optional"`
}

func (uo UnlockOpts) ToArgs() (args []string) {
	args = maybeAddArgBool(args, "--remove-all", uo.RemoveAll)

	return
}

type BackupOpts struct {
	Exclude []string `hcl:"Exclude,optional"`
	Include []string `hcl:"Include,optional"`
	Tags    []string `hcl:"Tags,optional"`
	Host    string   `hcl:"Host,optional"`
}

func (bo BackupOpts) ToArgs() (args []string) {
	args = maybeAddArgsList(args, "--exclude", bo.Exclude)
	args = maybeAddArgsList(args, "--include", bo.Include)
	args = maybeAddArgsList(args, "--tag", bo.Tags)
	args = maybeAddArgString(args, "--host", bo.Host)

	return
}

type RestoreOpts struct {
	Exclude []string `hcl:"Exclude,optional"`
	Include []string `hcl:"Include,optional"`
	Host    []string `hcl:"Host,optional"`
	Tags    []string `hcl:"Tags,optional"`
	Path    string   `hcl:"Path,optional"`
	Target  string   `hcl:"Target,optional"`
	Verify  bool     `hcl:"Verify,optional"`
}

func (ro RestoreOpts) ToArgs() (args []string) {
	args = maybeAddArgsList(args, "--exclude", ro.Exclude)
	args = maybeAddArgsList(args, "--include", ro.Include)
	args = maybeAddArgsList(args, "--host", ro.Host)
	args = maybeAddArgsList(args, "--tag", ro.Tags)
	args = maybeAddArgString(args, "--path", ro.Path)
	args = maybeAddArgString(args, "--target", ro.Target)
	args = maybeAddArgBool(args, "--verify", ro.Verify)

	return
}

type TagList []string

func (t TagList) String() string {
	return strings.Join(t, ",")
}

type ForgetOpts struct {
	KeepLast    int `hcl:"KeepLast,optional"`
	KeepHourly  int `hcl:"KeepHourly,optional"`
	KeepDaily   int `hcl:"KeepDaily,optional"`
	KeepWeekly  int `hcl:"KeepWeekly,optional"`
	KeepMonthly int `hcl:"KeepMonthly,optional"`
	KeepYearly  int `hcl:"KeepYearly,optional"`

	KeepWithin        time.Duration `hcl:"KeepWithin,optional"`
	KeepWithinHourly  time.Duration `hcl:"KeepWithinHourly,optional"`
	KeepWithinDaily   time.Duration `hcl:"KeepWithinDaily,optional"`
	KeepWithinWeekly  time.Duration `hcl:"KeepWithinWeekly,optional"`
	KeepWithinMonthly time.Duration `hcl:"KeepWithinMonthly,optional"`
	KeepWithinYearly  time.Duration `hcl:"KeepWithinYearly,optional"`

	Tags     []TagList `hcl:"Tags,optional"`
	KeepTags []TagList `hcl:"KeepTags,optional"`

	Prune bool `hcl:"Prune,optional"`
}

func (fo ForgetOpts) ToArgs() (args []string) {
	args = maybeAddArgInt(args, "--keep-last", fo.KeepLast)
	args = maybeAddArgInt(args, "--keep-hourly", fo.KeepHourly)
	args = maybeAddArgInt(args, "--keep-daily", fo.KeepDaily)
	args = maybeAddArgInt(args, "--keep-weekly", fo.KeepWeekly)
	args = maybeAddArgInt(args, "--keep-monthly", fo.KeepMonthly)
	args = maybeAddArgInt(args, "--keep-yearly", fo.KeepYearly)

	// Add keep-within-*

	if fo.KeepWithin > 0 {
		args = append(args, "--keep-within", fo.KeepWithin.String())
	}

	if fo.KeepWithinHourly > 0 {
		args = append(args, "--keep-within-hourly", fo.KeepWithinHourly.String())
	}

	if fo.KeepWithinDaily > 0 {
		args = append(args, "--keep-within-daily", fo.KeepWithinDaily.String())
	}

	if fo.KeepWithinWeekly > 0 {
		args = append(args, "--keep-within-weekly", fo.KeepWithinWeekly.String())
	}

	if fo.KeepWithinMonthly > 0 {
		args = append(args, "--keep-within-monthly", fo.KeepWithinMonthly.String())
	}

	if fo.KeepWithinYearly > 0 {
		args = append(args, "--keep-within-yearly", fo.KeepWithinYearly.String())
	}

	// Add tags
	for _, tagList := range fo.Tags {
		args = append(args, "--tag", tagList.String())
	}

	for _, tagList := range fo.KeepTags {
		args = append(args, "--keep-tag", tagList.String())
	}

	// Add prune options
	args = maybeAddArgBool(args, "--prune", fo.Prune)

	return args
}

type ResticGlobalOpts struct {
	CaCertFile        string            `hcl:"CaCertFile,optional"`
	CacheDir          string            `hcl:"CacheDir,optional"`
	PasswordFile      string            `hcl:"PasswordFile,optional"`
	TLSClientCertFile string            `hcl:"TlsClientCertFile,optional"`
	LimitDownload     int               `hcl:"LimitDownload,optional"`
	LimitUpload       int               `hcl:"LimitUpload,optional"`
	VerboseLevel      int               `hcl:"VerboseLevel,optional"`
	Options           map[string]string `hcl:"Options,optional"`
	CleanupCache      bool              `hcl:"CleanupCache,optional"`
	InsecureTLS       bool              `hcl:"InsecureTls,optional"`
	NoCache           bool              `hcl:"NoCache,optional"`
	NoLock            bool              `hcl:"NoLock,optional"`
}

func (glo ResticGlobalOpts) ToArgs() (args []string) {
	args = maybeAddArgString(args, "--cacert", glo.CaCertFile)
	args = maybeAddArgString(args, "--cache-dir", glo.CacheDir)
	args = maybeAddArgString(args, "--password-file", glo.PasswordFile)
	args = maybeAddArgString(args, "--tls-client-cert", glo.TLSClientCertFile)
	args = maybeAddArgInt(args, "--limit-download", glo.LimitDownload)
	args = maybeAddArgInt(args, "--limit-upload", glo.LimitUpload)
	args = maybeAddArgInt(args, "--verbose", glo.VerboseLevel)
	args = maybeAddArgBool(args, "--cleanup-cache", glo.CleanupCache)
	args = maybeAddArgBool(args, "--insecure-tls", glo.InsecureTLS)
	args = maybeAddArgBool(args, "--no-cache", glo.NoCache)
	args = maybeAddArgBool(args, "--no-lock", glo.NoLock)

	for key, value := range glo.Options {
		args = append(args, "--option", fmt.Sprintf("%s='%s'", key, value))
	}

	return args
}

type Restic struct {
	Logger     *log.Logger
	Repo       string
	Env        map[string]string
	Passphrase string
	GlobalOpts *ResticGlobalOpts
	Cwd        string
}

func (rcmd Restic) BuildEnv() []string {
	if rcmd.Env == nil {
		rcmd.Env = map[string]string{}
	}

	if rcmd.Passphrase != "" {
		rcmd.Env["RESTIC_PASSWORD"] = rcmd.Passphrase
	}

	envList := os.Environ()

	for name, value := range rcmd.Env {
		envList = append(envList, fmt.Sprintf("%s=%s", name, value))
	}

	return envList
}

type ResticError struct {
	OriginalError error
	Command       string
	Output        []string
}

func NewResticError(command string, output []string, originalError error) *ResticError {
	return &ResticError{
		OriginalError: originalError,
		Command:       command,
		Output:        output,
	}
}

func (e *ResticError) Error() string {
	return fmt.Sprintf(
		"error running restic %s: %s\nOutput:\n%s",
		e.Command,
		e.OriginalError,
		strings.Join(e.Output, "\n"),
	)
}

func (e *ResticError) Unwrap() error {
	return e.OriginalError
}

func (rcmd Restic) RunRestic(
	command string,
	options CommandOptions,
	commandArgs ...string,
) (*CapturedCommandLogWriter, error) {
	args := []string{}
	if rcmd.GlobalOpts != nil {
		args = rcmd.GlobalOpts.ToArgs()
	}

	args = append(args, "--repo", rcmd.Repo, command)
	args = append(args, options.ToArgs()...)
	args = append(args, commandArgs...)

	cmd := exec.Command("restic", args...)

	output := NewCapturedCommandLogWriter(rcmd.Logger)
	cmd.Stdout = output.Stdout
	cmd.Stderr = output.Stderr
	cmd.Env = rcmd.BuildEnv()
	cmd.Dir = rcmd.Cwd

	if err := cmd.Run(); err != nil {
		responseErr := ErrRestic
		if lineIn("Is there a repository at the following location?", output.Stderr.Lines) {
			responseErr = ErrRepoNotFound
		}

		return output, NewResticError(command, output.AllLines(), responseErr)
	}

	return output, nil
}

func (rcmd Restic) Backup(files []string, opts BackupOpts) error {
	_, err := rcmd.RunRestic("backup", opts, files...)

	return err
}

func (rcmd Restic) Restore(snapshot string, opts RestoreOpts) error {
	_, err := rcmd.RunRestic("restore", opts, snapshot)

	return err
}

func (rcmd Restic) Forget(forgetOpts ForgetOpts) error {
	_, err := rcmd.RunRestic("forget", forgetOpts)

	return err
}

func (rcmd Restic) Check() error {
	_, err := rcmd.RunRestic("check", NoOpts{})

	return err
}

func (rcmd Restic) Unlock(unlockOpts UnlockOpts) error {
	_, err := rcmd.RunRestic("unlock", unlockOpts)

	return err
}

type Snapshot struct {
	UID      int       `json:"uid"`
	GID      int       `json:"gid"`
	Time     time.Time `json:"time"`
	Tree     string    `json:"tree"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"` //nolint:tagliatelle
	Paths    []string  `json:"paths"`
	Tags     []string  `json:"tags,omitempty"`
}

func (rcmd Restic) ReadSnapshots() ([]Snapshot, error) {
	output, err := rcmd.RunRestic("snapshots", GenericOpts{"--json"})
	if err != nil {
		return nil, err
	}

	if len(output.Stdout.Lines) == 0 {
		return nil, fmt.Errorf("no snapshot output to parse: %w", ErrRestic)
	}

	singleLineOutput := strings.Join(output.Stdout.Lines, "")

	snapshots := new([]Snapshot)
	if err = json.Unmarshal([]byte(singleLineOutput), snapshots); err != nil {
		return nil, fmt.Errorf("failed parsing snapshot results from %s: %w", singleLineOutput, err)
	}

	return *snapshots, nil
}

func (rcmd Restic) Snapshots() error {
	_, err := rcmd.RunRestic("snapshots", NoOpts{})

	return err
}

func (rcmd Restic) EnsureInit() error {
	if err := rcmd.Snapshots(); errors.Is(err, ErrRepoNotFound) {
		_, err := rcmd.RunRestic("init", NoOpts{})

		return err
	}

	return nil
}
