package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/charts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/dashboard"
	"github.com/spf13/cobra"
)

var devDashboardNoTLS bool

var devDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the dev services dashboard",
	Long: `Start a local HTTP server that serves a dashboard for all dev services.

The dashboard uses virtual-host routing via *.localtest.me subdomains
so every service gets a meaningful URL on a single port:

  marquez.localtest.me:<port>      → Marquez Web UI
  marquez-api.localtest.me:<port>  → Marquez API
  redpanda.localtest.me:<port>     → Schema Registry
  s3.localtest.me:<port>           → LocalStack S3

The landing page at localtest.me:<port> shows cards for all services
with health indicators and clickable links.

Press Ctrl+C to stop the dashboard server.`,
	RunE: runDevDashboard,
}

func init() {
	devCmd.AddCommand(devDashboardCmd)
	devDashboardCmd.Flags().BoolVar(&devDashboardNoTLS, "no-tls", false, "Disable TLS and serve over plain HTTP")
}

func runDevDashboard(cmd *cobra.Command, args []string) error {
	// Build service list from chart definitions
	services := buildServiceProxies()

	if len(services) == 0 {
		return fmt.Errorf("no services configured")
	}

	// Set up TLS if not disabled
	var opts []dashboard.Option
	if !devDashboardNoTLS {
		certFile, keyFile, tlsErr := dashboard.EnsureCerts()
		if tlsErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: TLS setup failed: %v\n", tlsErr)
		}
		if certFile != "" && keyFile != "" {
			opts = append(opts, dashboard.WithTLS(certFile, keyFile))
		} else if tlsErr == nil {
			fmt.Println("mkcert not found — serving over HTTP (install mkcert for HTTPS)")
		}
	}

	// Create dashboard server (binds to random port)
	srv, err := dashboard.New(services, opts...)
	if err != nil {
		return fmt.Errorf("failed to start dashboard: %w", err)
	}

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	scheme := "http"
	if srv.TLS() {
		scheme = "https"
	}

	fmt.Printf("Dashboard running at: %s\n", srv.URL())
	fmt.Println()
	fmt.Println("Services:")

	for _, svc := range services {
		if svc.Subdomain != "" {
			fmt.Printf("  %-18s %s://%s.localtest.me:%d%s\n", svc.Label+":", scheme, svc.Subdomain, srv.Port(), svc.DefaultPath)
		} else {
			fmt.Printf("  %-18s %s (TCP)\n", svc.Label+":", svc.TargetURL)
		}
	}

	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")

	// Open browser
	if err := openBrowser(srv.URL()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
		fmt.Fprintf(os.Stderr, "Open %s manually\n", srv.URL())
	}

	// Wait for Ctrl+C or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\nShutting down dashboard...")
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("dashboard server error: %w", err)
		}
	}

	ctx := context.Background()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	return nil
}

// buildServiceProxies creates dashboard ServiceProxy entries from DefaultCharts.
func buildServiceProxies() []dashboard.ServiceProxy {
	var services []dashboard.ServiceProxy

	for _, chart := range charts.DefaultCharts {
		for _, ep := range chart.DisplayEndpoints {
			services = append(services, dashboard.ServiceProxy{
				Subdomain:   ep.Subdomain,
				Label:       ep.Label,
				TargetURL:   ep.URL,
				Description: ep.Description,
				DefaultPath: ep.DefaultPath,
			})
		}
	}

	return services
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
