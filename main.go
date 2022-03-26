package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

var (
	// version of restic-scheduler being run.
	version = "dev"
)

func parseConfig(path string) ([]Job, error) {
	var config Config

	if err := hclsimple.DecodeFile(path, nil, &config); err != nil {
		return nil, fmt.Errorf("%s: Failed to decode file: %w", path, err)
	}

	if len(config.Jobs) == 0 {
		log.Printf("%s: No jobs defined in file", path)

		return []Job{}, nil
	}

	for _, job := range config.Jobs {
		if err := job.Validate(); err != nil {
			return nil, fmt.Errorf("%s: Invalid job: %w", path, err)
		}
	}

	return config.Jobs, nil
}

func readJobs(paths []string) ([]Job, error) {
	allJobs := []Job{}

	for _, path := range paths {
		jobs, err := parseConfig(path)
		if err != nil {
			return nil, err
		}

		if jobs != nil {
			allJobs = append(allJobs, jobs...)
		}
	}

	return allJobs, nil
}

type Set map[string]bool

func NewSetFrom(l []string) Set {
	s := make(Set)
	for _, l := range l {
		s[l] = true
	}

	return s
}

func runBackupJobs(jobs []Job, names []string) error {
	nameSet := NewSetFrom(names)
	_, runAll := nameSet["all"]

	for _, job := range jobs {
		if _, found := nameSet[job.Name]; runAll || found {
			if err := job.RunBackup(); err != nil {
				return err
			}
		}
	}

	return nil
}

func runRestoreJobs(jobs []Job, names []string) error {
	nameSet := NewSetFrom(names)
	_, runAll := nameSet["all"]

	for _, job := range jobs {
		if _, found := nameSet[job.Name]; runAll || found {
			if err := job.RunRestore(); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	showVersion := flag.Bool("version", false, "Display the version and exit")
	backup := flag.String("backup", "", "Run backup jobs now. Names are comma separated and `all` will run all.")
	restore := flag.String("restore", "", "Run restore jobs now. Names are comma separated and `all` will run all.")
	once := flag.Bool("once", false, "Run jobs specified using -backup and -restore once and exit")
	flag.StringVar(&JobBaseDir, "base-dir", JobBaseDir, "Base dir to create intermediate job files like SQL dumps.")
	flag.Parse()

	// Print version if flag is provided
	if *showVersion {
		fmt.Println("restic-scheduler version:", version)

		return
	}

	if flag.NArg() == 0 {
		log.Fatalf("Requires a path to a job file, but found none")
	}

	jobs, err := readJobs(flag.Args())
	if err != nil {
		log.Fatalf("Failed to read jobs from files: %v", err)
	}

	if len(jobs) == 0 {
		log.Fatal("No jobs found in provided configuration")
	}

	// Run specified backup jobs
	if err := runBackupJobs(jobs, strings.Split(*backup, ",")); err != nil {
		log.Fatalf("Failed running backup jobs: %v", err)
	}

	// Run specified restore jobs
	if err := runRestoreJobs(jobs, strings.Split(*restore, ",")); err != nil {
		log.Fatalf("Failed running backup jobs: %v", err)
	}

	// Exit if only running once
	if *once {
		return
	}

	// TODO: Add healthcheck handler using Job.Healthy()
	if err := ScheduleAndRunJobs(jobs); err != nil {
		log.Fatalf("failed running jobs: %v", err)
	}
}
