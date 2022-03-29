package main_test

import (
	"fmt"
	"os"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
)

const MinCoverage = 0.5

func TestMain(m *testing.M) {
	testResult := m.Run()

	if testResult == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < MinCoverage {
			fmt.Printf("Tests passed but coverage failed at %0.2f and minimum to pass is %0.2f\n", c, MinCoverage)

			testResult = -1
		}
	}

	os.Exit(testResult)
}

func TestReadJobs(t *testing.T) {
	t.Parallel()

	jobs, err := main.ReadJobs([]string{"./test/sample.hcl"})

	if err != nil {
		t.Errorf("Unexpected error reading jobs: %v", err)
	}

	if len(jobs) == 0 {
		t.Error("Expected read jobs but found none")
	}
}
