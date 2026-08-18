package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"git.iamthefij.com/iamthefij/restic-scheduler/metrics"
)

func TestInitMetrics(t *testing.T) {
	t.Parallel()

	metrics := metrics.InitMetrics()

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.Registry)
	assert.NotNil(t, metrics.JobStartTime)
	assert.NotNil(t, metrics.JobFailureCount)
	assert.NotNil(t, metrics.SnapshotCurrentCount)
	assert.NotNil(t, metrics.SnapshotLatestTime)
}

// PushToGateway is difficult to test directly without mocking HTTP responses
// In a real test environment we would use httptest.Server to mock responses
