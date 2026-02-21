package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/collector"
	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"github.com/kadirbelkuyu/kubecrsh/internal/notifier"
	"github.com/kadirbelkuyu/kubecrsh/internal/reporter"
	"github.com/kadirbelkuyu/kubecrsh/internal/watcher"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
)

type Server struct {
	client    kubernetes.Interface
	watcher   *watcher.Watcher
	collector *collector.Collector
	store     reporter.Storage
	pruner    interface {
		Prune(retention time.Duration) (reporter.PruneResult, error)
	}
	notifiers         []notifier.Notifier
	metrics           *Metrics
	httpAddr          string
	apiReportsEnabled bool
	apiToken          string
	apiAllowFull      bool
	reportRetention   time.Duration
	pruneInterval     time.Duration
	collectTimeout    time.Duration
	crashWorkers      int
	crashQueueSize    int
	notifierWorkers   int
	notifierQueueSize int
	crashQueue        chan crashJob
	notifyQueue       chan notifyJob
	redactor          interface {
		Apply(report *domain.ForensicReport)
	}
}

type Config struct {
	Namespace         string
	Reasons           []string
	HTTPAddr          string
	CrashWorkers      int
	CrashQueueSize    int
	NotifierWorkers   int
	NotifierQueueSize int
	Notifiers         []notifier.Notifier
	Storage           reporter.Storage
	APIReportsEnabled bool
	APIToken          string
	APIAllowFull      bool
	ReportRetention   time.Duration
	PruneInterval     time.Duration
	CollectTimeout    time.Duration
	Redactor          interface {
		Apply(report *domain.ForensicReport)
	}
}

type crashJob struct {
	crash      domain.PodCrash
	enqueuedAt time.Time
}

type notifyJob struct {
	notifier   notifier.Notifier
	report     domain.ForensicReport
	enqueuedAt time.Time
}

const (
	defaultCrashWorkers      = 4
	defaultCrashQueueSize    = 512
	defaultNotifierWorkers   = 4
	defaultNotifierQueueSize = 1024
)

func New(client kubernetes.Interface, cfg Config) *Server {
	metrics := NewMetrics()
	prometheus.MustRegister(
		metrics.CrashesTotal,
		metrics.ReportSize,
		metrics.NotificationsSent,
		metrics.CollectDuration,
		metrics.SaveDuration,
		metrics.ProcessDuration,
		metrics.NotifyDuration,
		metrics.CrashQueueDepth,
		metrics.NotifyQueueDepth,
		metrics.QueueDropped,
		metrics.CrashQueueWait,
		metrics.NotifyQueueWait,
	)

	crashWorkers := cfg.CrashWorkers
	if crashWorkers <= 0 {
		crashWorkers = defaultCrashWorkers
	}

	crashQueueSize := cfg.CrashQueueSize
	if crashQueueSize <= 0 {
		crashQueueSize = defaultCrashQueueSize
	}

	notifierWorkers := cfg.NotifierWorkers
	if notifierWorkers <= 0 {
		notifierWorkers = defaultNotifierWorkers
	}

	notifierQueueSize := cfg.NotifierQueueSize
	if notifierQueueSize <= 0 {
		notifierQueueSize = defaultNotifierQueueSize
	}

	srv := &Server{
		client:            client,
		collector:         collector.New(client),
		store:             cfg.Storage,
		notifiers:         cfg.Notifiers,
		metrics:           metrics,
		httpAddr:          cfg.HTTPAddr,
		apiReportsEnabled: cfg.APIReportsEnabled,
		apiToken:          cfg.APIToken,
		apiAllowFull:      cfg.APIAllowFull,
		reportRetention:   cfg.ReportRetention,
		pruneInterval:     cfg.PruneInterval,
		collectTimeout:    cfg.CollectTimeout,
		crashWorkers:      crashWorkers,
		crashQueueSize:    crashQueueSize,
		notifierWorkers:   notifierWorkers,
		notifierQueueSize: notifierQueueSize,
		crashQueue:        make(chan crashJob, crashQueueSize),
		notifyQueue:       make(chan notifyJob, notifierQueueSize),
		redactor:          cfg.Redactor,
	}

	opts := []watcher.Option{watcher.WithReasons(cfg.Reasons)}
	if cfg.Namespace != "" {
		opts = append(opts, watcher.WithNamespace(cfg.Namespace))
	}

	srv.watcher = watcher.New(client, srv.handleCrash, opts...)

	if p, ok := cfg.Storage.(interface {
		Prune(retention time.Duration) (reporter.PruneResult, error)
	}); ok {
		srv.pruner = p
	}

	return srv
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.Handle("/metrics", promhttp.Handler())
	if s.apiReportsEnabled {
		mux.HandleFunc("/reports", s.reportsListHandler)
		mux.HandleFunc("/reports/", s.reportGetHandler)
	}

	httpServer := &http.Server{
		Addr:    s.httpAddr,
		Handler: mux,
	}

	errCh := make(chan error, 1)

	for i := 0; i < s.crashWorkers; i++ {
		go s.runCrashWorker(ctx)
	}

	if len(s.notifiers) > 0 {
		for i := 0; i < s.notifierWorkers; i++ {
			go s.runNotifierWorker(ctx)
		}
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server error: %w", err)
		}
	}()

	go func() {
		if err := s.watcher.Start(ctx); err != nil {
			errCh <- fmt.Errorf("watcher error: %w", err)
		}
	}()

	go s.pruneLoop(ctx)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

func (s *Server) handleCrash(crash domain.PodCrash) {
	job := crashJob{
		crash:      crash,
		enqueuedAt: time.Now(),
	}

	select {
	case s.crashQueue <- job:
		s.metrics.CrashQueueDepth.Inc()
	default:
		s.metrics.QueueDropped.WithLabelValues("crash").Inc()
		fmt.Printf(
			"Dropping crash event due to full queue: namespace=%s pod=%s container=%s reason=%s\n",
			crash.Namespace,
			crash.PodName,
			crash.ContainerName,
			crash.Reason,
		)
	}
}

func (s *Server) runCrashWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.crashQueue:
			s.metrics.CrashQueueDepth.Dec()
			s.metrics.CrashQueueWait.Observe(time.Since(job.enqueuedAt).Seconds())
			s.processCrash(job.crash)
		}
	}
}

func (s *Server) processCrash(crash domain.PodCrash) {
	processStart := time.Now()

	timeout := s.collectTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	collectStart := time.Now()
	report, err := s.collector.CollectForensics(ctx, crash)
	s.metrics.CollectDuration.Observe(time.Since(collectStart).Seconds())
	if err != nil {
		fmt.Printf("Failed to collect forensics: %v\n", err)
		return
	}

	if s.redactor != nil {
		s.redactor.Apply(report)
	}

	s.metrics.CrashesTotal.WithLabelValues(
		crash.Namespace,
		crash.Reason,
	).Inc()

	saveStart := time.Now()
	savedBytes, saveErr := s.saveReport(report)
	s.metrics.SaveDuration.Observe(time.Since(saveStart).Seconds())
	if saveErr != nil {
		fmt.Printf("Failed to save report: %v\n", saveErr)
	}

	if savedBytes > 0 {
		s.metrics.ReportSize.Observe(float64(savedBytes))
		s.enqueueNotifications(*report)
		s.metrics.ProcessDuration.Observe(time.Since(processStart).Seconds())
		return
	}

	data, err := json.Marshal(report)
	if err != nil {
		fmt.Printf("Failed to measure report size: %v\n", err)
		return
	}

	s.metrics.ReportSize.Observe(float64(len(data)))
	s.enqueueNotifications(*report)
	s.metrics.ProcessDuration.Observe(time.Since(processStart).Seconds())
}

func (s *Server) saveReport(report *domain.ForensicReport) (int64, error) {
	if saver, ok := s.store.(reporter.SaveWithResult); ok {
		res, err := saver.SaveWithResult(report)
		if err != nil {
			return 0, err
		}
		return res.BytesWritten, nil
	}

	if err := s.store.Save(report); err != nil {
		return 0, err
	}

	return 0, nil
}

func (s *Server) enqueueNotifications(report domain.ForensicReport) {
	for _, n := range s.notifiers {
		job := notifyJob{
			notifier:   n,
			report:     report,
			enqueuedAt: time.Now(),
		}

		select {
		case s.notifyQueue <- job:
			s.metrics.NotifyQueueDepth.Inc()
		default:
			s.metrics.QueueDropped.WithLabelValues("notify").Inc()
			fmt.Printf(
				"Dropping notification due to full queue: report=%s notifier=%s\n",
				report.ID,
				n.Name(),
			)
		}
	}
}

func (s *Server) runNotifierWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.notifyQueue:
			s.metrics.NotifyQueueDepth.Dec()
			s.metrics.NotifyQueueWait.Observe(time.Since(job.enqueuedAt).Seconds())

			start := time.Now()
			err := job.notifier.Notify(job.report)
			duration := time.Since(start).Seconds()

			if err != nil {
				s.metrics.NotifyDuration.WithLabelValues(job.notifier.Name(), "failure").Observe(duration)
				s.metrics.NotificationsSent.WithLabelValues(job.notifier.Name(), "failure").Inc()
				fmt.Printf("Failed to send notification: %v\n", err)
				continue
			}

			s.metrics.NotifyDuration.WithLabelValues(job.notifier.Name(), "success").Observe(duration)
			s.metrics.NotificationsSent.WithLabelValues(job.notifier.Name(), "success").Inc()
		}
	}
}

func (s *Server) pruneLoop(ctx context.Context) {
	if s.pruner == nil || s.reportRetention <= 0 {
		return
	}

	interval := s.pruneInterval
	if interval <= 0 {
		interval = time.Hour
	}

	if _, err := s.pruner.Prune(s.reportRetention); err != nil {
		fmt.Printf("Failed to prune reports: %v\n", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.pruner.Prune(s.reportRetention); err != nil {
				fmt.Printf("Failed to prune reports: %v\n", err)
			}
		}
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Ready"))
}
