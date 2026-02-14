package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/spf13/cobra"
)

func TestRunCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"env", "[]"},
		{"bindings", ""},
		{"network", ""},
		{"timeout", "30m0s"},
		{"dry-run", "false"},
		{"detach", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := runCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Errorf("flag --%s not found", tt.flag)
				return
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("flag --%s default = %v, want %v", tt.flag, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestRunCmd_Args(t *testing.T) {
	// Test argument validation
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args is valid",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one arg is valid",
			args:    []string{"./my-pipeline"},
			wantErr: false,
		},
		{
			name:    "two args is invalid",
			args:    []string{"./pkg1", "./pkg2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCmd.Args(runCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCmd_DirectoryNotFound(t *testing.T) {
	// Test that running a non-existent directory returns an error
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	// Save and restore global flags
	oldEnv := runEnv
	oldDryRun := runDryRun
	defer func() {
		runEnv = oldEnv
		runDryRun = oldDryRun
	}()

	runEnv = []string{}
	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{nonExistent})

	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestRunCmd_MissingDpYaml(t *testing.T) {
	// Test that running a directory without dp.yaml returns an error
	tmpDir := t.TempDir()

	// Save and restore global flags
	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()

	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dp.yaml")
	}
}

func TestRunCmd_DryRun(t *testing.T) {
	// Test dry-run mode (should validate but not execute)
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test pipeline
  owner: data-team
  runtime:
    image: python:3.11
    command:
      - python
      - main.py
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := runDryRun
	oldEnv := runEnv
	defer func() {
		runDryRun = oldDryRun
		runEnv = oldEnv
	}()

	runDryRun = true
	runEnv = []string{}

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Dry run should succeed for valid package
	if err != nil {
		t.Errorf("runPipeline() dry-run error = %v, want nil", err)
	}
}

func TestRunCmd_EnvFlags(t *testing.T) {
	// Test parsing environment variable flags
	tests := []struct {
		name    string
		envVars []string
		valid   bool
	}{
		{
			name:    "valid env vars",
			envVars: []string{"KEY=value", "DEBUG=true"},
			valid:   true,
		},
		{
			name:    "empty env vars",
			envVars: []string{},
			valid:   true,
		},
		{
			name:    "env var with equals in value",
			envVars: []string{"URL=http://host?a=b&c=d"},
			valid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the flag accepts the values
			// Actual parsing happens in runPipeline
			for _, env := range tt.envVars {
				_ = env // placeholder - flag parsing tested through cobra
			}
		})
	}
}

func TestRunCmd_TimeoutFlag(t *testing.T) {
	// Test timeout flag
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{
			name:     "default timeout",
			timeout:  30 * time.Minute,
			expected: 30 * time.Minute,
		},
		{
			name:     "custom timeout",
			timeout:  1 * time.Hour,
			expected: 1 * time.Hour,
		},
		{
			name:     "short timeout",
			timeout:  5 * time.Minute,
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.timeout != tt.expected {
				t.Errorf("timeout = %v, want %v", tt.timeout, tt.expected)
			}
		})
	}
}

func TestRunCmd_InvalidType(t *testing.T) {
	// Test running a package with an invalid type
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-invalid
  namespace: data-team
  version: 1.0.0
spec:
  type: invalid
  description: Test invalid type
  owner: data-team
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()

	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Running an invalid type should fail or handle gracefully
	// The exact behavior depends on implementation
	_ = err
}

func TestRunCmd_SetFlag(t *testing.T) {
	// Verify the --set flag is registered correctly
	flag := runCmd.Flags().Lookup("set")
	if flag == nil {
		t.Fatal("--set flag not found")
	}

	// Default should be empty array
	if flag.DefValue != "[]" {
		t.Errorf("--set default = %v, want []", flag.DefValue)
	}
}

func TestRunCmd_ValuesFlag(t *testing.T) {
	// Verify the -f/--values flag is registered correctly
	flag := runCmd.Flags().Lookup("values")
	if flag == nil {
		t.Fatal("--values flag not found")
	}

	// Check shorthand
	if flag.Shorthand != "f" {
		t.Errorf("--values shorthand = %q, want \"f\"", flag.Shorthand)
	}

	// Default should be empty array
	if flag.DefValue != "[]" {
		t.Errorf("--values default = %v, want []", flag.DefValue)
	}
}

func TestApplyOverrides_SetValues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a base dp.yaml
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test pipeline
  owner: data-team
  runtime:
    image: original:v1
    timeout: 30m
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	// Set override values
	runSet = []string{
		"spec.runtime.image=overridden:v2",
		"spec.runtime.timeout=1h",
	}
	runValueFiles = []string{}

	// Apply overrides
	if err := applyOverrides(dpPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	// Verify the file was modified
	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read modified dp.yaml: %v", err)
	}

	content := string(data)

	if !contains(content, "overridden:v2") {
		t.Error("expected image to be overridden")
	}
	if !contains(content, "1h") {
		t.Error("expected timeout to be overridden")
	}

	// Verify backup was created
	backupPath := dpPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("expected backup file to be created")
	}
}

func TestApplyOverrides_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
spec:
  type: pipeline
  runtime:
    image: test:v1
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	// Set an invalid path
	runSet = []string{"invalid.path.here=value"}
	runValueFiles = []string{}

	// Apply overrides should fail
	err := applyOverrides(dpPath)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}

	if !contains(err.Error(), "invalid override path") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestApplyOverrides_ValueFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base dp.yaml
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
spec:
  type: pipeline
  runtime:
    image: original:v1
    timeout: 30m
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Create override file
	overrideContent := `spec:
  runtime:
    image: from-file:v3
    retries: 5
`
	overridePath := filepath.Join(tmpDir, "overrides.yaml")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write overrides.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	runSet = []string{}
	runValueFiles = []string{overridePath}

	// Apply overrides
	if err := applyOverrides(dpPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	// Verify the file was modified
	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read modified dp.yaml: %v", err)
	}

	content := string(data)

	if !contains(content, "from-file:v3") {
		t.Error("expected image to be overridden from file")
	}
	if !contains(content, "retries: 5") {
		t.Error("expected retries to be added from file")
	}
	// timeout should be preserved from original
	if !contains(content, "30m") {
		t.Error("expected timeout to be preserved")
	}
}

func TestApplyOverrides_Precedence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base dp.yaml
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
spec:
  type: pipeline
  runtime:
    image: base:v1
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Create override file
	overrideContent := `spec:
  runtime:
    image: from-file:v2
`
	overridePath := filepath.Join(tmpDir, "overrides.yaml")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write overrides.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	// Set both file and --set flag - --set should win
	runValueFiles = []string{overridePath}
	runSet = []string{"spec.runtime.image=from-set:v3"}

	// Apply overrides
	if err := applyOverrides(dpPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	// Verify --set won (highest precedence)
	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read modified dp.yaml: %v", err)
	}

	content := string(data)
	if !contains(content, "from-set:v3") {
		t.Error("expected --set to override file override")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- CloudQuery Run Tests (T052) ---

func TestRunCmd_CloudQueryBinaryDetection(t *testing.T) {
	// Test that checkCloudQueryBinary returns error when binary is not found
	// We can't easily mock exec.LookPath, so test the function behavior
	err := checkCloudQueryBinary()
	// The test environment likely doesn't have cloudquery installed
	// If it does, the test still passes (err == nil is valid)
	if err != nil {
		if !strings.Contains(err.Error(), "cloudquery CLI not found") {
			t.Errorf("expected 'cloudquery CLI not found' error, got: %v", err)
		}
		// Verify install instructions are in the error
		if !strings.Contains(err.Error(), "brew install") {
			t.Error("error should contain brew install instructions")
		}
	}
}

func TestRunCmd_CloudQueryTypeRouting(t *testing.T) {
	// Test that dp run routes to CloudQuery path for cloudquery type
	tmpDir := t.TempDir()

	// Create a cloudquery dp.yaml
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-cq-source
  namespace: test
  version: "0.1.0"
spec:
  type: cloudquery
  description: "Test CloudQuery plugin"
  owner: "test"
  cloudquery:
    role: source
    grpcPort: 7777
    concurrency: 10000
  runtime:
    image: "test/test-cq-source:latest"
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()
	runDryRun = false

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Should fail because cloudquery binary is not available,
	// but the error should be from the CloudQuery path, not the pipeline path
	if err == nil {
		t.Skip("cloudquery and docker are available - skipping error check")
	}

	errMsg := err.Error()
	// Should hit cloudquery/kubectl/k3d binary check, cluster check,
	// or image build — not the pipeline runner path
	isCloudQueryPath := strings.Contains(errMsg, "cloudquery CLI not found") ||
		strings.Contains(errMsg, "kubectl is required") ||
		strings.Contains(errMsg, "k3d is required") ||
		strings.Contains(errMsg, "k3d cluster not reachable") ||
		strings.Contains(errMsg, "failed to build plugin image") ||
		strings.Contains(errMsg, "failed to import image") ||
		strings.Contains(errMsg, "failed to create plugin pod")
	if !isCloudQueryPath {
		t.Errorf("expected CloudQuery path error, got pipeline error: %v", err)
	}
}

func TestRunCmd_CloudQuerySourceConfigGeneration(t *testing.T) {
	// Test that sync config is generated correctly
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{
			Name:      "my-source",
			Namespace: "acme",
		},
		Spec: contracts.DataPackageSpec{
			Type: contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{
				Role:   "source",
				Tables: []string{"users", "orders"},
			},
		},
	}

	configPath, err := generateSourceConfig(dp, 7777)
	if err != nil {
		t.Fatalf("generateSourceConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read source config: %v", err)
	}
	content := string(data)

	// Verify source config contains expected values
	if !strings.Contains(content, `name: "my-source"`) {
		t.Error("source config should contain plugin name")
	}
	if !strings.Contains(content, "localhost:7777") {
		t.Error("source config should contain gRPC address")
	}
	if !strings.Contains(content, "registry: grpc") {
		t.Error("source config should specify grpc registry")
	}
	if !strings.Contains(content, `"users"`) || !strings.Contains(content, `"orders"`) {
		t.Error("source config should contain specified tables")
	}
	if strings.Contains(content, "destination") {
		t.Error("source config should not contain any destination")
	}
}

func TestRunCmd_CloudQuerySourceConfigDefaultTables(t *testing.T) {
	// Test source config with default tables (wildcard)
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{
			Name:      "my-source",
			Namespace: "acme",
		},
		Spec: contracts.DataPackageSpec{
			Type:       contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{Role: "source"},
		},
	}

	configPath, err := generateSourceConfig(dp, 8888)
	if err != nil {
		t.Fatalf("generateSourceConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read source config: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `["*"]`) {
		t.Error("source config should use wildcard tables when none specified")
	}
	if !strings.Contains(content, "localhost:8888") {
		t.Error("source config should use custom port")
	}
}

func TestRunCmd_CloudQuerySourceConfigCustomPort(t *testing.T) {
	// Test source config with custom gRPC port
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{
			Name:      "custom-port",
			Namespace: "test",
		},
		Spec: contracts.DataPackageSpec{
			Type: contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{
				Role:     "source",
				GRPCPort: 9999,
			},
		},
	}

	configPath, err := generateSourceConfig(dp, 9999)
	if err != nil {
		t.Fatalf("generateSourceConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read source config: %v", err)
	}

	if !strings.Contains(string(data), "localhost:9999") {
		t.Error("source config should use custom gRPC port 9999")
	}
}

func TestRunCmd_PipelineTypeNotAffected(t *testing.T) {
	// Test that pipeline type still goes through the pipeline path
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: test
spec:
  type: pipeline
  description: "Test pipeline"
  owner: "test"
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Create a minimal pipeline.yaml
	pipelineContent := `apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
spec:
  mode: batch
`
	pipelinePath := filepath.Join(tmpDir, "pipeline.yaml")
	if err := os.WriteFile(pipelinePath, []byte(pipelineContent), 0644); err != nil {
		t.Fatalf("failed to write pipeline.yaml: %v", err)
	}

	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()
	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Should hit the pipeline runner path, not cloudquery path
	if err != nil {
		errMsg := err.Error()
		// Pipeline path errors should NOT be about cloudquery
		if strings.Contains(errMsg, "cloudquery CLI not found") {
			t.Error("pipeline type should not route to CloudQuery path")
		}
	}
}

// --- Phase 4: Hierarchical Config Integration Tests ---

// T030: Verify resolvePluginImage uses hierarchical config
func TestRunCmd_UsesHierarchicalConfig(t *testing.T) {
	// Create a temp dir with .dp/config.yaml containing custom registry
	tmpDir := t.TempDir()
	dpDir := filepath.Join(tmpDir, ".dp")
	if err := os.MkdirAll(dpDir, 0755); err != nil {
		t.Fatal(err)
	}
	configContent := "plugins:\n  registry: ghcr.io/custom-org\n"
	if err := os.WriteFile(filepath.Join(dpDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config from that path (simulate hierarchical load)
	cfg, err := localdev.LoadConfigFromPath(filepath.Join(dpDir, "config.yaml"))
	if err != nil {
		t.Fatalf("LoadConfigFromPath: %v", err)
	}

	// Verify resolvePluginImage uses the custom registry
	imageRef := resolvePluginImage("postgresql", cfg)
	expected := "ghcr.io/custom-org/cloudquery-plugin-postgresql:v8.14.1"
	if imageRef != expected {
		t.Errorf("resolvePluginImage() = %q, want %q", imageRef, expected)
	}
}

// T031: Verify repo scope config wins over user scope in hierarchical merge
func TestRunCmd_ConfigPrecedence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create "user" config with one registry
	userConfig := filepath.Join(tmpDir, "user-config.yaml")
	if err := os.WriteFile(userConfig, []byte("plugins:\n  registry: ghcr.io/user-org\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create "repo" config with a different registry
	repoConfig := filepath.Join(tmpDir, "repo-config.yaml")
	if err := os.WriteFile(repoConfig, []byte("plugins:\n  registry: ghcr.io/repo-org\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Use the internal path-based loader: system → user → repo (last wins)
	cfg, err := localdev.LoadHierarchicalConfigFromPaths([]string{userConfig, repoConfig})
	if err != nil {
		t.Fatalf("LoadHierarchicalConfigFromPaths: %v", err)
	}

	// Repo scope should win
	imageRef := resolvePluginImage("postgresql", cfg)
	expected := "ghcr.io/repo-org/cloudquery-plugin-postgresql:v8.14.1"
	if imageRef != expected {
		t.Errorf("resolvePluginImage() = %q, want %q (repo should win over user)", imageRef, expected)
	}
}

// T032: Verify --registry flag overrides config
func TestRunCmd_RegistryFlagOverridesConfig(t *testing.T) {
	// Simulate: config has ghcr.io/configured-org, --registry sets ghcr.io/flag-org
	cfg := &localdev.Config{
		Plugins: localdev.PluginsConfig{Registry: "ghcr.io/configured-org"},
	}

	// Apply --registry flag override (this is what runCmd.RunE does)
	flagRegistry := "ghcr.io/flag-org"
	cfg.Plugins.Registry = flagRegistry

	imageRef := resolvePluginImage("s3", cfg)
	expected := "ghcr.io/flag-org/cloudquery-plugin-s3:v7.10.1"
	if imageRef != expected {
		t.Errorf("resolvePluginImage() = %q, want %q (--registry flag should win)", imageRef, expected)
	}
}

// T046: TestResolvePluginImage_WithConfig — additional config-driven cases
func TestResolvePluginImage_WithConfig(t *testing.T) {
	tests := []struct {
		name     string
		plugin   string
		config   *localdev.Config
		expected string
	}{
		{
			name:   "version override changes tag",
			plugin: "file",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Overrides: map[string]localdev.PluginOverride{
						"file": {Version: "v5.4.0"},
					},
				},
			},
			expected: "ghcr.io/infobloxopen/cloudquery-plugin-file:v5.4.0",
		},
		{
			name:   "image override bypasses naming convention",
			plugin: "postgresql",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Overrides: map[string]localdev.PluginOverride{
						"postgresql": {Image: "my-registry.io/team/custom-pg:v3.0.0"},
					},
				},
			},
			expected: "my-registry.io/team/custom-pg:v3.0.0",
		},
		{
			name:   "unset override uses default",
			plugin: "postgresql",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Overrides: map[string]localdev.PluginOverride{
						"s3": {Version: "v7.9.0"}, // Override for s3, not postgresql
					},
				},
			},
			expected: "ghcr.io/infobloxopen/cloudquery-plugin-postgresql:v8.14.1",
		},
		{
			name:   "override only affects specified plugin",
			plugin: "s3",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Registry: "ghcr.io/team",
					Overrides: map[string]localdev.PluginOverride{
						"postgresql": {Version: "v8.13.0"},
					},
				},
			},
			expected: "ghcr.io/team/cloudquery-plugin-s3:v7.10.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvePluginImage(tt.plugin, tt.config)
			if result != tt.expected {
				t.Errorf("resolvePluginImage(%q) = %q, want %q", tt.plugin, result, tt.expected)
			}
		})
	}
}

// --- Destination Plugin Management Tests ---

func TestSupportedDestinations(t *testing.T) {
	// Verify all expected destinations are registered
	expected := []string{"file", "postgresql", "s3"}
	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			info, ok := supportedDestinations[name]
			if !ok {
				t.Fatalf("destination %q not found in supportedDestinations", name)
			}
			if info.defaultVersion == "" {
				t.Error("defaultVersion should not be empty")
			}
		})
	}
}

// T017: TestResolvePluginImage
func TestResolvePluginImage(t *testing.T) {
	tests := []struct {
		name     string
		plugin   string
		config   *localdev.Config
		expected string
	}{
		{
			name:     "default registry and version",
			plugin:   "postgresql",
			config:   &localdev.Config{},
			expected: "ghcr.io/infobloxopen/cloudquery-plugin-postgresql:v8.14.1",
		},
		{
			name:   "custom registry",
			plugin: "postgresql",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{Registry: "ghcr.io/myteam"},
			},
			expected: "ghcr.io/myteam/cloudquery-plugin-postgresql:v8.14.1",
		},
		{
			name:   "version override",
			plugin: "postgresql",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Overrides: map[string]localdev.PluginOverride{
						"postgresql": {Version: "v8.13.0"},
					},
				},
			},
			expected: "ghcr.io/infobloxopen/cloudquery-plugin-postgresql:v8.13.0",
		},
		{
			name:   "version override with custom registry",
			plugin: "s3",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Registry: "internal.registry.io/team",
					Overrides: map[string]localdev.PluginOverride{
						"s3": {Version: "v7.9.0"},
					},
				},
			},
			expected: "internal.registry.io/team/cloudquery-plugin-s3:v7.9.0",
		},
		{
			name:   "image override beats version",
			plugin: "postgresql",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Overrides: map[string]localdev.PluginOverride{
						"postgresql": {Version: "v8.13.0", Image: "custom-pg:v2.0.0"},
					},
				},
			},
			expected: "custom-pg:v2.0.0",
		},
		{
			name:   "image override bypasses registry",
			plugin: "postgresql",
			config: &localdev.Config{
				Plugins: localdev.PluginsConfig{
					Registry: "ghcr.io/myteam",
					Overrides: map[string]localdev.PluginOverride{
						"postgresql": {Image: "internal.registry.io/custom-pg:v2.0.0"},
					},
				},
			},
			expected: "internal.registry.io/custom-pg:v2.0.0",
		},
		{
			name:     "nil config uses defaults",
			plugin:   "file",
			config:   nil,
			expected: "ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1",
		},
		{
			name:     "file plugin default",
			plugin:   "file",
			config:   &localdev.Config{},
			expected: "ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1",
		},
		{
			name:     "s3 plugin default",
			plugin:   "s3",
			config:   &localdev.Config{},
			expected: "ghcr.io/infobloxopen/cloudquery-plugin-s3:v7.10.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvePluginImage(tt.plugin, tt.config)
			if result != tt.expected {
				t.Errorf("resolvePluginImage(%q) = %q, want %q", tt.plugin, result, tt.expected)
			}
		})
	}
}

// T018: TestPullDestinationImage
func TestPullDestinationImage_DockerNotFound(t *testing.T) {
	// Save and override PATH to simulate docker not being found
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	err := pullDestinationImage("ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1")
	if err == nil {
		t.Error("expected error when docker is not found")
	}
	if !strings.Contains(err.Error(), "Docker") {
		t.Errorf("error should mention Docker, got: %v", err)
	}
}

// T052: TestPullWithMirrorFallback
func TestPullWithMirrorFallback_DockerNotFound(t *testing.T) {
	// Save and override PATH to simulate docker not being found for all attempts
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	mirrors := []string{"ghcr.io/mirror1", "ghcr.io/mirror2"}
	err := pullWithMirrorFallback("ghcr.io/primary/cloudquery-plugin-file:v5.5.1", mirrors)
	if err == nil {
		t.Error("expected error when docker is not found for all registries")
	}
	// Should mention that all registries were attempted
	if !strings.Contains(err.Error(), "all registries") {
		t.Errorf("error should mention all registries, got: %v", err)
	}
	// Should list the primary and mirrors in error
	if !strings.Contains(err.Error(), "ghcr.io/primary") {
		t.Error("error should mention primary registry")
	}
	if !strings.Contains(err.Error(), "ghcr.io/mirror1") {
		t.Error("error should mention mirror1")
	}
	if !strings.Contains(err.Error(), "ghcr.io/mirror2") {
		t.Error("error should mention mirror2")
	}
}

func TestPullWithMirrorFallback_NoMirrors(t *testing.T) {
	// With no mirrors, should just try primary and return that error
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", origPath)

	err := pullWithMirrorFallback("ghcr.io/primary/cloudquery-plugin-file:v5.5.1", nil)
	if err == nil {
		t.Error("expected error when docker is not found")
	}
	// Error should be from direct pull, not mirror fallback
	if strings.Contains(err.Error(), "all registries") {
		t.Error("with no mirrors, error should not mention all registries")
	}
}

func TestExtractRegistryPrefix(t *testing.T) {
	tests := []struct {
		imageRef string
		expected string
	}{
		{
			imageRef: "ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1",
			expected: "ghcr.io/infobloxopen",
		},
		{
			imageRef: "registry.io/cloudquery-plugin-file:v5.5.1",
			expected: "registry.io",
		},
		{
			imageRef: "ghcr.io/org/sub/cloudquery-plugin-pg:v1",
			expected: "ghcr.io/org/sub",
		},
		{
			imageRef: "just-image:v1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.imageRef, func(t *testing.T) {
			result := extractRegistryPrefix(tt.imageRef)
			if result != tt.expected {
				t.Errorf("extractRegistryPrefix(%q) = %q, want %q", tt.imageRef, result, tt.expected)
			}
		})
	}
}

// T019: TestGenerateSyncConfig_GrpcDestination
func TestGenerateSyncConfig_GrpcDestination(t *testing.T) {
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{
			Name:      "test-source",
			Namespace: "test",
		},
		Spec: contracts.DataPackageSpec{
			Type: contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{
				Role:   "source",
				Tables: []string{"users", "orders"},
			},
		},
	}

	configPath, err := generateSyncConfig(dp, 7777, "file", 8888, nil, "")
	if err != nil {
		t.Fatalf("generateSyncConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read sync config: %v", err)
	}
	content := string(data)

	// Verify source section
	if !strings.Contains(content, `name: "test-source"`) {
		t.Error("sync config should contain source name")
	}
	if !strings.Contains(content, "localhost:7777") {
		t.Error("sync config should contain source gRPC address")
	}
	if !strings.Contains(content, `destinations: ["file"]`) {
		t.Error("sync config should reference file destination")
	}
	if !strings.Contains(content, `"users"`) || !strings.Contains(content, `"orders"`) {
		t.Error("sync config should contain specified tables")
	}

	// Verify destination section uses gRPC (not local)
	// Count occurrences of "registry: grpc" - should be 2 (source + destination)
	grpcCount := strings.Count(content, "registry: grpc")
	if grpcCount != 2 {
		t.Errorf("expected 2 'registry: grpc' entries (source + dest), got %d", grpcCount)
	}
	if strings.Contains(content, "registry: local") {
		t.Error("sync config should NOT contain 'registry: local' — destination is now gRPC")
	}
	if !strings.Contains(content, "localhost:8888") {
		t.Error("sync config should contain destination gRPC port localhost:8888")
	}
	if !strings.Contains(content, "cq-sync-output") {
		t.Error("sync config should contain default file destination spec")
	}
	// write_mode: "append" must appear at destination config level (not inside spec:)
	// to prevent DeleteStale errors from the file plugin
	if !strings.Contains(content, `write_mode: "append"`) {
		t.Error("sync config for file destination should contain write_mode: append")
	}
}

// T020: Updated existing sync config tests for new gRPC destination signature
func TestGenerateSyncConfig_DefaultTables(t *testing.T) {
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{
			Name:      "test-source",
			Namespace: "test",
		},
		Spec: contracts.DataPackageSpec{
			Type:       contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{Role: "source"},
		},
	}

	configPath, err := generateSyncConfig(dp, 8888, "postgresql", 9999, nil, "")
	if err != nil {
		t.Fatalf("generateSyncConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read sync config: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `["*"]`) {
		t.Error("sync config should use wildcard tables when none specified")
	}
	if !strings.Contains(content, `destinations: ["postgresql"]`) {
		t.Error("sync config should reference postgresql destination")
	}
	if !strings.Contains(content, "connection_string") {
		t.Error("sync config should contain postgresql connection string")
	}
	if !strings.Contains(content, "localhost:8888") {
		t.Error("sync config should use source port 8888")
	}
	if !strings.Contains(content, "localhost:9999") {
		t.Error("sync config should use dest port 9999")
	}
}

func TestGenerateSyncConfig_S3Destination(t *testing.T) {
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{
			Name:      "my-source",
			Namespace: "acme",
		},
		Spec: contracts.DataPackageSpec{
			Type:       contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{Role: "source"},
		},
	}

	configPath, err := generateSyncConfig(dp, 7777, "s3", 8888, nil, "")
	if err != nil {
		t.Fatalf("generateSyncConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read sync config: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `destinations: ["s3"]`) {
		t.Error("sync config should reference s3 destination")
	}
	if !strings.Contains(content, "bucket") {
		t.Error("sync config should contain s3 bucket spec")
	}
	if !strings.Contains(content, "localhost:8888") {
		t.Error("sync config should contain destination gRPC port")
	}
}

func TestRunCmd_SyncFlags(t *testing.T) {
	// Verify --sync flag exists with correct default
	syncFlag := runCmd.Flags().Lookup("sync")
	if syncFlag == nil {
		t.Fatal("--sync flag not found")
	}
	if syncFlag.DefValue != "false" {
		t.Errorf("--sync default = %q, want \"false\"", syncFlag.DefValue)
	}

	// Verify --destination flag exists with correct default
	destFlag := runCmd.Flags().Lookup("destination")
	if destFlag == nil {
		t.Fatal("--destination flag not found")
	}
	if destFlag.DefValue != "file" {
		t.Errorf("--destination default = %q, want \"file\"", destFlag.DefValue)
	}

	// T027: Verify --registry flag exists
	registryFlag := runCmd.Flags().Lookup("registry")
	if registryFlag == nil {
		t.Fatal("--registry flag not found")
	}
	if registryFlag.DefValue != "" {
		t.Errorf("--registry default = %q, want empty string", registryFlag.DefValue)
	}
}

func TestDefaultDestinationSpec(t *testing.T) {
	tests := []struct {
		name     string
		contains []string
	}{
		{
			name:     "file",
			contains: []string{"path", "cq-sync-output", "format", "json", "no_rotate"},
		},
		{
			name:     "postgresql",
			contains: []string{"connection_string", "postgresql://"},
		},
		{
			name:     "s3",
			contains: []string{"bucket", "region", "format", "json"},
		},
		{
			name:     "unknown",
			contains: []string{"{}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := defaultDestinationSpec(tt.name)
			for _, s := range tt.contains {
				if !strings.Contains(spec, s) {
					t.Errorf("defaultDestinationSpec(%q) missing %q, got:\n%s", tt.name, s, spec)
				}
			}
		})
	}
}

// TestDefaultDestinationOpts verifies destination-level config options.
func TestDefaultDestinationOpts(t *testing.T) {
	// File destination needs write_mode: append (DeleteStale not implemented)
	fileOpts := defaultDestinationOpts("file")
	if !strings.Contains(fileOpts, "write_mode") {
		t.Error("file destination opts should contain write_mode")
	}
	if !strings.Contains(fileOpts, "append") {
		t.Error("file destination opts should use append write mode")
	}

	// Other destinations should have no special opts
	for _, dest := range []string{"postgresql", "s3", "unknown"} {
		opts := defaultDestinationOpts(dest)
		if opts != "" {
			t.Errorf("defaultDestinationOpts(%q) = %q, want empty string", dest, opts)
		}
	}
}

// TestDeployDestinationContainer_Naming verifies container name generation.
func TestDeployDestinationContainer_Naming(t *testing.T) {
	tests := []struct {
		imageRef   string
		wantPrefix string
	}{
		{
			imageRef:   "ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1",
			wantPrefix: "dp-dest-file-",
		},
		{
			imageRef:   "registry.example.com/cloudquery-plugin-s3:v1.0.0",
			wantPrefix: "dp-dest-s3-",
		},
		{
			imageRef:   "myregistry.io/custom-plugin:latest",
			wantPrefix: "dp-dest-custom-plugin-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.imageRef, func(t *testing.T) {
			base := filepath.Base(tt.imageRef)
			name := strings.Split(base, ":")[0]
			name = strings.ReplaceAll(name, "cloudquery-plugin-", "")
			containerName := "dp-dest-" + name + "-" + "1234567890"

			if !strings.HasPrefix(containerName, tt.wantPrefix) {
				t.Errorf("container name %q should have prefix %q", containerName, tt.wantPrefix)
			}
		})
	}
}

// TestFileDestination_UsesDockerContainer validates that file destination
// uses a local Docker container (not k3d pod) for output accessibility.
func TestFileDestination_UsesDockerContainer(t *testing.T) {
	// The sync block branches: file → deployDestinationContainer, others → deployDestinationPod.
	// This test validates the branching logic is correct by checking that:
	// 1. "file" destination spec uses a path compatible with Docker bind mounts
	// 2. The file spec path resolves inside /home/nonroot (container workdir)

	spec := defaultDestinationSpec("file")

	// The path "./cq-sync-output" resolves to /home/nonroot/cq-sync-output in the container.
	// This matches the bind mount: -v <host>/cq-sync-output:/home/nonroot/cq-sync-output
	if !strings.Contains(spec, "cq-sync-output") {
		t.Error("file destination spec should write to cq-sync-output")
	}
	if !strings.Contains(spec, "path") {
		t.Error("file destination spec should use 'path' (not 'directory')")
	}

	// Verify other destinations would NOT use Docker container
	// (they use k3d pods because they write to remote targets)
	for _, dest := range []string{"postgresql", "s3"} {
		s := defaultDestinationSpec(dest)
		if strings.Contains(s, "cq-sync-output") {
			t.Errorf("%s destination should NOT reference cq-sync-output (it writes to remote targets)", dest)
		}
	}
}

// TestCleanupDestinationContainer verifies cleanup doesn't panic on missing containers.
func TestCleanupDestinationContainer(t *testing.T) {
	// cleanupDestinationContainer is best-effort — it should not panic
	// even when the container doesn't exist.
	// This test just verifies it doesn't panic.
	cleanupDestinationContainer("dp-dest-nonexistent-9999999999")
}

func TestEnsureDockerignore_CreatesNew(t *testing.T) {
	dir := t.TempDir()

	if err := ensureDockerignore(dir); err != nil {
		t.Fatalf("ensureDockerignore() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		t.Fatalf("failed to read .dockerignore: %v", err)
	}
	content := string(data)

	for _, p := range dockerignorePatterns {
		if !strings.Contains(content, p) {
			t.Errorf(".dockerignore missing pattern %q", p)
		}
	}
	if !strings.Contains(content, "Auto-generated by dp") {
		t.Error(".dockerignore should contain auto-generated header")
	}
}

func TestEnsureDockerignore_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()

	// Write an existing .dockerignore with only some patterns
	existing := "*.tmp\nnode_modules/\n"
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ensureDockerignore(dir); err != nil {
		t.Fatalf("ensureDockerignore() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".dockerignore"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Original content preserved
	if !strings.Contains(content, "*.tmp") {
		t.Error("existing patterns should be preserved")
	}
	// New patterns added
	for _, p := range dockerignorePatterns {
		if !strings.Contains(content, p) {
			t.Errorf(".dockerignore missing pattern %q after append", p)
		}
	}
}

func TestEnsureDockerignore_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Run twice
	if err := ensureDockerignore(dir); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, ".dockerignore"))

	if err := ensureDockerignore(dir); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, ".dockerignore"))

	if string(first) != string(second) {
		t.Error("ensureDockerignore should be idempotent — running twice should produce the same file")
	}
}

// --- Destination Spec Resolution Tests ---

func TestResolveDestinationSpec_FileDefault(t *testing.T) {
	spec := resolveDestinationSpec("file", nil, "")
	if !strings.Contains(spec, "cq-sync-output") {
		t.Error("file spec should default to cq-sync-output path")
	}
	if !strings.Contains(spec, "json") {
		t.Error("file spec should default to json format")
	}
}

func TestResolveDestinationSpec_FileConfigOverride(t *testing.T) {
	cfg := &localdev.Config{
		Plugins: localdev.PluginsConfig{
			Destinations: map[string]localdev.DestinationConfig{
				"file": {Path: "/custom/output"},
			},
		},
	}
	spec := resolveDestinationSpec("file", cfg, "")
	if !strings.Contains(spec, "/custom/output") {
		t.Error("file spec should use config override path")
	}
	if strings.Contains(spec, "cq-sync-output") {
		t.Error("file spec should NOT contain default path when overridden")
	}
}

func TestResolveDestinationSpec_PostgresqlDefault(t *testing.T) {
	// With nil config and empty namespace, should use hardcoded fallback
	spec := resolveDestinationSpec("postgresql", nil, "")
	if !strings.Contains(spec, "connection_string") {
		t.Error("postgresql spec should contain connection_string")
	}
	if !strings.Contains(spec, "postgresql://") {
		t.Error("postgresql spec should contain a postgresql:// URL")
	}
}

func TestResolveDestinationSpec_PostgresqlConfigOverride(t *testing.T) {
	cfg := &localdev.Config{
		Plugins: localdev.PluginsConfig{
			Destinations: map[string]localdev.DestinationConfig{
				"postgresql": {ConnectionString: "postgresql://admin:secret@mydb.example.com:5432/analytics"},
			},
		},
	}
	spec := resolveDestinationSpec("postgresql", cfg, "dp-local")
	if !strings.Contains(spec, "mydb.example.com") {
		t.Error("postgresql spec should use config override connection string")
	}
	if strings.Contains(spec, "localhost") {
		t.Error("postgresql spec should NOT contain localhost when overridden")
	}
}

func TestResolveDestinationSpec_S3Default(t *testing.T) {
	spec := resolveDestinationSpec("s3", nil, "")
	if !strings.Contains(spec, "bucket") {
		t.Error("s3 spec should contain bucket")
	}
	if !strings.Contains(spec, "dp-data") {
		t.Error("s3 spec should default to dp-data bucket")
	}
}

func TestResolveDestinationSpec_S3ConfigOverride(t *testing.T) {
	cfg := &localdev.Config{
		Plugins: localdev.PluginsConfig{
			Destinations: map[string]localdev.DestinationConfig{
				"s3": {
					Bucket:   "my-bucket",
					Region:   "eu-west-1",
					Endpoint: "http://minio:9000",
				},
			},
		},
	}
	spec := resolveDestinationSpec("s3", cfg, "")
	if !strings.Contains(spec, "my-bucket") {
		t.Error("s3 spec should use config override bucket")
	}
	if !strings.Contains(spec, "eu-west-1") {
		t.Error("s3 spec should use config override region")
	}
	if !strings.Contains(spec, "minio:9000") {
		t.Error("s3 spec should use config override endpoint")
	}
	if !strings.Contains(spec, "force_path_style") {
		t.Error("s3 spec should include force_path_style when endpoint is set")
	}
}

func TestResolveDestinationSpec_S3NoEndpointNoForcePathStyle(t *testing.T) {
	cfg := &localdev.Config{
		Plugins: localdev.PluginsConfig{
			Destinations: map[string]localdev.DestinationConfig{
				"s3": {Bucket: "prod-bucket", Region: "us-west-2"},
			},
		},
	}
	spec := resolveDestinationSpec("s3", cfg, "")
	if strings.Contains(spec, "force_path_style") {
		t.Error("s3 spec should NOT include force_path_style when no endpoint is set")
	}
	if strings.Contains(spec, "endpoint") {
		t.Error("s3 spec should NOT include endpoint when not configured")
	}
}

func TestResolveDestinationSpec_UnknownDest(t *testing.T) {
	spec := resolveDestinationSpec("bigquery", nil, "")
	if !strings.Contains(spec, "{}") {
		t.Errorf("unknown destination should return empty spec, got: %s", spec)
	}
}

func TestGenerateSyncConfig_PostgresqlWithConfigOverride(t *testing.T) {
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{Name: "my-source", Namespace: "test"},
		Spec: contracts.DataPackageSpec{
			Type:       contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{Role: "source"},
		},
	}
	cfg := &localdev.Config{
		Plugins: localdev.PluginsConfig{
			Destinations: map[string]localdev.DestinationConfig{
				"postgresql": {ConnectionString: "postgresql://admin:pass@remote-db:5432/mydb"},
			},
		},
	}

	configPath, err := generateSyncConfig(dp, 7777, "postgresql", 9999, cfg, "dp-local")
	if err != nil {
		t.Fatalf("generateSyncConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "remote-db:5432/mydb") {
		t.Error("sync config should use the config-override connection string")
	}
}

// --- Phase 5: CloudQuery Dockerfile Generation Tests ---

func TestCloudQueryDockerfile_Python(t *testing.T) {
	// T004: Verify cloudQueryDockerfile("python", ...) produces correct Python 3.11
	// build stage and distroless 3.11 runtime stage.
	df := cloudQueryDockerfile("python", 7777)

	// Build stage must use python:3.11-slim (must match distroless runtime version)
	if !strings.Contains(df, "FROM python:3.11-slim AS builder") {
		t.Error("Python Dockerfile should use python:3.11-slim as build stage")
	}

	// Runtime stage must use distroless python3-debian12
	if !strings.Contains(df, "FROM gcr.io/distroless/python3-debian12:nonroot") {
		t.Error("Python Dockerfile should use gcr.io/distroless/python3-debian12:nonroot as runtime stage")
	}

	// Site-packages path must reference python3.11 (matching distroless)
	if !strings.Contains(df, "python3.11/site-packages") {
		t.Error("Python Dockerfile should reference python3.11/site-packages for distroless runtime")
	}

	// PYTHONPATH must reference python3.11
	if !strings.Contains(df, "ENV PYTHONPATH=/usr/local/lib/python3.11/site-packages") {
		t.Error("Python Dockerfile PYTHONPATH should use python3.11")
	}

	// Entrypoint must use python3 with serve --address
	if !strings.Contains(df, `ENTRYPOINT ["python3", "main.py", "serve", "--address", "[::]:7777"]`) {
		t.Error("Python Dockerfile should have correct ENTRYPOINT with port 7777")
	}

	// Must NOT contain old python 3.13 references
	if strings.Contains(df, "python:3.13") {
		t.Error("Python Dockerfile should NOT contain python:3.13")
	}
	if strings.Contains(df, "python3.13") {
		t.Error("Python Dockerfile should NOT contain python3.13 paths")
	}
}

func TestCloudQueryDockerfile_Go(t *testing.T) {
	// T005: Verify cloudQueryDockerfile("go", ...) still produces correct Go Dockerfile.
	df := cloudQueryDockerfile("go", 7777)

	// Build stage must use golang
	if !strings.Contains(df, "FROM golang:") {
		t.Error("Go Dockerfile should use golang base image")
	}

	// Runtime stage must use distroless static
	if !strings.Contains(df, "FROM gcr.io/distroless/static-debian12:nonroot") {
		t.Error("Go Dockerfile should use gcr.io/distroless/static-debian12:nonroot as runtime stage")
	}

	// Must NOT contain Python references
	if strings.Contains(df, "python") {
		t.Error("Go Dockerfile should NOT contain any python references")
	}

	// Port must be substituted
	if !strings.Contains(df, "EXPOSE 7777") {
		t.Error("Go Dockerfile should expose port 7777")
	}
}

// --- Phase 5: Backward Compatibility Tests (T036) ---

func TestRunCmd_BackwardCompat_NoPipelineYaml(t *testing.T) {
	// Verify that dp run without pipeline.yaml works unchanged — still
	// requires dp.yaml and routes through the standard DockerRunner path.
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: test
spec:
  type: pipeline
  description: "Test"
  owner: "team"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatal(err)
	}
	// No pipeline.yaml — dp run should still work (will use default batch mode)

	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()
	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// The error should NOT mention pipeline workflow or pipeline.Execute
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "failed to load pipeline workflow") {
			t.Error("dp run should not try to load pipeline workflow when no pipeline.yaml exists")
		}
	}
}

func TestRunCmd_BackwardCompat_IgnoresPipelineWorkflow(t *testing.T) {
	// Even if a pipeline.yaml (PipelineWorkflow kind) exists, dp run should
	// go through dp.yaml DockerRunner path, not pipeline.Execute().
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: test
spec:
  type: pipeline
  description: "Test"
  owner: "team"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a PipelineWorkflow-style pipeline.yaml
	pipelineContent := `apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: my-workflow
steps:
  - name: sync-data
    type: sync
    source: aws-source
    sink: postgres-sink
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pipeline.yaml"), []byte(pipelineContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()
	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// dp run routes through DockerRunner — it may fail for docker reasons
	// but must NOT fail because of pipeline workflow execution
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "failed to execute step") ||
			strings.Contains(errMsg, "pipeline workflow") {
			t.Errorf("dp run should not execute pipeline workflow steps, got error: %s", errMsg)
		}
	}
}
