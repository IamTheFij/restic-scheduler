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
	DryRun            bool     `hcl:"DryRun,optional"`
	Exclude           []string `hcl:"Exclude,optional"`
	ExcludeCaches     bool     `hcl:"ExcludeCaches,optional"`
	ExcludeFile       []string `hcl:"ExcludeFile,optional"`
	ExcludeIfPresent  []string `hcl:"ExcludeIfPresent,optional"`
	ExcludeLargerThan string   `hcl:"ExcludeLargerThan,optional"`
	FilesFrom         []string `hcl:"FilesFrom,optional"`
	FilesFromRaw      []string `hcl:"FilesFromRaw,optional"`
	FilesFromVerbatim []string `hcl:"FilesFromVerbatim,optional"`
	Force             bool     `hcl:"Force,optional"`
	GroupBy           string   `hcl:"GroupBy,optional"`
	Host              string   `hcl:"Host,optional"`
	IExclude          []string `hcl:"IExclude,optional"`
	IExcludeFile      []string `hcl:"IExcludeFile,optional"`
	IgnoreCtime       bool     `hcl:"IgnoreCtime,optional"`
	IgnoreInode       bool     `hcl:"IgnoreInode,optional"`
	Include           []string `hcl:"Include,optional"`
	NoScan            bool     `hcl:"NoScan,optional"`
	OneFileSystem     bool     `hcl:"OneFileSystem,optional"`
	Parent            string   `hcl:"Parent,optional"`
	ReadConcurrency   int      `hcl:"ReadConcurrency,optional"`
	SkipIfUnchanged   bool     `hcl:"SkipIfUnchanged,optional"`
	Tags              []string `hcl:"Tags,optional"`
	WithATime         bool     `hcl:"WithATime,optional"`
}

// ToArgs returns the structs arguments as a slice of strings.
func (bo BackupOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgBool(args, "--dry-run", bo.DryRun)
	args = utils.MaybeAddArgsList(args, "--exclude", bo.Exclude)
	args = utils.MaybeAddArgBool(args, "--exclude-caches", bo.ExcludeCaches)
	args = utils.MaybeAddArgsList(args, "--exclude-file", bo.ExcludeFile)
	args = utils.MaybeAddArgsList(args, "--exclude-if-present", bo.ExcludeIfPresent)
	args = utils.MaybeAddArgString(args, "--exclude-larger-than", bo.ExcludeLargerThan)
	args = utils.MaybeAddArgsList(args, "--files-from", bo.FilesFrom)
	args = utils.MaybeAddArgsList(args, "--files-from-raw", bo.FilesFromRaw)
	args = utils.MaybeAddArgsList(args, "--files-from-verbatim", bo.FilesFromVerbatim)
	args = utils.MaybeAddArgBool(args, "--force", bo.Force)
	args = utils.MaybeAddArgString(args, "--group-by", bo.GroupBy)
	args = utils.MaybeAddArgString(args, "--host", bo.Host)
	args = utils.MaybeAddArgsList(args, "--iexclude", bo.IExclude)
	args = utils.MaybeAddArgsList(args, "--iexclude-file", bo.IExcludeFile)
	args = utils.MaybeAddArgBool(args, "--ignore-ctime", bo.IgnoreCtime)
	args = utils.MaybeAddArgBool(args, "--ignore-inode", bo.IgnoreInode)
	args = utils.MaybeAddArgsList(args, "--include", bo.Include)
	args = utils.MaybeAddArgBool(args, "--no-scan", bo.NoScan)
	args = utils.MaybeAddArgBool(args, "--one-file-system", bo.OneFileSystem)
	args = utils.MaybeAddArgString(args, "--parent", bo.Parent)
	args = utils.MaybeAddArgInt(args, "--read-concurrency", bo.ReadConcurrency)
	args = utils.MaybeAddArgBool(args, "--skip-if-unchanged", bo.SkipIfUnchanged)
	args = utils.MaybeAddArgsList(args, "--tag", bo.Tags)
	args = utils.MaybeAddArgBool(args, "--with-atime", bo.WithATime)

	return
}

type RestoreOpts struct {
	Delete       bool     `hcl:"Delete,optional"`
	DryRun       bool     `hcl:"DryRun,optional"`
	Exclude      []string `hcl:"Exclude,optional"`
	ExcludeFile  []string `hcl:"ExcludeFile,optional"`
	ExcludeXattr []string `hcl:"ExcludeXattr,optional"`
	Hosts        []string `hcl:"Hosts,optional"`
	IExclude     []string `hcl:"IExclude,optional"`
	IExcludeFile []string `hcl:"IExcludeFile,optional"`
	IInclude     []string `hcl:"IInclude,optional"`
	IIncludeFile []string `hcl:"IIncludeFile,optional"`
	Include      []string `hcl:"Include,optional"`
	IncludeFile  []string `hcl:"IncludeFile,optional"`
	IncludeXattr []string `hcl:"IncludeXattr,optional"`
	Overwrite    string   `hcl:"Overwrite,optional"`
	Paths        []string `hcl:"Paths,optional"`
	Sparse       bool     `hcl:"Sparse,optional"`
	Tags         []string `hcl:"Tags,optional"`
	Target       string   `hcl:"Target,optional"`
	Verify       bool     `hcl:"Verify,optional"`
}

// ToArgs returns the structs arguments as a slice of strings.
func (ro RestoreOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgBool(args, "--delete", ro.Delete)
	args = utils.MaybeAddArgBool(args, "--dry-run", ro.DryRun)
	args = utils.MaybeAddArgsList(args, "--exclude", ro.Exclude)
	args = utils.MaybeAddArgsList(args, "--exclude-file", ro.ExcludeFile)
	args = utils.MaybeAddArgsList(args, "--exclude-xattr", ro.ExcludeXattr)
	args = utils.MaybeAddArgsList(args, "--host", ro.Hosts)
	args = utils.MaybeAddArgsList(args, "--iexclude", ro.IExclude)
	args = utils.MaybeAddArgsList(args, "--iexclude-file", ro.IExcludeFile)
	args = utils.MaybeAddArgsList(args, "--iinclude", ro.IInclude)
	args = utils.MaybeAddArgsList(args, "--iinclude-file", ro.IIncludeFile)
	args = utils.MaybeAddArgsList(args, "--include", ro.Include)
	args = utils.MaybeAddArgsList(args, "--include-file", ro.IncludeFile)
	args = utils.MaybeAddArgsList(args, "--include-xattr", ro.IncludeXattr)
	args = utils.MaybeAddArgString(args, "--overwrite", ro.Overwrite)
	args = utils.MaybeAddArgsList(args, "--path", ro.Paths)
	args = utils.MaybeAddArgBool(args, "--sparse", ro.Sparse)
	args = utils.MaybeAddArgsList(args, "--tag", ro.Tags)
	args = utils.MaybeAddArgString(args, "--target", ro.Target)
	args = utils.MaybeAddArgBool(args, "--verify", ro.Verify)

	return
}

type ForgetOpts struct {
	KeepLast    int `hcl:"KeepLast,optional"`
	KeepDaily   int `hcl:"KeepDaily,optional"`
	KeepHourly  int `hcl:"KeepHourly,optional"`
	KeepMonthly int `hcl:"KeepMonthly,optional"`
	KeepWeekly  int `hcl:"KeepWeekly,optional"`
	KeepYearly  int `hcl:"KeepYearly,optional"`

	KeepWithin        time.Duration `hcl:"KeepWithin,optional"`
	KeepWithinDaily   time.Duration `hcl:"KeepWithinDaily,optional"`
	KeepWithinHourly  time.Duration `hcl:"KeepWithinHourly,optional"`
	KeepWithinMonthly time.Duration `hcl:"KeepWithinMonthly,optional"`
	KeepWithinWeekly  time.Duration `hcl:"KeepWithinWeekly,optional"`
	KeepWithinYearly  time.Duration `hcl:"KeepWithinYearly,optional"`

	Compact              bool     `hcl:"Compact,optional"`
	DryRun               bool     `hcl:"DryRun,optional"`
	GroupBy              string   `hcl:"GroupBy,optional"`
	Hosts                []string `hcl:"Hosts,optional"`
	Paths                []string `hcl:"Paths,optional"`
	Tags                 []string `hcl:"Tags,optional"`
	KeepTags             []string `hcl:"KeepTags,optional"`
	UnsafeAllowRemoveAll bool     `hcl:"UnsafeAllowRemoveAll,optional"`

	Prune               bool   `hcl:"Prune,optional"`
	MaxUnused           string `hcl:"MaxUnused,optional"`
	MaxRepackSize       string `hcl:"MaxRepackSize,optional"`
	RepackCacheableOnly bool   `hcl:"RepackCacheableOnly,optional"`
	RepackSmall         bool   `hcl:"RepackSmall,optional"`
	RepackUncompressed  bool   `hcl:"RepackUncompressed,optional"`
	RepackSmallerThan   string `hcl:"RepackSmallerThan,optional"`
}

// ToArgs returns the structs arguments as a slice of strings.
func (fo ForgetOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgInt(args, "--keep-last", fo.KeepLast)
	args = utils.MaybeAddArgInt(args, "--keep-hourly", fo.KeepHourly)
	args = utils.MaybeAddArgInt(args, "--keep-daily", fo.KeepDaily)
	args = utils.MaybeAddArgInt(args, "--keep-weekly", fo.KeepWeekly)
	args = utils.MaybeAddArgInt(args, "--keep-monthly", fo.KeepMonthly)
	args = utils.MaybeAddArgInt(args, "--keep-yearly", fo.KeepYearly)
	args = utils.MaybeAddArgDuration(args, "--keep-within", fo.KeepWithin)
	args = utils.MaybeAddArgDuration(args, "--keep-within-hourly", fo.KeepWithinHourly)
	args = utils.MaybeAddArgDuration(args, "--keep-within-daily", fo.KeepWithinDaily)
	args = utils.MaybeAddArgDuration(args, "--keep-within-weekly", fo.KeepWithinWeekly)
	args = utils.MaybeAddArgDuration(args, "--keep-within-monthly", fo.KeepWithinMonthly)
	args = utils.MaybeAddArgDuration(args, "--keep-within-yearly", fo.KeepWithinYearly)
	args = utils.MaybeAddArgBool(args, "--compact", fo.Compact)
	args = utils.MaybeAddArgBool(args, "--dry-run", fo.DryRun)
	args = utils.MaybeAddArgString(args, "--group-by", fo.GroupBy)
	args = utils.MaybeAddArgsList(args, "--host", fo.Hosts)
	args = utils.MaybeAddArgsList(args, "--path", fo.Paths)
	args = utils.MaybeAddArgsList(args, "--tag", fo.Tags)
	args = utils.MaybeAddArgsList(args, "--keep-tag", fo.KeepTags)
	args = utils.MaybeAddArgBool(args, "--unsafe-allow-remove-all", fo.UnsafeAllowRemoveAll)
	args = utils.MaybeAddArgBool(args, "--prune", fo.Prune)
	args = utils.MaybeAddArgString(args, "--max-unused", fo.MaxUnused)
	args = utils.MaybeAddArgString(args, "--max-repack-size", fo.MaxRepackSize)
	args = utils.MaybeAddArgBool(args, "--repack-cacheable-only", fo.RepackCacheableOnly)
	args = utils.MaybeAddArgBool(args, "--repack-small", fo.RepackSmall)
	args = utils.MaybeAddArgBool(args, "--repack-uncompressed", fo.RepackUncompressed)
	args = utils.MaybeAddArgString(args, "--repack-smaller-than", fo.RepackSmallerThan)

	return
}

// CopyOpts contains options for the restic copy command
type CopyOpts struct {
	FromInsecureNoPassword bool     `hcl:"FromInsecureNoPassword,optional"`
	FromKeyHint            string   `hcl:"FromKeyHint,optional"`
	FromPasswordCommand    string   `hcl:"FromPasswordCommand,optional"`
	FromPasswordFile       string   `hcl:"FromPasswordFile,optional"`
	FromRepo               string   `hcl:"FromRepo,optional"`
	FromRepositoryFile     string   `hcl:"FromRepositoryFile,optional"`
	Hosts                  []string `hcl:"Hosts,optional"`
	Paths                  []string `hcl:"Paths,optional"`
	Tags                   []string `hcl:"Tags,optional"`
}

// ToArgs returns the structs arguments as a slice of strings.
func (co CopyOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgBool(args, "--from-insecure-no-password", co.FromInsecureNoPassword)
	args = utils.MaybeAddArgString(args, "--from-key-hint", co.FromKeyHint)
	args = utils.MaybeAddArgString(args, "--from-password-command", co.FromPasswordCommand)
	args = utils.MaybeAddArgString(args, "--from-password-file", co.FromPasswordFile)
	args = utils.MaybeAddArgString(args, "--from-repo", co.FromRepo)
	args = utils.MaybeAddArgString(args, "--from-repository-file", co.FromRepositoryFile)
	args = utils.MaybeAddArgsList(args, "--host", co.Hosts)
	args = utils.MaybeAddArgsList(args, "--path", co.Paths)
	args = utils.MaybeAddArgsList(args, "--tag", co.Tags)

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

func (rcmd Restic) Copy(copyOpts CopyOpts, snapshots ...string) error {
	_, err := rcmd.RunRestic("copy", copyOpts, snapshots...)

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
