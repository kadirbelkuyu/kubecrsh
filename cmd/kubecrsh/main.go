package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kadirbelkuyu/kubecrsh/internal/config"
	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	"github.com/kadirbelkuyu/kubecrsh/internal/reporter"
	"github.com/kadirbelkuyu/kubecrsh/internal/tui"
	"github.com/kadirbelkuyu/kubecrsh/internal/watcher"
	"github.com/kadirbelkuyu/kubecrsh/pkg/kubernetes"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	kubeconfig string
	k8sContext string
	namespace  string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "kubecrsh",
	Short: "Kubernetes Crash Scene Investigator",
	Long: `A forensic CLI tool that watches Kubernetes pods for crashes and captures
logs, events, and environment variables before they disappear.`,
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for pod crashes in real-time",
	Long: `Start watching for pod crashes and display them in a TUI.
When a pod crashes, kubecrsh will automatically capture logs, events,
and environment variables for forensic analysis.`,
	RunE: runWatch,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved crash reports",
	Long:  `Display all saved crash reports in the TUI.`,
	RunE:  runList,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubecrsh/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	rootCmd.PersistentFlags().StringVar(&k8sContext, "context", "", "kubernetes context to use")

	watchCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "namespace to watch (default: all namespaces)")

	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(listCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
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
	if namespace != "" {
		cfg.Namespace = namespace
	}

	client, err := kubernetes.NewClient(kubernetes.ClientConfig{
		Kubeconfig: cfg.Kubeconfig,
		Context:    cfg.Context,
	})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	store, err := reporter.NewStore(cfg.Reports.Path)
	if err != nil {
		return fmt.Errorf("failed to create report store: %w", err)
	}

	tuiModel := tui.New(client, store)
	p := tea.NewProgram(
		tuiModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		p.Quit()
	}()

	crashHandler := func(crash domain.PodCrash) {
		p.Send(tui.OnCrash(crash))
	}

	opts := []watcher.Option{watcher.WithReasons(cfg.Watch.Reasons)}
	if cfg.Namespace != "" {
		opts = append(opts, watcher.WithNamespace(cfg.Namespace))
	}

	w := watcher.New(client, crashHandler, opts...)

	go func() {
		if err := w.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
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

	store, err := reporter.NewStore(cfg.Reports.Path)
	if err != nil {
		return fmt.Errorf("failed to create report store: %w", err)
	}

	tuiModel := tui.New(client, store)
	p := tea.NewProgram(
		tuiModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
