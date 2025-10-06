package main_test

import (
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
	"github.com/stretchr/testify/assert"
)

func TestInitMetrics(t *testing.T) {
	t.Parallel()

	metrics := main.InitMetrics()

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.Registry)
	assert.NotNil(t, metrics.JobStartTime)
	assert.NotNil(t, metrics.JobFailureCount)
	assert.NotNil(t, metrics.SnapshotCurrentCount)
	assert.NotNil(t, metrics.SnapshotLatestTime)
}

// PushToGateway is difficult to test directly without mocking HTTP responses
// In a real test environment we would use httptest.Server to mock responses
