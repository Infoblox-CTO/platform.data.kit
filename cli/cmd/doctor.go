package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/spf13/cobra"
)

// MinGoMajor is the minimum required Go major version.
const MinGoMajor = 1

// MinGoMinor is the minimum required Go minor version.
const MinGoMinor = 25

// doctorCmd validates the local development setup.
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate your local development setup",
	Long: `Run a series of checks to validate that your local development
environment is correctly configured for working with DK.

Checks performed:
  • Go version (minimum ` + fmt.Sprintf("%d.%d", MinGoMajor, MinGoMinor) + `)
  • Container runtime (Docker or Rancher Desktop) installed and running
  • k3d installed
  • kubectl installed
  • helm installed
  • OCI registry connectivity
  • Dev stack health (k3d cluster, services)

Examples:
  # Run all checks
  dk doctor

  # Run checks with verbose output
  dk doctor -v`,
	RunE: runDoctor,
}

var doctorVerbose bool

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false,
		"Show detailed output for each check")
}

// CheckStatus represents the result of a single doctor check.
type CheckStatus int

const (
	// CheckOK means the check passed.
	CheckOK CheckStatus = iota
	// CheckWarn means the check passed with a warning.
	CheckWarn
	// CheckFail means the check failed.
	CheckFail
)

// CheckResult stores the outcome of a single doctor check.
type CheckResult struct {
	Name    string
	Status  CheckStatus
	Message string
	Detail  string // shown only with --verbose
	Fix     string // actionable fix suggestion
}

// statusIcon returns the icon for a check status.
func statusIcon(s CheckStatus) string {
	switch s {
	case CheckOK:
		return "✓"
	case CheckWarn:
		return "⚠"
	default:
		return "✗"
	}
}

// runDoctor executes all doctor checks and prints the results.
func runDoctor(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Fprintln(cmd.OutOrStdout(), "DK Doctor")
	fmt.Fprintln(cmd.OutOrStdout(), "=========")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Running environment checks...\n\n")

	checks := gatherChecks(ctx)
	printResults(cmd, checks)

	// Determine overall status
	var failCount, warnCount int
	for _, c := range checks {
		switch c.Status {
		case CheckFail:
			failCount++
		case CheckWarn:
			warnCount++
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	if failCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Result: %d check(s) failed, %d warning(s)\n", failCount, warnCount)
		fmt.Fprintln(cmd.OutOrStdout(), "Fix the issues above and run dk doctor again.")
		return fmt.Errorf("%d check(s) failed", failCount)
	}
	if warnCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Result: all checks passed with %d warning(s)\n", warnCount)
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Result: all checks passed — your environment is ready!")
	return nil
}

// printResults prints the check results to the command output.
func printResults(cmd *cobra.Command, checks []CheckResult) {
	for _, c := range checks {
		icon := statusIcon(c.Status)
		fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s\n", icon, c.Name, c.Message)
		if doctorVerbose && c.Detail != "" {
			for _, line := range strings.Split(c.Detail, "\n") {
				fmt.Fprintf(cmd.OutOrStdout(), "      %s\n", line)
			}
		}
		if c.Status == CheckFail && c.Fix != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "    Fix: %s\n", c.Fix)
		}
	}
}

// gatherChecks runs all doctor checks and collects results.
func gatherChecks(ctx context.Context) []CheckResult {
	var results []CheckResult
	results = append(results, checkGoVersion(ctx))
	results = append(results, checkContainerRuntime(ctx))
	results = append(results, checkK3d(ctx))
	results = append(results, checkKubectl(ctx))
	results = append(results, checkHelm(ctx))
	results = append(results, checkRegistryConnectivity(ctx))
	results = append(results, checkDevStack(ctx))
	return results
}

// checkGoVersion validates that Go is installed and meets the minimum version.
func checkGoVersion(ctx context.Context) CheckResult {
	result := CheckResult{Name: "Go"}

	goCmd := exec.CommandContext(ctx, "go", "version")
	out, err := goCmd.Output()
	if err != nil {
		result.Status = CheckFail
		result.Message = "Go not found"
		result.Fix = "Install Go from https://go.dev/dl/"
		return result
	}

	version := strings.TrimSpace(string(out))
	major, minor, ok := parseGoVersion(version)
	if !ok {
		result.Status = CheckWarn
		result.Message = fmt.Sprintf("installed (%s) — could not parse version", version)
		result.Detail = "Unable to determine if minimum version is met"
		return result
	}

	if major < MinGoMajor || (major == MinGoMajor && minor < MinGoMinor) {
		result.Status = CheckFail
		result.Message = fmt.Sprintf("go%d.%d found, minimum go%d.%d required",
			major, minor, MinGoMajor, MinGoMinor)
		result.Fix = fmt.Sprintf("Upgrade Go to %d.%d+ from https://go.dev/dl/", MinGoMajor, MinGoMinor)
		return result
	}

	result.Status = CheckOK
	result.Message = fmt.Sprintf("go%d.%d", major, minor)
	result.Detail = version
	return result
}

// goVersionRe matches "go1.25" or "go1.25.3" etc.
var goVersionRe = regexp.MustCompile(`go(\d+)\.(\d+)`)

// parseGoVersion parses major.minor from a "go version" output string.
func parseGoVersion(versionOutput string) (major, minor int, ok bool) {
	matches := goVersionRe.FindStringSubmatch(versionOutput)
	if len(matches) < 3 {
		return 0, 0, false
	}
	major, err1 := strconv.Atoi(matches[1])
	minor, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return major, minor, true
}

// checkContainerRuntime validates Docker or Rancher Desktop is installed and running.
func checkContainerRuntime(ctx context.Context) CheckResult {
	result := CheckResult{Name: "Container Runtime"}

	cr := localdev.GetContainerRuntimeName()
	checker := localdev.NewPrerequisiteChecker(localdev.RuntimeK3d)

	// Check if any container runtime binary is available
	switch checker.GetContainerRuntime() {
	case localdev.ContainerRuntimeNone:
		result.Status = CheckFail
		result.Message = "no container runtime found"
		if runtime.GOOS == "darwin" {
			result.Fix = "Install Docker Desktop (https://docker.com/products/docker-desktop) or Rancher Desktop (https://rancherdesktop.io)"
		} else {
			result.Fix = "Install Docker: https://docs.docker.com/engine/install/"
		}
		return result
	}

	// Check if daemon is running
	if !localdev.IsContainerRuntimeRunning(ctx) {
		result.Status = CheckFail
		result.Message = fmt.Sprintf("%s installed but not running", cr)
		result.Fix = fmt.Sprintf("Start %s and try again", cr)
		return result
	}

	// Get version info
	dockerCmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	out, err := dockerCmd.Output()
	ver := strings.TrimSpace(string(out))
	if err != nil || ver == "" {
		ver = "unknown"
	}

	result.Status = CheckOK
	result.Message = fmt.Sprintf("%s %s — daemon running", cr, ver)
	return result
}

// checkK3d validates that k3d is installed.
func checkK3d(ctx context.Context) CheckResult {
	return checkToolVersion(ctx, "k3d", []string{"k3d", "version"},
		"Install k3d: https://k3d.io/#installation")
}

// checkKubectl validates that kubectl is installed.
func checkKubectl(ctx context.Context) CheckResult {
	return checkToolVersion(ctx, "kubectl", []string{"kubectl", "version", "--client", "--short"},
		"Install kubectl: https://kubernetes.io/docs/tasks/tools/")
}

// checkHelm validates that helm is installed.
func checkHelm(ctx context.Context) CheckResult {
	return checkToolVersion(ctx, "helm", []string{"helm", "version", "--short"},
		"Install helm: https://helm.sh/docs/intro/install/")
}

// checkToolVersion is a generic helper that runs a version command and reports availability.
func checkToolVersion(ctx context.Context, name string, cmd []string, fixMsg string) CheckResult {
	result := CheckResult{Name: name}

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	if err := execCmd.Run(); err != nil {
		result.Status = CheckFail
		result.Message = fmt.Sprintf("%s not found", name)
		result.Fix = fixMsg
		return result
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		out = strings.TrimSpace(stderr.String())
	}
	// Take first line, strip common prefixes
	if lines := strings.SplitN(out, "\n", 2); len(lines) > 0 {
		out = strings.TrimSpace(lines[0])
	}

	result.Status = CheckOK
	result.Message = out
	return result
}

// checkRegistryConnectivity validates network connectivity to OCI registries.
func checkRegistryConnectivity(ctx context.Context) CheckResult {
	result := CheckResult{Name: "Registry Connectivity"}

	// Determine registries to check
	registries := []string{"ghcr.io:443"}

	// Also check configured registry from dk config
	config, err := localdev.LoadConfig()
	if err == nil && config != nil && config.Plugins.Registry != "" {
		host := registryHost(config.Plugins.Registry)
		if host != "" && host != "ghcr.io:443" {
			registries = append(registries, host)
		}
	}

	var passed, failed []string
	for _, reg := range registries {
		conn, err := net.DialTimeout("tcp", reg, 5*time.Second)
		if err != nil {
			failed = append(failed, reg)
			continue
		}
		conn.Close()
		passed = append(passed, reg)
	}

	if len(failed) > 0 {
		result.Status = CheckWarn
		result.Message = fmt.Sprintf("cannot reach %s", strings.Join(failed, ", "))
		result.Detail = fmt.Sprintf("Reachable: %s\nUnreachable: %s",
			strings.Join(passed, ", "), strings.Join(failed, ", "))
		result.Fix = "Check your network connection and firewall/proxy settings"
		return result
	}

	result.Status = CheckOK
	result.Message = fmt.Sprintf("reachable (%s)", strings.Join(passed, ", "))
	return result
}

// registryHost normalises a registry string to host:port for dialing.
func registryHost(raw string) string {
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	// Drop path component
	if idx := strings.Index(raw, "/"); idx >= 0 {
		raw = raw[:idx]
	}
	if raw == "" {
		return ""
	}
	// Add default port if missing
	if !strings.Contains(raw, ":") {
		raw += ":443"
	}
	return raw
}

// checkDevStack validates the k3d dev cluster and service health.
func checkDevStack(ctx context.Context) CheckResult {
	result := CheckResult{Name: "Dev Stack"}

	manager, err := localdev.NewK3dManager("dk-local")
	if err != nil {
		result.Status = CheckWarn
		result.Message = "could not initialise k3d manager"
		result.Detail = err.Error()
		return result
	}

	status, err := manager.Status(ctx)
	if err != nil {
		result.Status = CheckWarn
		result.Message = "dev stack not running (dk dev up to start)"
		result.Detail = err.Error()
		return result
	}

	if !status.Running {
		result.Status = CheckWarn
		result.Message = "dev stack not running (dk dev up to start)"
		return result
	}

	// Count healthy vs unhealthy services
	var healthy, unhealthy int
	var details []string
	for _, svc := range status.Services {
		if svc.Health == "healthy" || svc.Health == "running" || svc.Status == "Running" {
			healthy++
		} else {
			unhealthy++
		}
		details = append(details, fmt.Sprintf("%s: %s (%s)", svc.Name, svc.Status, svc.Health))
	}

	if unhealthy > 0 {
		result.Status = CheckWarn
		result.Message = fmt.Sprintf("running — %d/%d services healthy",
			healthy, healthy+unhealthy)
		result.Detail = strings.Join(details, "\n")
		result.Fix = "Run dk dev status for details, or dk dev down --volumes && dk dev up to restart"
		return result
	}

	total := healthy + unhealthy
	if total == 0 {
		result.Status = CheckOK
		result.Message = "cluster running (no services deployed)"
		return result
	}

	result.Status = CheckOK
	result.Message = fmt.Sprintf("running — %d/%d services healthy", healthy, total)
	result.Detail = strings.Join(details, "\n")
	return result
}
