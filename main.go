package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	// version of restic-scheduler being run.
	version        = "dev"
	ErrJobNotFound = errors.New("jobs not found")
)

func ParseConfig(path string) ([]Job, error) {
	var config Config

	ctx := hcl.EvalContext{
		Variables: nil,
		Functions: map[string]function.Function{
			"env": function.New(&function.Spec{
				Params: []function.Parameter{{
					Name: "var",
					Type: cty.String,
				}},
				VarParam: nil,
				Type:     function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					return cty.StringVal(os.Getenv(args[0].AsString())), nil
				},
			}),
			"readfile": function.New(&function.Spec{
				Params: []function.Parameter{{
					Name: "path",
					Type: cty.String,
				}},
				VarParam: nil,
				Type:     function.StaticReturnType(cty.String),
				Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
					content, err := os.ReadFile(args[0].AsString())
					if err != nil {
						return cty.StringVal(""), err
					}

					return cty.StringVal(string(content)), nil
				},
			}),
		},
	}

	if err := hclsimple.DecodeFile(path, &ctx, &config); err != nil {
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
		return allJobs, fmt.Errorf("No jobs found in provided configuration: %w", ErrJobNotFound)
	}

	return allJobs, nil
}

type Set map[string]bool

func (s Set) Contains(key string) bool {
	_, contains := s[key]

	return contains
}

func NewSetFrom(l []string) Set {
	s := make(Set)
	for _, l := range l {
		s[l] = true
	}

	return s
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

func runRestoreJobs(jobs []Job, names string) error {
	namesSlice := strings.Split(names, ",")

	if len(namesSlice) == 0 {
		return nil
	}

	jobs, filterJobErr := FilterJobs(jobs, namesSlice)
	for _, job := range jobs {
		if err := job.RunRestore(); err != nil {
			return err
		}
	}

	return filterJobErr
}

type Flags struct {
	showVersion        bool
	backup             string
	restore            string
	once               bool
	healthCheckAddr    string
	metricsPushGateway string
}

func readFlags() Flags {
	flags := Flags{} //nolint:exhaustruct
	flag.BoolVar(&flags.showVersion, "version", false, "Display the version and exit")
	flag.StringVar(&flags.backup, "backup", "", "Run backup jobs now. Names are comma separated. `all` will run all.")
	flag.StringVar(&flags.restore, "restore", "", "Run restore jobs now. Names are comma separated. `all` will run all.")
	flag.BoolVar(&flags.once, "once", false, "Run jobs specified using -backup and -restore once and exit")
	flag.StringVar(&flags.healthCheckAddr, "addr", "0.0.0.0:8080", "address to bind health check API")
	flag.StringVar(&flags.metricsPushGateway, "push-gateway", "", "url of push gateway service for batch runs (optional)")
	flag.StringVar(&JobBaseDir, "base-dir", JobBaseDir, "Base dir to create intermediate job files like SQL dumps.")
	flag.Parse()

	return flags
}

func runSpecifiedJobs(jobs []Job, backupJobs, restoreJobs string) error {
	// Run specified backup jobs
	if err := runBackupJobs(jobs, backupJobs); err != nil {
		return fmt.Errorf("Failed running backup jobs: %w", err)
	}

	// Run specified restore jobs
	if err := runRestoreJobs(jobs, restoreJobs); err != nil {
		return fmt.Errorf("Failed running restore jobs: %w", err)
	}

	return nil
}

func maybePushMetrics(metricsPushGateway string) error {
	if metricsPushGateway != "" {
		if err := Metrics.PushToGateway(metricsPushGateway); err != nil {
			return fmt.Errorf("Failed pushing metrics after jobs run: %w", err)
		}
	}

	return nil
}

func main() {
	flags := readFlags()

	// Print version if flag is provided
	if flags.showVersion {
		fmt.Println("restic-scheduler version:", version)

		return
	}

	if _, err := exec.LookPath("restic"); err != nil {
		log.Fatalf("Could not find restic in path. Make sure it's installed")
	}

	if flag.NArg() == 0 {
		log.Fatalf("Requires a path to a job file, but found none")
	}

	jobs, err := ReadJobs(flag.Args())
	if err != nil {
		log.Fatalf("Failed to read jobs from files: %v", err)
	}

	if err := runSpecifiedJobs(jobs, flags.backup, flags.restore); err != nil {
		log.Fatal(err)
	}

	// Exit if only running once
	if flags.once {
		if err := maybePushMetrics(flags.metricsPushGateway); err != nil {
			log.Fatal(err)
		}

		return
	}

	go func() {
		_ = RunHTTPHandlers(flags.healthCheckAddr)
	}()

	// TODO: Add healthcheck handler using Job.Healthy()
	if err := ScheduleAndRunJobs(jobs); err != nil {
		log.Fatalf("failed running jobs: %v", err)
	}
}
