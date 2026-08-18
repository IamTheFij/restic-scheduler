package restic

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"git.iamthefij.com/iamthefij/restic-scheduler/utils"
)

var (
	ErrRestic       = errors.New("restic error")
	ErrRepoNotFound = errors.Join(errors.New("repository not found or uninitialized"), ErrRestic)
)

// CommandOptions interface dictates a ToArgs() method should return each commandline arg as a string slice.
type CommandOptions interface {
	// ToArgs returns the structs arguments as a slice of strings.
	ToArgs() []string
}

// GenericOpts allows passing an arbitrary string slice as a set of command line options compatible with CommandOptions.
type GenericOpts []string

// ToArgs returns the structs arguments as a slice of strings.
func (o GenericOpts) ToArgs() []string {
	return o
}

// NoOpts is a struct that fulfils the CommandOptions interface but provides no arguments.
type NoOpts struct{}

// ToArgs returns the structs arguments as a slice of strings.
func (NoOpts) ToArgs() []string {
	return []string{}
}

// UnlockOpts holds optional arguments for unlock command.
type UnlockOpts struct {
	RemoveAll bool `hcl:"RemoveAll,optional"`
}

// ToArgs returns the structs arguments as a slice of strings.
func (uo UnlockOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgBool(args, "--remove-all", uo.RemoveAll)

	return
}

// BackupOpts holds optional arguments for the Restic backup command.
type BackupOpts struct {
	Exclude []string `hcl:"Exclude,optional"`
	Include []string `hcl:"Include,optional"`
	Tags    []string `hcl:"Tags,optional"`
	Host    string   `hcl:"Host,optional"`
}

// ToArgs returns the structs arguments as a slice of strings.
func (bo BackupOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgsList(args, "--exclude", bo.Exclude)
	args = utils.MaybeAddArgsList(args, "--include", bo.Include)
	args = utils.MaybeAddArgsList(args, "--tag", bo.Tags)
	args = utils.MaybeAddArgString(args, "--host", bo.Host)

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

// ToArgs returns the structs arguments as a slice of strings.
func (ro RestoreOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgsList(args, "--exclude", ro.Exclude)
	args = utils.MaybeAddArgsList(args, "--include", ro.Include)
	args = utils.MaybeAddArgsList(args, "--host", ro.Host)
	args = utils.MaybeAddArgsList(args, "--tag", ro.Tags)
	args = utils.MaybeAddArgString(args, "--path", ro.Path)
	args = utils.MaybeAddArgString(args, "--target", ro.Target)
	args = utils.MaybeAddArgBool(args, "--verify", ro.Verify)

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

// ToArgs returns the structs arguments as a slice of strings.
func (fo ForgetOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgInt(args, "--keep-last", fo.KeepLast)
	args = utils.MaybeAddArgInt(args, "--keep-hourly", fo.KeepHourly)
	args = utils.MaybeAddArgInt(args, "--keep-daily", fo.KeepDaily)
	args = utils.MaybeAddArgInt(args, "--keep-weekly", fo.KeepWeekly)
	args = utils.MaybeAddArgInt(args, "--keep-monthly", fo.KeepMonthly)
	args = utils.MaybeAddArgInt(args, "--keep-yearly", fo.KeepYearly)

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
	args = utils.MaybeAddArgBool(args, "--prune", fo.Prune)

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

// ToArgs returns the structs arguments as a slice of strings.
func (glo ResticGlobalOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgString(args, "--cacert", glo.CaCertFile)
	args = utils.MaybeAddArgString(args, "--cache-dir", glo.CacheDir)
	args = utils.MaybeAddArgString(args, "--password-file", glo.PasswordFile)
	args = utils.MaybeAddArgString(args, "--tls-client-cert", glo.TLSClientCertFile)
	args = utils.MaybeAddArgInt(args, "--limit-download", glo.LimitDownload)
	args = utils.MaybeAddArgInt(args, "--limit-upload", glo.LimitUpload)
	args = utils.MaybeAddArgInt(args, "--verbose", glo.VerboseLevel)
	args = utils.MaybeAddArgBool(args, "--cleanup-cache", glo.CleanupCache)
	args = utils.MaybeAddArgBool(args, "--insecure-tls", glo.InsecureTLS)
	args = utils.MaybeAddArgBool(args, "--no-cache", glo.NoCache)
	args = utils.MaybeAddArgBool(args, "--no-lock", glo.NoLock)

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
) (*utils.CapturedCommandLogWriter, error) {
	args := []string{}
	if rcmd.GlobalOpts != nil {
		args = rcmd.GlobalOpts.ToArgs()
	}

	args = append(args, "--repo", rcmd.Repo, command)
	args = append(args, options.ToArgs()...)
	args = append(args, commandArgs...)

	cmd := exec.Command("restic", args...)

	output := utils.NewCapturedCommandLogWriter(rcmd.Logger)
	cmd.Stdout = output.Stdout
	cmd.Stderr = output.Stderr
	cmd.Env = rcmd.BuildEnv()
	cmd.Dir = rcmd.Cwd

	if err := cmd.Run(); err != nil {
		responseErr := ErrRestic
		// Check to see if the error is due to a missing repository
		singleLineOutput := strings.Join(output.Stderr.Lines, "\n")
		repositoryMissingMessage := "Is there a repository at the following location?"

		if strings.Contains(singleLineOutput, repositoryMissingMessage) {
			responseErr = ErrRepoNotFound
		}

		return output, NewResticError(command, output.AllLines(), errors.Join(err, responseErr))
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
