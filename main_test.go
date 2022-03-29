package main_test

import (
	"fmt"
	"os"
	"testing"
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
