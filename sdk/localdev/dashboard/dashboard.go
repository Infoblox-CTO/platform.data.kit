// Package dashboard provides an HTTP server that serves a dev services
// dashboard and reverse-proxies to local dev services via virtual hosts.
//
// The server listens on a single random port and routes requests based on
// the Host header. Requests to <subdomain>.localtest.me are reverse-proxied
// to the corresponding backend service. Requests to the bare localtest.me
// host serve the dashboard landing page.
//
// localtest.me is a public DNS wildcard — *.localtest.me always resolves
// to 127.0.0.1, so no /etc/hosts configuration is needed.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ServiceProxy defines a service that can be reverse-proxied via a subdomain.
type ServiceProxy struct {
	// Subdomain is the virtual host subdomain (e.g., "marquez", "s3").
	// Empty means the service is not HTTP-proxyable (e.g., TCP services).
	Subdomain string

	// Label is the human-readable display name (e.g., "Marquez Web").
	Label string

	// TargetURL is the backend URL to proxy to (e.g., "http://localhost:3000").
	TargetURL string

	// Description is a short description (e.g., "Data lineage tracking UI").
	Description string

	// DefaultPath is appended to the proxy URL for clickable links
	// (e.g., "/subjects" for Schema Registry). Empty means "/".
	DefaultPath string
}

// DefaultDomain is the fallback domain when no cluster domain is configured.
// localtest.me is a public DNS wildcard that resolves to 127.0.0.1.
const DefaultDomain = "localtest.me"

// Option configures a Server.
type Option func(*Server)

// WithTLS configures the server to serve over HTTPS using the given cert and key files.
func WithTLS(certFile, keyFile string) Option {
	return func(s *Server) {
		s.certFile = certFile
		s.keyFile = keyFile
		s.tls = true
	}
}

// WithDomain sets the base domain for virtual-host routing.
// Services are available at <subdomain>.<domain>:<port>.
// Defaults to "localtest.me" if not set.
func WithDomain(domain string) Option {
	return func(s *Server) {
		s.domain = domain
	}
}

// Server is the dashboard HTTP server with virtual-host reverse proxying.
type Server struct {
	listener net.Listener
	server   *http.Server
	services []ServiceProxy
	proxies  map[string]*httputil.ReverseProxy
	domain   string // base domain for vhost routing (default: localtest.me)
	tls      bool
	certFile string
	keyFile  string
	mu       sync.RWMutex
}

// Domain returns the configured base domain, defaulting to localtest.me.
func (s *Server) Domain() string {
	if s.domain != "" {
		return s.domain
	}
	return DefaultDomain
}

// New creates a new dashboard server bound to a random port.
// Pass WithTLS(...) to serve over HTTPS.
func New(services []ServiceProxy, opts ...Option) (*Server, error) {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind: %w", err)
	}

	s := &Server{
		listener: listener,
		services: services,
		proxies:  make(map[string]*httputil.ReverseProxy),
	}

	// Build reverse proxies for each proxyable service
	for _, svc := range services {
		if svc.Subdomain == "" {
			continue
		}
		target, err := url.Parse(svc.TargetURL)
		if err != nil {
			listener.Close()
			return nil, fmt.Errorf("invalid target URL %q for %s: %w", svc.TargetURL, svc.Label, err)
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		// Rewrite the Host header so the backend sees its own address
		// instead of the *.localtest.me vhost from the browser.
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = target.Host
		}
		s.proxies[svc.Subdomain] = proxy
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)
	mux.HandleFunc("/_api/status", s.handleStatus)

	s.server = &http.Server{
		Handler: mux,
	}

	// Apply options
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s, nil
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// URL returns the dashboard URL (e.g., "https://console.mydev.test:54321").
func (s *Server) URL() string {
	return fmt.Sprintf("%s://console.%s:%d", s.scheme(), s.Domain(), s.Port())
}

// TLS returns true if the server is configured to serve over HTTPS.
func (s *Server) TLS() bool {
	return s.tls
}

// scheme returns "https" if TLS is configured, "http" otherwise.
func (s *Server) scheme() string {
	if s.tls {
		return "https"
	}
	return "http"
}

// Start begins serving requests. This method blocks until the server is
// shut down or encounters a fatal error. If TLS is configured via WithTLS,
// the server serves HTTPS; otherwise plain HTTP.
func (s *Server) Start() error {
	var err error
	if s.tls {
		err = s.server.ServeTLS(s.listener, s.certFile, s.keyFile)
	} else {
		err = s.server.Serve(s.listener)
	}
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// handleRequest routes requests based on the Host header.
// Subdomain requests are reverse-proxied; bare host serves the dashboard.
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	subdomain := extractSubdomain(r.Host, s.Domain())

	if subdomain != "" {
		s.mu.RLock()
		proxy, ok := s.proxies[subdomain]
		s.mu.RUnlock()

		if ok {
			proxy.ServeHTTP(w, r)
			return
		}
	}

	// Serve dashboard landing page
	s.serveDashboard(w, r)
}

// handleStatus returns JSON health status for all services.
// Used by the dashboard JS to auto-refresh status indicators.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	type serviceStatus struct {
		Label       string `json:"label"`
		Subdomain   string `json:"subdomain,omitempty"`
		TargetURL   string `json:"targetUrl"`
		Description string `json:"description"`
		Healthy     bool   `json:"healthy"`
		ProxyURL    string `json:"proxyUrl,omitempty"`
	}

	port := s.Port()
	statuses := make([]serviceStatus, 0, len(s.services))

	for _, svc := range s.services {
		st := serviceStatus{
			Label:       svc.Label,
			Subdomain:   svc.Subdomain,
			TargetURL:   svc.TargetURL,
			Description: svc.Description,
		}

		if svc.Subdomain != "" {
			st.ProxyURL = fmt.Sprintf("%s://%s.localtest.me:%d%s", s.scheme(), svc.Subdomain, port, svc.DefaultPath)
		}

		// Quick health check
		st.Healthy = checkHealth(svc.TargetURL)

		statuses = append(statuses, st)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// checkHealth performs a quick HTTP GET to determine if a service is reachable.
func checkHealth(targetURL string) bool {
	if targetURL == "" {
		return false
	}

	// Only check HTTP services
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		return false
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

// extractSubdomain extracts the subdomain from a Host header value.
// For "marquez.localtest.me:54321" it returns "marquez".
// For "localtest.me:54321" or "localhost:54321" it returns "".
func extractSubdomain(host, domain string) string {
	// Strip port
	hostname := host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		hostname = host[:idx]
	}

	hostname = strings.ToLower(hostname)
	suffix := "." + strings.ToLower(domain)

	if strings.HasSuffix(hostname, suffix) {
		sub := strings.TrimSuffix(hostname, suffix)
		if sub != "" && !strings.Contains(sub, ".") {
			return sub
		}
	}

	return ""
}

// serveDashboard renders the HTML dashboard landing page.
func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request) {
	port := s.Port()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, renderDashboardHTML(s.services, port, s.scheme()))
}

// renderDashboardHTML generates the dashboard HTML with service cards.
func renderDashboardHTML(services []ServiceProxy, port int, scheme string) string {
	var cards strings.Builder
	for _, svc := range services {
		statusClass := "status-unknown"
		var linkHTML string

		if svc.Subdomain != "" {
			proxyURL := fmt.Sprintf("%s://%s.localtest.me:%d%s", scheme, svc.Subdomain, port, svc.DefaultPath)
			linkHTML = fmt.Sprintf(`<a href="%s" target="_blank" class="card-link">Open %s</a>`, proxyURL, svc.Label)
		} else {
			linkHTML = fmt.Sprintf(`<span class="card-connection">%s</span>`, svc.TargetURL)
		}

		cards.WriteString(fmt.Sprintf(`
        <div class="card" data-target="%s">
          <div class="card-header">
            <span class="status-dot %s"></span>
            <h3>%s</h3>
          </div>
          <p class="card-desc">%s</p>
          <div class="card-footer">
            %s
          </div>
        </div>`, svc.TargetURL, statusClass, svc.Label, svc.Description, linkHTML))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DataKit Dev Dashboard</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: #0f172a;
      color: #e2e8f0;
      min-height: 100vh;
    }
    .header {
      background: #1e293b;
      border-bottom: 1px solid #334155;
      padding: 1.5rem 2rem;
      display: flex;
      align-items: center;
      gap: 1rem;
    }
    .header h1 {
      font-size: 1.5rem;
      font-weight: 600;
      color: #f8fafc;
    }
    .header .badge {
      background: #059669;
      color: white;
      font-size: 0.75rem;
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
      font-weight: 500;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
      gap: 1.25rem;
      padding: 2rem;
      max-width: 1200px;
      margin: 0 auto;
    }
    .card {
      background: #1e293b;
      border: 1px solid #334155;
      border-radius: 8px;
      padding: 1.25rem;
      transition: border-color 0.2s;
    }
    .card:hover { border-color: #475569; }
    .card-header {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 0.5rem;
    }
    .card-header h3 { font-size: 1rem; font-weight: 600; color: #f1f5f9; }
    .status-dot {
      width: 10px;
      height: 10px;
      border-radius: 50%%;
      flex-shrink: 0;
    }
    .status-healthy { background: #22c55e; box-shadow: 0 0 6px #22c55e88; }
    .status-unhealthy { background: #ef4444; box-shadow: 0 0 6px #ef444488; }
    .status-unknown { background: #64748b; }
    .card-desc {
      font-size: 0.875rem;
      color: #94a3b8;
      margin-bottom: 1rem;
      line-height: 1.4;
    }
    .card-footer { display: flex; align-items: center; }
    .card-link {
      color: #38bdf8;
      text-decoration: none;
      font-size: 0.875rem;
      font-weight: 500;
    }
    .card-link:hover { text-decoration: underline; }
    .card-connection {
      font-family: 'SF Mono', Monaco, monospace;
      font-size: 0.8rem;
      color: #94a3b8;
      background: #0f172a;
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
    }
    .refresh-info {
      text-align: center;
      padding: 1rem;
      color: #64748b;
      font-size: 0.75rem;
    }
  </style>
</head>
<body>
  <div class="header">
    <h1>DataKit Dev Dashboard</h1>
    <span class="badge">local · %s</span>
  </div>
  <div class="grid">%s
  </div>
  <div class="refresh-info">Status refreshes every 10 seconds</div>
  <script>
    async function refreshStatus() {
      try {
        const resp = await fetch('/_api/status');
        const statuses = await resp.json();
        document.querySelectorAll('.card').forEach(card => {
          const target = card.dataset.target;
          const svc = statuses.find(s => s.targetUrl === target);
          if (!svc) return;
          const dot = card.querySelector('.status-dot');
          dot.className = 'status-dot ' + (svc.healthy ? 'status-healthy' : 'status-unhealthy');
        });
      } catch (e) {
        // Silently ignore refresh errors
      }
    }
    // Initial status check
    refreshStatus();
    // Refresh every 10 seconds
    setInterval(refreshStatus, 10000);
  </script>
</body>
</html>`, scheme, cards.String())
}
