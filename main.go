package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

var (
	// version of restic-scheduler being run
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

	if err := hclsimple.DecodeFile("config.hcl", nil, &config); err != nil {
		log.Fatalf("Failed to load configuration: %s", err)
	}

	log.Printf("Configuration is %#v", config)
}
