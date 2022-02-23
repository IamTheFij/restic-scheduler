package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type CommandOptions interface {
	ToArgs() []string
}

type NoOpts struct{}

func (NoOpts) ToArgs() []string {
	return []string{}
}

type ResticGlobalOpts struct {
	CaCertFile        string `hcl:"CaCertFile,optional"`
	CacheDir          string `hcl:"CacheDir,optional"`
	PasswordFile      string `hcl:"PasswordFile,optional"`
	TLSClientCertFile string `hcl:"TlsClientCertFile,optional"`
	LimitDownload     int    `hcl:"LimitDownload,optional"`
	LimitUpload       int    `hcl:"LimitUpload,optional"`
	VerboseLevel      int    `hcl:"VerboseLevel,optional"`
	CleanupCache      bool   `hcl:"CleanupCache,optional"`
	NoCache           bool   `hcl:"NoCache,optional"`
	NoLock            bool   `hcl:"NoLock,optional"`
}

// nolint:cyclop
func (glo ResticGlobalOpts) ToArgs() (args []string) {
	if glo.CaCertFile != "" {
		args = append(args, "--cacert", glo.CaCertFile)
	}

	if glo.CacheDir != "" {
		args = append(args, "--cache-dir", glo.CacheDir)
	}

	if glo.PasswordFile != "" {
		args = append(args, "--password-file", glo.PasswordFile)
	}

	if glo.TLSClientCertFile != "" {
		args = append(args, "--tls-client-cert", glo.TLSClientCertFile)
	}

	if glo.LimitDownload > 0 {
		args = append(args, "--limit-download", fmt.Sprint(glo.LimitDownload))
	}

	if glo.LimitUpload > 0 {
		args = append(args, "--limit-upload", fmt.Sprint(glo.LimitUpload))
	}

	if glo.VerboseLevel > 0 {
		args = append(args, "--verbose", fmt.Sprint(glo.VerboseLevel))
	}

	if glo.CleanupCache {
		args = append(args, "--cleanup-cache")
	}

	if glo.NoCache {
		args = append(args, "--no-cache")
	}

	if glo.NoLock {
		args = append(args, "--no-lock")
	}

	return args
}

type ResticCmd struct {
	Logger     *log.Logger
	Repo       string
	Env        map[string]string
	Passphrase string
	GlobalOpts *ResticGlobalOpts
	Cwd        string
}

func (rcmd ResticCmd) BuildEnv() []string {
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

func (rcmd ResticCmd) RunRestic(command string, options CommandOptions, commandArgs ...string) error {
	args := []string{}
	if rcmd.GlobalOpts != nil {
		args = rcmd.GlobalOpts.ToArgs()
	}

	args = append(args, "--repo", rcmd.Repo, command)
	args = append(args, options.ToArgs()...)
	args = append(args, commandArgs...)

	cmd := exec.Command("restic", args...)

	cmd.Stdout = NewLogWriter(rcmd.Logger)
	cmd.Stderr = cmd.Stdout
	cmd.Env = rcmd.BuildEnv()
	cmd.Dir = rcmd.Cwd

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running restic: %w", err)
	}

	return nil
}

type BackupOpts struct {
	Exclude []string `hcl:"Exclude,optional"`
	Include []string `hcl:"Include,optional"`
	Tags    []string `hcl:"Tags,optional"`
	Host    string   `hcl:"Host,optional"`
}

func (bo BackupOpts) ToArgs() (args []string) {
	for _, tag := range bo.Tags {
		args = append(args, "--tag", tag)
	}

	if bo.Host != "" {
		args = append(args, "--host", bo.Host)
	}

	return
}

func (rcmd ResticCmd) Backup(files []string, options *BackupOpts) error {
	if options == nil {
		options = &BackupOpts{} // nolint:exhaustivestruct
	}

	err := rcmd.RunRestic("backup", options, files...)

	return err
}

type RestoreOpts struct {
	Exclude []string `hcl:"Exclude,optional"`
	Host    []string `hcl:"Host,optional"`
	Include []string `hcl:"Include,optional"`
	Path    string   `hcl:"Path,optional"`
	Tags    []string `hcl:"Tags,optional"`
	Target  string   `hcl:"Target,optional"`
	Verify  bool     `hcl:"Verify,optional"`
}

func (ro RestoreOpts) ToArgs() (args []string) {
	for _, exclude := range ro.Exclude {
		args = append(args, "--exclude", exclude)
	}

	for _, include := range ro.Include {
		args = append(args, "--include", include)
	}

	for _, host := range ro.Host {
		args = append(args, "--host", host)
	}

	if ro.Path != "" {
		args = append(args, "--path", ro.Path)
	}

	for _, tag := range ro.Tags {
		args = append(args, "--tag", tag)
	}

	if ro.Target != "" {
		args = append(args, "--target", ro.Target)
	}

	if ro.Verify {
		args = append(args, "--verify")
	}

	return
}

func (rcmd ResticCmd) Restore(snapshot string, opts *RestoreOpts) error {
	if opts == nil {
		opts = &RestoreOpts{} // nolint:exhaustivestruct
	}

	err := rcmd.RunRestic("restore", opts, snapshot)

	return err
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

	Tags     []string `hcl:"Tags,optional"`
	KeepTags []string `hcl:"KeepTags,optional"`

	Prune bool `hcl:"Prune,optional"`
}

// nolint:funlen,cyclop
func (fo ForgetOpts) ToArgs() (args []string) {
	// Add keep-*
	if fo.KeepLast > 0 {
		args = append(args, "--keep-last", fmt.Sprint(fo.KeepLast))
	}

	if fo.KeepHourly > 0 {
		args = append(args, "--keep-hourly", fmt.Sprint(fo.KeepHourly))
	}

	if fo.KeepDaily > 0 {
		args = append(args, "--keep-daily", fmt.Sprint(fo.KeepDaily))
	}

	if fo.KeepWeekly > 0 {
		args = append(args, "--keep-weekly", fmt.Sprint(fo.KeepWeekly))
	}

	if fo.KeepMonthly > 0 {
		args = append(args, "--keep-monthly", fmt.Sprint(fo.KeepMonthly))
	}

	if fo.KeepYearly > 0 {
		args = append(args, "--keep-yearly", fmt.Sprint(fo.KeepYearly))
	}

	if fo.KeepWithin > 0 {
		args = append(args, "--keep-within", fmt.Sprint(fo.KeepWithin))
	}

	// Add keep-within-*
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
	if len(fo.Tags) > 0 {
		args = append(args, "--tag", strings.Join(fo.Tags, ","))
	}

	if len(fo.KeepTags) > 0 {
		args = append(args, "--keep-tag", strings.Join(fo.Tags, ","))
	}

	// Add prune options

	if fo.Prune {
		args = append(args, "--prune")
	}

	return args
}

func (rcmd ResticCmd) Forget(forgetOpts *ForgetOpts) error {
	if forgetOpts == nil {
		forgetOpts = &ForgetOpts{} // nolint:exhaustivestruct
	}

	err := rcmd.RunRestic("forget", forgetOpts)

	return err
}

func (rcmd ResticCmd) Check() error {
	err := rcmd.RunRestic("check", NoOpts{})

	return err
}

func (rcmd ResticCmd) EnsureInit() error {
	if err := rcmd.RunRestic("snapshots", NoOpts{}); err != nil {
		return rcmd.RunRestic("init", NoOpts{})
	}

	return nil
}
