package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
)

var (
	jobResultsLock = sync.Mutex{}
	jobResults     = map[string]JobResult{}
)

type JobResult struct {
	JobName   string
	JobType   string
	Success   bool
	LastError error
	Message   string
}

func (r JobResult) Format() string {
	return fmt.Sprintf("%s %s ok? %v\n\n%+v", r.JobName, r.JobType, r.Success, r.LastError)
}

func JobComplete(result JobResult) {
	fmt.Printf("Completed job %+v\n", result)

	jobResultsLock.Lock()
	jobResults[result.JobName] = result
	jobResultsLock.Unlock()
}

func writeJobResult(writer http.ResponseWriter, jobName string) {
	writer.Header().Set("Content-Type", "application/json")

	if jobResult, ok := jobResults[jobName]; ok {
		if !jobResult.Success {
			// Set a 503 status code if the last job run was not successful
			writer.WriteHeader(http.StatusServiceUnavailable)
		}

		// Set message from LastError if available
		if jobResult.LastError != nil {
			jobResult.Message = jobResult.LastError.Error()
		}

		// Write the job result as JSON
		if err := json.NewEncoder(writer).Encode(jobResult); err != nil {
			// If encoding fails, write an error message
			_, _ = writer.Write(fmt.Appendf(nil, "failed writing json for %s", jobResult.Format()))
		}
	} else {
		// Job not found
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte("{\"Message\": \"Unknown job\"}"))
	}
}

func healthHandleFunc(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	if jobName, ok := query["job"]; ok {
		writeJobResult(writer, jobName[0])

		return
	}

	_, _ = writer.Write([]byte("ok"))
}

func RunHTTPHandlers(addr string) error {
	http.HandleFunc("/health", healthHandleFunc)
	http.Handle("/metrics", promhttp.HandlerFor(
		Metrics.Registry,
		promhttp.HandlerOpts{Registry: Metrics.Registry}, //nolint:exhaustruct
	))

	return fmt.Errorf("error on http server: %w", http.ListenAndServe(addr, nil)) //#nosec: g114
}

func ScheduleAndRunJobs(jobs []Job) error {
	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	scheduler := cron.New()

	for _, job := range jobs {
		fmt.Println("Scheduling", job.Name)

		if _, err := scheduler.AddJob(job.Schedule, job); err != nil {
			return fmt.Errorf("error scheduling job %s: %w", job.Name, err)
		}
	}

	scheduler.Start()

	switch <-signalChan {
	case syscall.SIGINT:
		fmt.Println("Stopping now...")

		defer scheduler.Stop()

		return nil
	case syscall.SIGTERM:
		fallthrough
	case syscall.SIGQUIT:
		// Wait for all jobs to complete
		fmt.Println("Stopping after running jobs complete...")

		defer func() {
			ctx := scheduler.Stop()
			<-ctx.Done()

			fmt.Println("All jobs successfully stopped")
		}()

		return nil
	}

	return nil
}
