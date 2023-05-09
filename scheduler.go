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

var jobResultsLock = sync.Mutex{}
var jobResults = map[string]JobResult{}

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
	if jobResult, ok := jobResults[jobName]; ok {
		if !jobResult.Success {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}

		jobResult.Message = jobResult.LastError.Error()
		if err := json.NewEncoder(writer).Encode(jobResult); err != nil {
			_, _ = writer.Write([]byte(fmt.Sprintf("failed writing json for %s", jobResult.Format())))
		}

		writer.Header().Set("Content-Type", "application/json")
	} else {
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
		}()

		return nil
	}

	return nil
}
