package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var defaultFlags = log.LstdFlags | log.Lmsgprefix

type ResticCmd struct {
	LogPrefix  string
	Repo       string
	Env        map[string]string
	Passphrase string
}

func (rcmd ResticCmd) BuildEnv() []string {
	rcmd.Env["RESTIC_PASSWORD"] = rcmd.Passphrase

	envList := []string{}

	for name, value := range rcmd.Env {
		envList = append(envList, fmt.Sprintf("%s=%s", name, value))
	}

	return envList
}

func (rcmd ResticCmd) RunRestic(args []string) error {
	cmd := exec.Command("restic", args...)

	cmd.Stdout = rcmd.Logger().Writer()
	cmd.Stderr = cmd.Stdout
	cmd.Env = rcmd.BuildEnv()

	err := cmd.Run()

	return err
}

func (rcmd ResticCmd) Logger() *log.Logger {
	logger := log.New(os.Stderr, rcmd.LogPrefix, defaultFlags)

	return logger
}

func (rcmd ResticCmd) Backup(path string, args []string) error {
	args = append([]string{"--repo", rcmd.Repo, "backup"}, args...)
	args = append(args, path)

	err := rcmd.RunRestic(args)

	return err
}

type ForgetOpts struct {
	KeepLast    int
	KeepHourly  int
	KeepDaily   int
	KeepWeekly  int
	KeepMonthly int
	KeepYearly  int

	KeepWithin        time.Duration
	KeepWithinHourly  time.Duration
	KeepWithinDaily   time.Duration
	KeepWithinWeekly  time.Duration
	KeepWithinMonthly time.Duration
	KeepWithinYearly  time.Duration

	Tags     []string
	KeepTags []string

	Prune bool
}

func (fo ForgetOpts) ToArgs() []string {
	args := []string{}

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

func (rcmd ResticCmd) Cleanup(forgetOpts ForgetOpts) error {
	args := append([]string{"--repo", rcmd.Repo, "forget"}, forgetOpts.ToArgs()...)

	err := rcmd.RunRestic(args)

	return err
}

func (rcmd ResticCmd) Check() error {
	args := []string{"--repo", rcmd.Repo, "check"}

	err := rcmd.RunRestic(args)

	return err
}
