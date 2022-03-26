package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"
)

func ScheduleAndRunJobs(jobs []Job) error {
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	runner := cron.New()

	for _, job := range jobs {
		fmt.Println("Scheduling", job.Name)

		if _, err := runner.AddJob(job.Schedule, job); err != nil {
			return fmt.Errorf("Error scheduling job %s: %w", job.Name, err)
		}
	}

	runner.Start()

	switch <-signalChan {
	case syscall.SIGINT:
		fmt.Println("Stopping now...")

		defer runner.Stop()

		return nil
	case syscall.SIGTERM:
		fallthrough
	case syscall.SIGQUIT:
		// Wait for all jobs to complete
		fmt.Println("Stopping after running jobs complete...")

		defer func() {
			ctx := runner.Stop()
			<-ctx.Done()
		}()

		return nil
	}

	return nil
}
