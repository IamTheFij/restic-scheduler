package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

var (
	// version of restic-scheduler being run.
	version = "dev"
)

func main() {
	showVersion := flag.Bool("version", false, "Display the version of minitor and exit")
	flag.Parse()

	// Print version if flag is provided
	if *showVersion {
		fmt.Println("restic-scheduler version:", version)

		return
	}

	var config Config

	args := flag.Args()
	if len(args) == 0 {
		log.Fatalf("Requires a path to a job file, but found none")
	}

	if err := hclsimple.DecodeFile(args[0], nil, &config); err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	log.Printf("Configuration is %#v", config)

	if len(config.Jobs) == 0 {
		log.Fatalf("No jobs defined in config")
	}

	for _, job := range config.Jobs {
		if err := job.RunBackup(); err != nil {
			log.Fatalf("%v", err)
		}
	}
}
