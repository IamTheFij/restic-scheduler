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
	DryRun            bool     `hcl:"dry_run,optional"`
	Exclude           []string `hcl:"exclude,optional"`
	ExcludeCaches     bool     `hcl:"exclude_caches,optional"`
	ExcludeFile       []string `hcl:"exclude_file,optional"`
	ExcludeIfPresent  []string `hcl:"exclude_if_present,optional"`
	ExcludeLargerThan string   `hcl:"exclude_larger_than,optional"`
	FilesFrom         []string `hcl:"files_from,optional"`
	FilesFromRaw      []string `hcl:"files_from_raw,optional"`
	FilesFromVerbatim []string `hcl:"files_from_verbatim,optional"`
	Force             bool     `hcl:"force,optional"`
	GroupBy           string   `hcl:"group_by,optional"`
	Host              string   `hcl:"host,optional"`
	IExclude          []string `hcl:"iexclude,optional"`
	IExcludeFile      []string `hcl:"iexclude_file,optional"`
	IgnoreCtime       bool     `hcl:"ignore_ctime,optional"`
	IgnoreInode       bool     `hcl:"ignore_inode,optional"`
	Include           []string `hcl:"include,optional"`
	NoScan            bool     `hcl:"no_scan,optional"`
	OneFileSystem     bool     `hcl:"one_file_system,optional"`
	Parent            string   `hcl:"parent,optional"`
	ReadConcurrency   int      `hcl:"read_concurrency,optional"`
	SkipIfUnchanged   bool     `hcl:"skip_if_unchanged,optional"`
	Tags              []string `hcl:"tags,optional"`
	WithATime         bool     `hcl:"with_atime,optional"`
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
	Delete       bool     `hcl:"delete,optional"`
	DryRun       bool     `hcl:"dry_run,optional"`
	Exclude      []string `hcl:"exclude,optional"`
	ExcludeFile  []string `hcl:"exclude_file,optional"`
	ExcludeXattr []string `hcl:"exclude_xattr,optional"`
	Hosts        []string `hcl:"hosts,optional"`
	IExclude     []string `hcl:"iexclude,optional"`
	IExcludeFile []string `hcl:"iexclude_file,optional"`
	IInclude     []string `hcl:"iinclude,optional"`
	IIncludeFile []string `hcl:"iinclude_file,optional"`
	Include      []string `hcl:"include,optional"`
	IncludeFile  []string `hcl:"include_file,optional"`
	IncludeXattr []string `hcl:"include_xattr,optional"`
	Overwrite    string   `hcl:"overwrite,optional"`
	Paths        []string `hcl:"paths,optional"`
	Sparse       bool     `hcl:"sparse,optional"`
	Tags         []string `hcl:"tags,optional"`
	Target       string   `hcl:"target,optional"`
	Verify       bool     `hcl:"verify,optional"`
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
	KeepLast    int `hcl:"keep_last,optional"`
	KeepDaily   int `hcl:"keep_daily,optional"`
	KeepHourly  int `hcl:"keep_hourly,optional"`
	KeepMonthly int `hcl:"keep_monthly,optional"`
	KeepWeekly  int `hcl:"keep_weekly,optional"`
	KeepYearly  int `hcl:"keep_yearly,optional"`

	KeepWithin        time.Duration `hcl:"keep_within,optional"`
	KeepWithinDaily   time.Duration `hcl:"keep_within_daily,optional"`
	KeepWithinHourly  time.Duration `hcl:"keep_within_hourly,optional"`
	KeepWithinMonthly time.Duration `hcl:"keep_within_monthly,optional"`
	KeepWithinWeekly  time.Duration `hcl:"keep_within_weekly,optional"`
	KeepWithinYearly  time.Duration `hcl:"keep_within_yearly,optional"`

	Compact              bool     `hcl:"compact,optional"`
	DryRun               bool     `hcl:"dry_run,optional"`
	GroupBy              string   `hcl:"group_by,optional"`
	Hosts                []string `hcl:"hosts,optional"`
	Paths                []string `hcl:"paths,optional"`
	Tags                 []string `hcl:"tags,optional"`
	KeepTags             []string `hcl:"keep_tags,optional"`
	UnsafeAllowRemoveAll bool     `hcl:"unsafe_allow_remove_all,optional"`

	Prune               bool   `hcl:"prune,optional"`
	MaxUnused           string `hcl:"max_unused,optional"`
	MaxRepackSize       string `hcl:"max_repack_size,optional"`
	RepackCacheableOnly bool   `hcl:"repack_cacheable_only,optional"`
	RepackSmall         bool   `hcl:"repack_small,optional"`
	RepackUncompressed  bool   `hcl:"repack_uncompressed,optional"`
	RepackSmallerThan   string `hcl:"repack_smaller_than,optional"`
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
	FromInsecureNoPassword bool     `hcl:"from_insecure_no_password,optional"`
	FromKeyHint            string   `hcl:"from_key_hint,optional"`
	FromPasswordCommand    string   `hcl:"from_password_command,optional"`
	FromPasswordFile       string   `hcl:"from_password_file,optional"`
	FromRepo               string   `hcl:"from_repo,optional"`
	FromRepositoryFile     string   `hcl:"from_repository_file,optional"`
	Hosts                  []string `hcl:"hosts,optional"`
	Paths                  []string `hcl:"paths,optional"`
	Tags                   []string `hcl:"tags,optional"`
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

type InitOpts struct {
	CopyChunkerParams      bool   `hcl:"copy_chunker_params,optional"`
	FromInsecureNoPassword bool   `hcl:"from_insecure_no_password,optional"`
	FromKeyHint            string `hcl:"from_key_hint,optional"`
	FromPasswordCommand    string `hcl:"from_password_command,optional"`
	FromPasswordFile       string `hcl:"from_password_file,optional"`
	FromRepo               string `hcl:"from_repo,optional"`
	FromRepositoryFile     string `hcl:"from_repository_file,optional"`
	RepositoryVersion      string `hcl:"repository_version,optional"`
}

// ToArgs returns the structs arguments as a slice of strings.
func (io InitOpts) ToArgs() (args []string) {
	args = utils.MaybeAddArgBool(args, "--copy-chunker-params", io.CopyChunkerParams)
	args = utils.MaybeAddArgBool(args, "--from-insecure-no-password", io.FromInsecureNoPassword)
	args = utils.MaybeAddArgString(args, "--from-key-hint", io.FromKeyHint)
	args = utils.MaybeAddArgString(args, "--from-password-command", io.FromPasswordCommand)
	args = utils.MaybeAddArgString(args, "--from-password-file", io.FromPasswordFile)
	args = utils.MaybeAddArgString(args, "--from-repo", io.FromRepo)
	args = utils.MaybeAddArgString(args, "--from-repository-file", io.FromRepositoryFile)
	args = utils.MaybeAddArgString(args, "--repository-version", io.RepositoryVersion)

	return args
}

type ResticGlobalOpts struct {
	CaCertFile        string            `hcl:"cacert,optional"`
	CacheDir          string            `hcl:"cache_dir,optional"`
	PasswordFile      string            `hcl:"password_file,optional"`
	TLSClientCertFile string            `hcl:"tls_client_cert,optional"`
	LimitDownload     int               `hcl:"limit_download,optional"`
	LimitUpload       int               `hcl:"limit_upload,optional"`
	VerboseLevel      int               `hcl:"verbose_level,optional"`
	Options           map[string]string `hcl:"options,optional"`
	CleanupCache      bool              `hcl:"cleanup_cache,optional"`
	InsecureTLS       bool              `hcl:"insecure_tls,optional"`
	NoCache           bool              `hcl:"no_cache,optional"`
	NoLock            bool              `hcl:"no_lock,optional"`
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

func (rcmd Restic) InitRepo(initOpts InitOpts) error {
	_, err := rcmd.RunRestic("init", initOpts)
	return err
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

func (rcmd Restic) EnsureInit(initOpts InitOpts) error {
	if err := rcmd.Snapshots(); errors.Is(err, ErrRepoNotFound) {
		err := rcmd.InitRepo(initOpts)

		return err
	}

	return nil
}
