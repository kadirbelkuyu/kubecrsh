package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kadirbelkuyu/kubecrsh/internal/config"
	"github.com/kadirbelkuyu/kubecrsh/internal/daemon"
	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"github.com/kadirbelkuyu/kubecrsh/internal/notifier"
	"github.com/kadirbelkuyu/kubecrsh/internal/redaction"
	"github.com/kadirbelkuyu/kubecrsh/internal/reporter"
	"github.com/kadirbelkuyu/kubecrsh/pkg/kubernetes"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run kubecrsh in daemon mode",
	Long: `Run kubecrsh as a background daemon for production use.
Continuously watches for pod crashes and sends notifications.`,
	RunE: runDaemon,
}

var (
	slackWebhook   string
	telegramToken  string
	telegramChatId string
	webhookURL     string
	webhookToken   string
	httpAddr       string
)

func init() {
	daemonCmd.Flags().StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook URL")
	daemonCmd.Flags().StringVar(&telegramToken, "telegram-token", "", "Telegram token URL")
	daemonCmd.Flags().StringVar(&telegramChatId, "telegram-chat-id", "", "Telegram chat ID")
	daemonCmd.Flags().StringVar(&webhookURL, "webhook-url", "", "Generic webhook URL")
	daemonCmd.Flags().StringVar(&webhookToken, "webhook-token", "", "Webhook authorization token")
	daemonCmd.Flags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address for metrics and health")

	rootCmd.AddCommand(daemonCmd)
}

func runDaemon(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if kubeconfig != "" {
		cfg.Kubeconfig = kubeconfig
	}
	if k8sContext != "" {
		cfg.Context = k8sContext
	}

	client, err := kubernetes.NewClient(kubernetes.ClientConfig{
		Kubeconfig: cfg.Kubeconfig,
		Context:    cfg.Context,
	})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	store, err := reporter.NewStore(cfg.Reports.Path, reporter.WithCompression(cfg.Reports.Compression))
	if err != nil {
		return fmt.Errorf("failed to create report store: %w", err)
	}

	var notifiers []notifier.Notifier

	if slackWebhook != "" {
		notifiers = append(notifiers, notifier.NewSlackNotifier(slackWebhook, ""))
	}

	if telegramToken != "" && telegramChatId != "" {
		notifiers = append(notifiers, notifier.NewTelegramNotifier(nil, telegramToken, telegramChatId))
	}

	if webhookURL != "" {
		headers := make(map[string]string)
		if webhookToken != "" {
			headers["Authorization"] = "Bearer " + webhookToken
		}
		notifiers = append(notifiers, notifier.NewWebhookNotifier(webhookURL, headers))
	}

	redactor, err := redaction.New(cfg.Reports.Redaction)
	if err != nil {
		return fmt.Errorf("failed to init redaction: %w", err)
	}

	var redactorCfg interface {
		Apply(report *domain.ForensicReport)
	}
	if redactor != nil {
		redactorCfg = redactor
	}

	daemonCfg := daemon.Config{
		Namespace:         cfg.Namespace,
		Reasons:           cfg.Watch.Reasons,
		HTTPAddr:          httpAddr,
		Notifiers:         notifiers,
		Storage:           store,
		APIReportsEnabled: cfg.API.ReportsEnabled,
		APIToken:          cfg.API.Token,
		APIAllowFull:      cfg.API.AllowFull,
		ReportRetention:   cfg.Reports.Retention,
		Redactor:          redactorCfg,
	}

	srv := daemon.New(client, daemonCfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Printf("Starting kubecrsh daemon on %s\n", httpAddr)
	fmt.Printf("Watching namespace: %s\n", cfg.Namespace)
	fmt.Printf("Notifiers: %d configured\n", len(notifiers))

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("daemon error: %w", err)
	}

	return nil
}
