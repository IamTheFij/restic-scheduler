package utils

import (
	"testing"

	"github.com/go-test/deep"
)

func AssertEqual(t *testing.T, message string, expected, actual any) bool {
	t.Helper()

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Errorf("%s: %v", message, diff)

		return false
	}

	return true
}

func AssertEqualFail(t *testing.T, message string, expected, actual any) {
	t.Helper()

	if !AssertEqual(t, message, expected, actual) {
		t.FailNow()
	}
}
