package daemon

import (
	"context"
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
	notifiers []notifier.Notifier
	metrics   *Metrics
	httpAddr  string
}

type Config struct {
	Namespace string
	Reasons   []string
	HTTPAddr  string
	Notifiers []notifier.Notifier
	Storage   reporter.Storage
}

func New(client kubernetes.Interface, cfg Config) *Server {
	metrics := NewMetrics()
	prometheus.MustRegister(metrics.CrashesTotal, metrics.ReportSize, metrics.NotificationsSent)

	srv := &Server{
		client:    client,
		collector: collector.New(client),
		store:     cfg.Storage,
		notifiers: cfg.Notifiers,
		metrics:   metrics,
		httpAddr:  cfg.HTTPAddr,
	}

	opts := []watcher.Option{watcher.WithReasons(cfg.Reasons)}
	if cfg.Namespace != "" {
		opts = append(opts, watcher.WithNamespace(cfg.Namespace))
	}

	srv.watcher = watcher.New(client, srv.handleCrash, opts...)
	return srv
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readyHandler)
	mux.Handle("/metrics", promhttp.Handler())

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
	ctx := context.Background()

	report, err := s.collector.CollectForensics(ctx, crash)
	if err != nil {
		fmt.Printf("Failed to collect forensics: %v\n", err)
		return
	}

	if err := s.store.Save(report); err != nil {
		fmt.Printf("Failed to save report: %v\n", err)
	}

	s.metrics.CrashesTotal.WithLabelValues(
		crash.Namespace,
		crash.Reason,
	).Inc()

	for _, n := range s.notifiers {
		if err := n.Notify(*report); err != nil {
			fmt.Printf("Failed to send notification: %v\n", err)
		} else {
			s.metrics.NotificationsSent.WithLabelValues(
				n.Name(),
				"success",
			).Inc()
		}
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}
