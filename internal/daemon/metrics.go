package daemon

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	CrashesTotal      *prometheus.CounterVec
	ReportSize        prometheus.Histogram
	NotificationsSent *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		CrashesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kubecrsh_crashes_total",
				Help: "Total number of pod crashes detected",
			},
			[]string{"namespace", "reason"},
		),
		ReportSize: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kubecrsh_report_size_bytes",
				Help:    "Size of forensic reports in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 10),
			},
		),
		NotificationsSent: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kubecrsh_notifications_sent_total",
				Help: "Total number of notifications sent",
			},
			[]string{"notifier", "status"},
		),
	}
}
