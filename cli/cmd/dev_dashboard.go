package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/charts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/dashboard"
	"github.com/spf13/cobra"
)

var (
	devDashboardNoTLS bool
	devDashboardDomain string
)

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

The landing page at console.<domain>:<port> shows cards for all services
with health indicators and clickable links.

The domain defaults to the k3d cluster name + ".test" (auto-detected from
kubectl context), or "localtest.me" if not in a k3d cluster.

Press Ctrl+C to stop the dashboard server.`,
	RunE: runDevDashboard,
}

func init() {
	devCmd.AddCommand(devDashboardCmd)
	devDashboardCmd.Flags().BoolVar(&devDashboardNoTLS, "no-tls", false, "Disable TLS and serve over plain HTTP")
	devDashboardCmd.Flags().StringVar(&devDashboardDomain, "domain", "", "Base domain for service URLs (default: auto-detect from k3d context)")
}

func runDevDashboard(cmd *cobra.Command, args []string) error {
	// Resolve domain: flag > auto-detect from k3d context > default
	domain := devDashboardDomain
	if domain == "" {
		domain = detectClusterDomain()
	}

	// If a k3d cluster is running with dk-dashboard deployed, just open browser
	if domain != "" && clusterHasDashboard(domain) {
		dashURL := fmt.Sprintf("https://console.%s", domain)
		fmt.Printf("Dashboard available at: %s\n", dashURL)
		if err := openBrowser(dashURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
			fmt.Fprintf(os.Stderr, "Open %s manually\n", dashURL)
		}
		return nil
	}

	// Fall back to in-process dashboard mode
	return runInProcessDashboard(domain)
}

// clusterHasDashboard checks if the dk-dashboard Ingress exists in the cluster.
func clusterHasDashboard(domain string) bool {
	ctx := strings.TrimSuffix(domain, ".test")
	kubeCtx := "k3d-" + ctx

	out, err := exec.Command("kubectl", "--context", kubeCtx,
		"get", "ingress", "dk-dashboard",
		"-n", "dk-local", "-o", "name",
	).Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// runInProcessDashboard falls back to the original in-process reverse-proxy dashboard.
func runInProcessDashboard(domain string) error {
	services := buildServiceProxies()
	if len(services) == 0 {
		return fmt.Errorf("no services configured")
	}

	var opts []dashboard.Option
	if domain != "" {
		opts = append(opts, dashboard.WithDomain(domain))
	}

	if !devDashboardNoTLS {
		tlsDomain := domain
		if tlsDomain == "" {
			tlsDomain = dashboard.DefaultDomain
		}
		certFile, keyFile, tlsErr := dashboard.EnsureCertsForDomain(tlsDomain)
		if tlsErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: TLS setup failed: %v\n", tlsErr)
		}
		if certFile != "" && keyFile != "" {
			opts = append(opts, dashboard.WithTLS(certFile, keyFile))
		} else if tlsErr == nil {
			fmt.Println("mkcert not found — serving over HTTP (install mkcert for HTTPS)")
		}
	}

	srv, err := dashboard.New(services, opts...)
	if err != nil {
		return fmt.Errorf("failed to start dashboard: %w", err)
	}

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
			fmt.Printf("  %-18s %s://%s.%s:%d%s\n", svc.Label+":", scheme, svc.Subdomain, srv.Domain(), srv.Port(), svc.DefaultPath)
		} else {
			fmt.Printf("  %-18s %s (TCP)\n", svc.Label+":", svc.TargetURL)
		}
	}

	upstream := fmt.Sprintf("http://127.0.0.1:%d", srv.Port())
	registered := registerWithDevedge(srv.Domain(), services, upstream)
	if registered > 0 {
		fmt.Printf("\nRegistered %d hostname(s) with devedge\n", registered)
	}

	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")

	if err := openBrowser(srv.URL()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
		fmt.Fprintf(os.Stderr, "Open %s manually\n", srv.URL())
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\nShutting down dashboard...")
		deregisterFromDevedge("dk-dashboard")
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

const devedgeAPI = "http://127.0.0.1:15353"

// registerWithDevedge registers the console and service hostnames with the
// devedge daemon for DNS resolution. Returns the number of routes registered.
// Best-effort — silently returns 0 if devedge isn't running.
func registerWithDevedge(domain string, services []dashboard.ServiceProxy, upstream string) int {
	count := 0

	// Register console.<domain>
	hosts := []string{"console." + domain}
	for _, svc := range services {
		if svc.Subdomain != "" {
			hosts = append(hosts, svc.Subdomain+"."+domain)
		}
	}

	for _, host := range hosts {
		body, _ := json.Marshal(map[string]string{
			"host":     host,
			"upstream": upstream,
			"project":  "dk-dashboard",
			"owner":    "dk",
		})
		req, _ := http.NewRequest("PUT", devedgeAPI+"/v1/routes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return count // devedge not running
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			count++
		}
	}
	return count
}

// deregisterFromDevedge removes all routes for the dk-dashboard project.
func deregisterFromDevedge(project string) {
	req, _ := http.NewRequest("DELETE", devedgeAPI+"/v1/projects/"+project, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// detectClusterDomain checks the current kubectl context for a k3d cluster
// and returns "<clustername>.test" if found. Returns "" otherwise.
func detectClusterDomain() string {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	if err != nil {
		return ""
	}
	ctx := strings.TrimSpace(string(out))
	// k3d contexts are named "k3d-<clustername>"
	if strings.HasPrefix(ctx, "k3d-") {
		clusterName := strings.TrimPrefix(ctx, "k3d-")
		return clusterName + ".test"
	}
	return ""
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
