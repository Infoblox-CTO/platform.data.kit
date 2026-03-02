package charts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// DeployCharts deploys all charts using a uniform mechanism.
// It extracts embedded charts, applies config overrides, and runs
// helm upgrade --install in parallel.
// Returns a DeployResult with per-chart success/failure status.
func DeployCharts(ctx context.Context, defs []ChartDefinition, overrides map[string]ChartOverride, kubeContext string) (*DeployResult, error) {
	tempDir, err := os.MkdirTemp("", "dk-charts-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract all charts to temp dir
	chartDirs := make(map[string]string)
	for _, def := range defs {
		chartDir := filepath.Join(tempDir, def.Name)
		if err := ExtractChart(def.Name, chartDir); err != nil {
			return nil, fmt.Errorf("failed to extract chart %s: %w", def.Name, err)
		}
		chartDirs[def.Name] = chartDir
	}

	result := &DeployResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, def := range defs {
		wg.Add(1)
		go func(d ChartDefinition, dir string) {
			defer wg.Done()

			args := buildHelmArgs(d, dir, kubeContext, overrides)
			cmd := exec.CommandContext(ctx, "helm", args...)

			var outBuf bytes.Buffer
			cmd.Stdout = &outBuf
			cmd.Stderr = &outBuf

			if err := cmd.Run(); err != nil {
				mu.Lock()
				result.Failed = append(result.Failed, ChartError{
					ChartName: d.Name,
					Error:     fmt.Errorf("helm install %s failed: %w\n%s", d.Name, err, outBuf.String()),
				})
				mu.Unlock()
				return
			}

			mu.Lock()
			result.Succeeded = append(result.Succeeded, d.Name)
			mu.Unlock()
		}(def, chartDirs[def.Name])
	}

	wg.Wait()
	return result, nil
}

// UninstallCharts uninstalls all chart releases.
func UninstallCharts(ctx context.Context, defs []ChartDefinition, kubeContext string) error {
	var errs []error
	for _, def := range defs {
		cmd := exec.CommandContext(ctx, "helm",
			"uninstall", def.ReleaseName,
			"--kube-context", kubeContext,
			"--namespace", def.Namespace,
			"--ignore-not-found",
		)
		var outBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf
		if err := cmd.Run(); err != nil {
			errs = append(errs, fmt.Errorf("failed to uninstall %s: %w\n%s", def.Name, err, outBuf.String()))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("uninstall errors: %v", errs)
	}
	return nil
}

// buildHelmArgs constructs the helm upgrade --install argument list for a chart.
func buildHelmArgs(def ChartDefinition, chartDir, kubeContext string, overrides map[string]ChartOverride) []string {
	args := []string{
		"upgrade", "--install", def.ReleaseName, chartDir,
		"--kube-context", kubeContext,
		"--namespace", def.Namespace,
		"--create-namespace",
		"--wait",
		"--timeout", "300s",
	}

	// Apply overrides if present
	if overrides != nil {
		if ov, ok := overrides[def.Name]; ok {
			// Version overrides are noted but not applied at chart-fetch time
			// because we use embedded charts. A future enhancement could
			// download the requested version from a chart repository.
			if ov.Version != "" {
				args = append(args, "--version", ov.Version)
			}
			args = ApplyOverrides(args, ov)
		}
	}

	return args
}

// ApplyOverrides merges ChartOverride values into helm install args.
// Version overrides are handled at chart fetch time (not here).
// Value overrides are added as --set flags.
func ApplyOverrides(args []string, override ChartOverride) []string {
	for key, val := range override.Values {
		args = append(args, "--set", fmt.Sprintf("%s=%v", key, val))
	}
	return args
}

// ExtractChart extracts an embedded chart to a destination directory.
func ExtractChart(chartName string, destDir string) error {
	_, err := FS.ReadDir(chartName)
	if err != nil {
		return fmt.Errorf("chart %q not found in embedded FS: %w", chartName, err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	return extractDir(chartName, destDir)
}

// extractDir recursively copies a directory from the embedded FS to disk.
func extractDir(srcPath string, destPath string) error {
	entries, err := FS.ReadDir(srcPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcFile := filepath.Join(srcPath, entry.Name())
		destFile := filepath.Join(destPath, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(destFile, 0755); err != nil {
				return err
			}
			if err := extractDir(srcFile, destFile); err != nil {
				return err
			}
		} else {
			content, err := FS.ReadFile(srcFile)
			if err != nil {
				return err
			}
			if err := os.WriteFile(destFile, content, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
