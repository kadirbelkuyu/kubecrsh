package daemon

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	CrashesTotal      *prometheus.CounterVec
	ReportSize        prometheus.Histogram
	NotificationsSent *prometheus.CounterVec
	CollectDuration   prometheus.Histogram
	SaveDuration      prometheus.Histogram
	ProcessDuration   prometheus.Histogram
	NotifyDuration    *prometheus.HistogramVec
	CrashQueueDepth   prometheus.Gauge
	NotifyQueueDepth  prometheus.Gauge
	QueueDropped      *prometheus.CounterVec
	CrashQueueWait    prometheus.Histogram
	NotifyQueueWait   prometheus.Histogram
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
		CollectDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kubecrsh_collect_duration_seconds",
				Help:    "Duration of forensic collection operation in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		SaveDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kubecrsh_save_duration_seconds",
				Help:    "Duration of report save operation in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		ProcessDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kubecrsh_crash_process_duration_seconds",
				Help:    "End-to-end crash processing duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		NotifyDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "kubecrsh_notify_duration_seconds",
				Help:    "Duration of notifier call in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"notifier", "status"},
		),
		CrashQueueDepth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kubecrsh_crash_queue_depth",
				Help: "Current number of pending crash jobs",
			},
		),
		NotifyQueueDepth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kubecrsh_notify_queue_depth",
				Help: "Current number of pending notification jobs",
			},
		),
		QueueDropped: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kubecrsh_queue_dropped_total",
				Help: "Total number of dropped jobs due to full queues",
			},
			[]string{"queue"},
		),
		CrashQueueWait: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kubecrsh_crash_queue_wait_seconds",
				Help:    "Time spent waiting in crash queue in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		NotifyQueueWait: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "kubecrsh_notify_queue_wait_seconds",
				Help:    "Time spent waiting in notification queue in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
	}
}
