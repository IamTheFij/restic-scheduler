package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type ResticMetrics struct {
	JobStartTime         *prometheus.GaugeVec
	JobFailureCount      *prometheus.GaugeVec
	SnapshotCurrentCount *prometheus.GaugeVec
	SnapshotLatestTime   *prometheus.GaugeVec
	Registry             *prometheus.Registry
}

func (m ResticMetrics) PushToGateway(url string) error {
	err := push.New(url, "batch").
		Gatherer(m.Registry).
		Add()
	if err != nil {
		return fmt.Errorf("error pushing to registry %s: %w", url, err)
	}

	return nil
}

func InitMetrics() *ResticMetrics {
	labelNames := []string{"job"}

	metrics := &ResticMetrics{
		Registry: prometheus.NewRegistry(),
		JobStartTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "restic_job_start_time",
				Help:        "time that a job was run",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
		JobFailureCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "restic_job_failure_count",
				Help:        "number of consecutive failures for jobs",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
		SnapshotCurrentCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "restic_snapshot_current_total",
				Help:        "number of current snapshots",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
		SnapshotLatestTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "restic_snapshot_latest_time",
				Help:        "time of the most recent snapshot",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
	}

	metrics.Registry.MustRegister(metrics.JobStartTime)
	metrics.Registry.MustRegister(metrics.JobFailureCount)
	metrics.Registry.MustRegister(metrics.SnapshotCurrentCount)
	metrics.Registry.MustRegister(metrics.SnapshotLatestTime)

	return metrics
}

var Metrics = InitMetrics()
