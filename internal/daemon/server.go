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
	redactor          interface {
		Apply(report *domain.ForensicReport)
	}
}

type Config struct {
	Namespace         string
	Reasons           []string
	HTTPAddr          string
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

func New(client kubernetes.Interface, cfg Config) *Server {
	metrics := NewMetrics()
	prometheus.MustRegister(metrics.CrashesTotal, metrics.ReportSize, metrics.NotificationsSent)

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
	timeout := s.collectTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	report, err := s.collector.CollectForensics(ctx, crash)
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

	for _, n := range s.notifiers {
		if err := n.Notify(*report); err != nil {
			fmt.Printf("Failed to send notification: %v\n", err)
			report.AddWarning(fmt.Sprintf("notify %s: %v", n.Name(), err))
			s.metrics.NotificationsSent.WithLabelValues(
				n.Name(),
				"failure",
			).Inc()
		} else {
			s.metrics.NotificationsSent.WithLabelValues(
				n.Name(),
				"success",
			).Inc()
		}
	}

	var savedBytes int64
	if saver, ok := s.store.(reporter.SaveWithResult); ok {
		res, err := saver.SaveWithResult(report)
		if err != nil {
			fmt.Printf("Failed to save report: %v\n", err)
		} else {
			savedBytes = res.BytesWritten
		}
	} else {
		if err := s.store.Save(report); err != nil {
			fmt.Printf("Failed to save report: %v\n", err)
		}
	}

	if savedBytes > 0 {
		s.metrics.ReportSize.Observe(float64(savedBytes))
		return
	}

	data, err := json.Marshal(report)
	if err != nil {
		fmt.Printf("Failed to measure report size: %v\n", err)
		return
	}

	s.metrics.ReportSize.Observe(float64(len(data)))
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
