package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ResticMetrics struct {
	JobStartTime         *prometheus.GaugeVec
	JobFailureCount      *prometheus.GaugeVec
	SnapshotCurrentCount *prometheus.GaugeVec
	SnapshotLatestTime   *prometheus.GaugeVec
}

func InitMetrics() *ResticMetrics {
	labelNames := []string{"job"}

	metrics := &ResticMetrics{
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
				Help:        "time it took for the last job run",
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

	prometheus.MustRegister(metrics.JobStartTime)
	prometheus.MustRegister(metrics.SnapshotCurrentCount)
	prometheus.MustRegister(metrics.SnapshotLatestTime)

	return metrics
}

var Metrics = InitMetrics()
