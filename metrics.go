package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ResticMetrics struct {
	JobRunTime         *prometheus.GaugeVec
	JobRunDuration     *prometheus.GaugeVec
	SnapshotCount      *prometheus.GaugeVec
	LatestSnapshotTime *prometheus.GaugeVec
	LatestSnapshotSize *prometheus.GaugeVec
}

func InitMetrics() *ResticMetrics {
	labelNames := []string{"job"}

	metrics := &ResticMetrics{
		JobRunTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "job_run_time",
				Help:        "time that a job was run",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
		JobRunDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "job_run_duration",
				Help:        "time it took for the last job run",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
		SnapshotCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "snapshot_total",
				Help:        "time it took for the last job run",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
		LatestSnapshotTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "latest_snapshot_time",
				Help:        "time of the most recent snapshot",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
		LatestSnapshotSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "latest_snapshot_size",
				Help:        "size of the most recent snapshot",
				Namespace:   "",
				Subsystem:   "",
				ConstLabels: nil,
			},
			labelNames,
		),
	}

	prometheus.MustRegister(metrics.JobRunTime)
	prometheus.MustRegister(metrics.JobRunDuration)
	prometheus.MustRegister(metrics.SnapshotCount)
	prometheus.MustRegister(metrics.LatestSnapshotTime)
	prometheus.MustRegister(metrics.LatestSnapshotSize)

	return metrics
}

var Metrics = InitMetrics()

func JobComplete() {

}
