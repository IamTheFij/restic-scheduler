package main_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	main "git.iamthefij.com/iamthefij/restic-scheduler"
	"github.com/stretchr/testify/assert"
)

func TestJobResultFormat(t *testing.T) {
	t.Parallel()

	result := main.JobResult{
		JobName:   "TestJob",
		JobType:   "backup",
		Success:   true,
		LastError: nil,
		Message:   "",
	}

	formatted := result.Format()
	assert.Contains(t, formatted, "TestJob")
	assert.Contains(t, formatted, "backup")
	assert.Contains(t, formatted, "true")
}

func TestJobComplete(t *testing.T) {
	t.Parallel()

	// Create a test job result
	result := main.JobResult{
		JobName:   "TestCompleteJob",
		JobType:   "backup",
		Success:   true,
		LastError: nil,
		Message:   "",
	}

	// Since JobComplete modifies global state, it's hard to test directly
	// This is more of a smoke test to ensure it doesn't panic

	// Call JobComplete
	main.JobComplete(result)
}

func TestHealthHandleFunc(t *testing.T) {
	t.Parallel()

	// Test general health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(main.HealthHandleFunc)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())

	// Test job-specific health endpoint
	// First register a job result
	result := main.JobResult{
		JobName:   "TestHealthJob",
		JobType:   "backup",
		Success:   true,
		LastError: nil,
		Message:   "Test job successful",
	}
	main.JobComplete(result)

	// Now query that job
	req, err = http.NewRequest("GET", "/health?job=TestHealthJob", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Decode the response
	var responseResult main.JobResult

	err = json.Unmarshal(rr.Body.Bytes(), &responseResult)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "TestHealthJob", responseResult.JobName)
	assert.Equal(t, "backup", responseResult.JobType)
	assert.True(t, responseResult.Success)
}

func TestHealthHandleFuncUnknownJob(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "/health?job=NonExistentJob", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(main.HealthHandleFunc)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "Unknown job")
}

func TestHealthHandleFuncFailedJob(t *testing.T) {
	t.Parallel()

	// Register a failed job result
	result := main.JobResult{
		JobName:   "TestFailedJob",
		JobType:   "backup",
		Success:   false,
		LastError: nil,
		Message:   "Job failed with error",
	}
	main.JobComplete(result)

	// Query the failed job
	req, err := http.NewRequest("GET", "/health?job=TestFailedJob", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(main.HealthHandleFunc)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	// Decode the response
	var responseResult main.JobResult

	err = json.Unmarshal(rr.Body.Bytes(), &responseResult)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "TestFailedJob", responseResult.JobName)
	assert.False(t, responseResult.Success)
	assert.NotEmpty(t, responseResult.Message)
}
