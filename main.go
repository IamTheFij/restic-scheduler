package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

var (
	// version of restic-scheduler being run.
	version        = "dev"
	ErrJobNotFound = errors.New("jobs not found")
)

func ReadJobs(paths []string) ([]Job, error) {
	allJobs := []Job{}

	for _, path := range paths {
		jobs, err := ParseConfig(path)
		if err != nil {
			return nil, err
		}

		if jobs != nil {
			allJobs = append(allJobs, jobs...)
		}
	}

	if len(allJobs) == 0 {
		return allJobs, fmt.Errorf("no jobs found in provided configuration: %w", ErrJobNotFound)
	}

	return allJobs, nil
}

// FilterJobs filters a list of jobs by a list of names.
func FilterJobs(jobs []Job, names []string) ([]Job, error) {
	nameSet := NewSetFrom(names)
	if nameSet.Contains("all") {
		return jobs, nil
	}

	filteredJobs := []Job{}

	for _, job := range jobs {
		if nameSet.Contains(job.Name) {
			filteredJobs = append(filteredJobs, job)

			delete(nameSet, job.Name)
		}
	}

	var err error
	if len(nameSet) > 0 {
		err = fmt.Errorf("%w: %v", ErrJobNotFound, nameSet)
	}

	return filteredJobs, err
}

func runBackupJobs(jobs []Job, names string) error {
	if names == "" {
		return nil
	}

	namesSlice := strings.Split(names, ",")

	if len(namesSlice) == 0 {
		return nil
	}

	jobs, filterJobErr := FilterJobs(jobs, namesSlice)
	for _, job := range jobs {
		if err := job.RunBackup(); err != nil {
			return err
		}
	}

	return filterJobErr
}

func runRestoreJobs(jobs []Job, names string, snapshot string) error {
	if names == "" {
		return nil
	}

	namesSlice := strings.Split(names, ",")

	if len(namesSlice) == 0 {
		return nil
	}

	jobs, filterJobErr := FilterJobs(jobs, namesSlice)
	for _, job := range jobs {
		if err := job.RunRestore(snapshot); err != nil {
			return err
		}
	}

	return filterJobErr
}

func runUnlockJobs(jobs []Job, names string) error {
	if names == "" {
		return nil
	}

	namesSlice := strings.Split(names, ",")

	if len(namesSlice) == 0 {
		return nil
	}

	jobs, filterJobErr := FilterJobs(jobs, namesSlice)
	for _, job := range jobs {
		if err := job.NewRestic().Unlock(UnlockOpts{RemoveAll: true}); err != nil {
			return err
		}
	}

	return filterJobErr
}

type Flags struct {
	showVersion        bool
	backup             string
	restore            string
	unlock             string
	restoreSnapshot    string
	once               bool
	healthCheckAddr    string
	metricsPushGateway string
}

func readFlags() Flags {
	flags := Flags{} //nolint:exhaustruct
	flag.BoolVar(&flags.showVersion, "version", false, "Display the version and exit")
	flag.StringVar(&flags.backup, "backup", "", "Run backup jobs now. Names are comma separated. `all` will run all.")
	flag.StringVar(&flags.restore, "restore", "", "Run restore jobs now. Names are comma separated. `all` will run all.")
	flag.StringVar(&flags.unlock, "unlock", "", "Unlock job repos now. Names are comma separated. `all` will run all.")
	flag.BoolVar(&flags.once, "once", false, "Run jobs specified using -backup and -restore once and exit")
	flag.StringVar(&flags.healthCheckAddr, "addr", "0.0.0.0:8080", "address to bind health check API")
	flag.StringVar(&flags.metricsPushGateway, "push-gateway", "", "url of push gateway service for batch runs (optional)")
	flag.StringVar(&JobBaseDir, "base-dir", JobBaseDir, "Base dir to create intermediate job files like SQL dumps.")
	flag.StringVar(&flags.restoreSnapshot, "snapshot", "latest", "the snapshot to restore")
	flag.Parse()

	return flags
}

func runSpecifiedJobs(jobs []Job, backupJobs, restoreJobs, unlockJobs, snapshot string) error {
	// Run specified job unlocks
	if err := runUnlockJobs(jobs, unlockJobs); err != nil {
		return fmt.Errorf("failed running unlock for jobs: %w", err)
	}

	// Run specified backup jobs
	if err := runBackupJobs(jobs, backupJobs); err != nil {
		return fmt.Errorf("failed running backup jobs: %w", err)
	}

	// Run specified restore jobs
	if err := runRestoreJobs(jobs, restoreJobs, snapshot); err != nil {
		return fmt.Errorf("failed running restore jobs: %w", err)
	}

	return nil
}

func maybePushMetrics(metricsPushGateway string) error {
	if metricsPushGateway != "" {
		fmt.Println("Pushing metrics to push gateway")

		if err := Metrics.PushToGateway(metricsPushGateway); err != nil {
			return fmt.Errorf("failed pushing metrics after jobs run: %w", err)
		}
	}

	return nil
}

// printVersion prints the scheduler version and the restic binary version if installed on the system.
func printVersion() {
	fmt.Println("restic-scheduler version:", version)
	// Also display restic binary version
	if resticVerOut, err := exec.Command("restic", "version").Output(); err == nil {
		fmt.Printf("restic binary version: %s\n", strings.TrimSpace(string(resticVerOut)))
	} else {
		if _, err := exec.LookPath("restic"); err != nil {
			fmt.Printf("restic binary version: unknown, restic missing from $PATH")
			return
		} else {
			fmt.Println("failed to get restic binary version:", err)
		}
	}
}

func main() {
	flags := readFlags()

	// Print version if flag is provided
	if flags.showVersion {
		printVersion()
		return
	}

	if _, err := exec.LookPath("restic"); err != nil {
		log.Fatalf("Could not find restic in path. Make sure it's installed")
	}

	if flag.NArg() == 0 {
		log.Fatalf("Requires a path to a job file, but found none")
	}

	// Capture the job file paths once here in main so reloads use the same input
	jobPaths := flag.Args()

	jobs, err := ReadJobs(jobPaths)
	if err != nil {
		log.Fatalf("Failed to read jobs from files: %v", err)
	}

	if err := runSpecifiedJobs(jobs, flags.backup, flags.restore, flags.unlock, flags.restoreSnapshot); err != nil {
		log.Fatal(err)
	}

	// Exit if only running once
	if flags.once {
		if err := maybePushMetrics(flags.metricsPushGateway); err != nil {
			log.Fatal(err)
		}

		return
	}

	// Create scheduler and start it with the initial job set.
	sched := NewScheduler()
	if err := sched.Start(jobs); err != nil {
		log.Fatalf("failed to start scheduler: %v", err)
	}

	// Start HTTP handlers and provide the scheduler so /active can report live jobs.
	go func() {
		_ = RunHTTPHandlers(flags.healthCheckAddr, sched)
	}()

	for _, job := range jobs {
		log.Printf("Refreshing metrics for job %s", job.Name)
		job.RefreshMetrics()
	}

	// Main owns signal handling and config reload.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGHUP:
			// Reload config and apply via scheduler.ReplaceJobs
			log.Println("Received SIGHUP; reloading configuration...")

			newJobs, readErr := ReadJobs(jobPaths)
			if readErr != nil {
				log.Printf("Failed to reload jobs: %v; keeping existing schedule", readErr)
				continue
			}

			// Refresh metrics for the new job set before replacing to populate gauges.
			for _, j := range newJobs {
				log.Printf("Refreshing metrics for job %s", j.Name)
				j.RefreshMetrics()
			}

			if err := sched.ReplaceJobs(newJobs); err != nil {
				log.Printf("Failed to apply reloaded jobs: %v; keeping previous schedule", err)
				continue
			}

			log.Println("Configuration reload successful")

		case syscall.SIGINT:
			// Immediate stop: do not wait for running jobs.
			log.Println("Received SIGINT; stopping immediately")
			sched.StopNow()

			return
		case syscall.SIGTERM, syscall.SIGQUIT:
			// Graceful stop: wait for running jobs to finish.
			log.Println("Received termination signal; stopping gracefully")
			sched.StopGraceful()

			return
		}
	}
}
