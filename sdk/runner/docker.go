package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/lineage"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"gopkg.in/yaml.v3"
)

func init() {
	RegisterRunner("docker", NewDockerRunner)
}

// DockerRunner executes pipelines using Docker.
type DockerRunner struct {
	mu   sync.RWMutex
	runs map[string]*RunResult
}

// NewDockerRunner creates a new Docker-based runner.
func NewDockerRunner() (Runner, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not found in PATH: %w", err)
	}

	return &DockerRunner{
		runs: make(map[string]*RunResult),
	}, nil
}

// Run executes a pipeline using Docker.
func (r *DockerRunner) Run(ctx context.Context, opts RunOptions) (*RunResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	dkPath := filepath.Join(opts.PackageDir, "dk.yaml")
	dpData, err := os.ReadFile(dkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dk.yaml: %w", err)
	}

	m, kind, err := manifest.ParseManifest(dpData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dk.yaml: %w", err)
	}

	var image string
	var runtimeEnv []contracts.EnvVar
	var mode contracts.Mode
	var timeout string
	var runtime contracts.Runtime
	var inputs, outputs []contracts.AssetRef

	// Only Transform manifests carry runtime/image fields.
	if kind != contracts.KindTransform {
		return nil, fmt.Errorf("only Transform manifests can be executed; got %s", kind)
	}
	t := m.(*contracts.Transform)
	image = t.Spec.Image
	runtimeEnv = t.Spec.Env
	mode = t.Spec.Mode
	timeout = t.Spec.Timeout
	runtime = t.Spec.Runtime
	inputs = t.Spec.Inputs
	outputs = t.Spec.Outputs

	// Handle cloudquery runtime - run via cloudquery CLI instead of Docker
	if runtime == contracts.RuntimeCloudQuery {
		return r.runCloudQuery(ctx, opts, m)
	}

	if runtime == contracts.RuntimeDBT {
		return r.runDBT(ctx, opts, m)
	}

	// Expand environment variables in image name
	image = os.ExpandEnv(image)

	// If image still contains unexpanded variables (e.g., ${REGISTRY}), treat as empty
	// This allows local development to fall back to building from Dockerfile
	if strings.Contains(image, "${") || strings.Contains(image, "$") {
		image = ""
	}

	// Validate the expanded image reference is usable
	// An image like "/foo1:" or empty segments means env vars weren't set
	if image != "" && !isValidImageReference(image) {
		image = ""
	}

	runID := GenerateRunID(m.GetName())
	jobNamespace := m.GetNamespace()
	jobName := m.GetName()

	result := &RunResult{
		RunID:     runID,
		Status:    contracts.RunStatusPending,
		StartTime: time.Now(),
	}

	r.mu.Lock()
	r.runs[runID] = result
	r.mu.Unlock()

	// Helper to emit lineage events
	emitLineage := func(eventType lineage.EventType, runErr error) {
		if opts.LineageEmitter == nil {
			return
		}
		event := lineage.NewEvent(eventType, runID, jobNamespace, jobName)

		// Add input datasets
		for _, input := range inputs {
			dataset := lineage.NewDataset(jobNamespace, input.Asset)
			event.AddInput(dataset)
		}

		// Add output datasets
		for _, output := range outputs {
			dataset := lineage.NewDataset(jobNamespace, output.Asset)
			event.AddOutput(dataset)
		}

		// Add error facet for failures
		if runErr != nil && eventType == lineage.EventTypeFail {
			event.WithErrorFacet(runErr.Error(), string(debug.Stack()))
		}

		if err := opts.LineageEmitter.Emit(ctx, event); err != nil && opts.Output != nil {
			fmt.Fprintf(opts.Output, "Warning: failed to emit lineage event: %v\n", err)
		}
	}

	if image == "" {
		// Detect language and generate Dockerfile internally
		lang := detectPipelineLanguage(opts.PackageDir)
		dockerfileContent := generateDockerfile(lang, opts.PackageDir)

		// Create temp directory for build context
		tempDir, err := os.MkdirTemp("", "dk-build-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp build directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Write Dockerfile to temp location
		dockerfilePath := filepath.Join(tempDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write Dockerfile: %w", err)
		}

		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Generated Dockerfile: %s\n", dockerfilePath)
		}

		// Build version tag with git revision
		versionTag := buildVersionTag(m.GetVersion(), opts.PackageDir)
		imageName := fmt.Sprintf("dk/%s:%s", m.GetName(), versionTag)
		if err := r.buildImageWithDockerfile(ctx, opts.PackageDir, dockerfilePath, imageName, opts.Output); err != nil {
			result.Status = contracts.RunStatusFailed
			result.Error = fmt.Sprintf("failed to build image: %v", err)
			emitLineage(lineage.EventTypeFail, err)
			return result, err
		}
		image = imageName
	}

	// Determine execution mode (defaults to batch)
	execMode := mode.Default()

	// Apply timeout if not set in opts
	if opts.Timeout == 0 && timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			opts.Timeout = d
		}
	}

	if opts.DryRun {
		result.Status = contracts.RunStatusCompleted
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Dry run complete. Would run image: %s\n", image)
		}
		return result, nil
	}

	args := []string{"run", "--rm"}
	args = append(args, "--name", runID)

	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}

	// Add package-derived env vars
	bindingEnvs, err := r.buildEnvVarsFromPackage(opts.PackageDir)
	if err != nil && opts.Output != nil {
		fmt.Fprintf(opts.Output, "Warning: failed to map package env vars: %v\n", err)
	}
	for k, v := range bindingEnvs {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add runtime env vars (override package env)
	for _, env := range runtimeEnv {
		if env.Value != "" {
			args = append(args, "-e", fmt.Sprintf("%s=%s", env.Name, env.Value))
		}
	}

	// Add opts env vars (override all)
	for k, v := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	absPackageDir, _ := filepath.Abs(opts.PackageDir)
	args = append(args, "-v", fmt.Sprintf("%s:/app/package:ro", absPackageDir))

	args = append(args, image)

	// Emit START lineage event
	emitLineage(lineage.EventTypeStart, nil)

	// Dispatch based on execution mode
	var runErr error
	if IsStreamingMode(execMode) {
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Running streaming pipeline (mode: streaming)\n")
		}
		runErr = r.RunStreaming(ctx, opts, image, result, args)
	} else {
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Running batch pipeline (mode: batch)\n")
		}
		runErr = r.RunBatch(ctx, opts, image, result, args)
	}

	// Emit completion lineage event
	if runErr != nil {
		emitLineage(lineage.EventTypeFail, runErr)
	} else if result.Status == contracts.RunStatusCompleted {
		emitLineage(lineage.EventTypeComplete, nil)
	}

	return result, runErr
}

// DefaultCloudQueryImage is the OCI image used to run cloudquery sync.
const DefaultCloudQueryImage = "ghcr.io/cloudquery/cloudquery:latest"

// k3d cluster defaults (duplicated from localdev to avoid import cycle risk).
const (
	defaultClusterName = "dk-local"
	defaultNamespace   = "dk-local"
)

// cqPlugin represents a CloudQuery plugin extracted from config.yaml.
type cqPlugin struct {
	Kind    string // "source" or "destination"
	Name    string
	Image   string // OCI image reference (from registry: docker)
	Port    int    // assigned gRPC port for the sidecar container
	Command string // binary path inside the container (resolved from ENTRYPOINT)
}

// runCloudQuery executes a CloudQuery pipeline as a Kubernetes Job in the
// local k3d cluster. Each plugin from config.yaml runs as a native sidecar
// container in the same pod, serving gRPC. The CloudQuery main container
// connects to the plugins over localhost. This gives the pod native access
// to cluster services (PostgreSQL, LocalStack S3, Redpanda, Marquez) without
// needing a Docker daemon.
func (r *DockerRunner) runCloudQuery(ctx context.Context, opts RunOptions, m manifest.Manifest) (*RunResult, error) {
	t, ok := m.(*contracts.Transform)
	if !ok {
		return nil, fmt.Errorf("runCloudQuery requires a Transform manifest")
	}

	// Auto-generate CloudQuery config.yaml from the manifest graph:
	// Transform → Asset → Store → Connector.
	// When a cell is specified, stores are resolved from k8s instead of store/.
	var cellResolver *CellResolver
	if opts.Cell != "" {
		kubeCtx := opts.KubeContext
		cellResolver = NewCellResolver(opts.Cell, kubeCtx, opts.Output)
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Resolving stores from cell %q (namespace: %s)\n", opts.Cell, cellResolver.cellNamespace())
		}
	}
	configData, plugins, err := generateCQConfigWithCell(t, opts.PackageDir, cellResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CloudQuery config: %w", err)
	}

	// Auto-seed: create tables & load sample data for any input asset that
	// declares dev.seed. This ensures the backing database has the expected
	// schema even after a pod restart (persistence is disabled in dev mode).
	seedOpts := SeedOptions{
		PackageDir: opts.PackageDir,
		Output:     opts.Output,
	}
	if seedResult, err := SeedPackage(ctx, seedOpts); err != nil {
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Warning: dev seed failed: %v\n", err)
		}
	} else if seedResult.AssetsSeeded > 0 && opts.Output != nil {
		fmt.Fprintf(opts.Output, "Seeded %d asset(s), %d row(s)\n",
			seedResult.AssetsSeeded, seedResult.RowsInserted)
	}

	// Determine k3d cluster settings.
	clusterName := defaultClusterName
	namespace := defaultNamespace
	kubeContext := fmt.Sprintf("k3d-%s", clusterName)

	// Try loading user config for cluster name override.
	if cfg, err := loadK3dClusterName(); err == nil && cfg != "" {
		clusterName = cfg
		kubeContext = fmt.Sprintf("k3d-%s", clusterName)
	}

	runID := GenerateRunID(m.GetName())
	configMapName := fmt.Sprintf("cq-config-%s", runID)
	jobName := fmt.Sprintf("cq-%s", runID)

	result := &RunResult{
		RunID:     runID,
		Status:    contracts.RunStatusRunning,
		StartTime: time.Now(),
	}

	r.mu.Lock()
	r.runs[runID] = result
	r.mu.Unlock()

	cqImage := DefaultCloudQueryImage

	if opts.DryRun {
		result.Status = contracts.RunStatusCompleted
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Dry run complete. Would create k8s Job %s in %s/%s\n",
				jobName, kubeContext, namespace)
			fmt.Fprintf(opts.Output, "  CloudQuery image: %s\n", cqImage)
			for _, p := range plugins {
				fmt.Fprintf(opts.Output, "  Plugin sidecar: %s (%s) → grpc://localhost:%d\n",
					p.Name, p.Image, p.Port)
			}
		}
		return result, nil
	}

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Running CloudQuery sync (%s) as k8s Job in cluster %s...\n",
			m.GetName(), clusterName)
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Verify the k3d cluster is reachable.
	if err := verifyCluster(ctx, kubeContext, namespace); err != nil {
		return nil, fmt.Errorf("k3d cluster %q not reachable — is it running? (dk dev up): %w",
			clusterName, err)
	}

	// Cleanup helper — removes ConfigMap and Job on exit.
	cleanup := func() {
		bg := context.Background()
		exec.CommandContext(bg, "kubectl", "--context", kubeContext,
			"delete", "job", jobName, "-n", namespace, "--ignore-not-found").Run()
		exec.CommandContext(bg, "kubectl", "--context", kubeContext,
			"delete", "configmap", configMapName, "-n", namespace, "--ignore-not-found").Run()
	}

	// Step 1b: Resolve the binary entrypoint for each plugin image.
	// Different plugin images have different ENTRYPOINT structures:
	//   - Destination plugins: ENTRYPOINT=["/entrypoint"] CMD=["serve", ...]
	//   - Source plugins:      ENTRYPOINT=["/cq-source-postgres","serve",...] CMD=null
	// We override command to just the binary, then supply consistent args.
	for i := range plugins {
		cmd, err := inspectPluginEntrypoint(ctx, plugins[i].Image)
		if err != nil {
			if opts.Output != nil {
				fmt.Fprintf(opts.Output, "  Warning: could not inspect %s entrypoint: %v\n",
					plugins[i].Image, err)
			}
		}
		plugins[i].Command = cmd
	}

	// Step 2: Import all images into k3d (CQ + plugin sidecars).
	images := []string{cqImage}
	for _, p := range plugins {
		images = append(images, p.Image)
	}
	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Importing %d image(s) into k3d cluster...\n", len(images))
	}
	importArgs := append([]string{"image", "import"}, images...)
	importArgs = append(importArgs, "--cluster", clusterName)
	importCmd := exec.CommandContext(ctx, "k3d", importArgs...)
	if out, err := importCmd.CombinedOutput(); err != nil {
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "  Warning: image import: %s (images may already exist)\n",
				strings.TrimSpace(string(out)))
		}
	}

	// Step 3: Rewrite generated config — registry:docker → registry:grpc with localhost ports.
	rewritten, err := rewriteCQConfigForGRPC(configData, plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to rewrite config for gRPC: %w", err)
	}

	if opts.Output != nil {
		for _, p := range plugins {
			fmt.Fprintf(opts.Output, "  Plugin sidecar: %s → %s (grpc://localhost:%d)\n",
				p.Name, p.Image, p.Port)
		}
	}

	// Step 4: Create ConfigMap from the rewritten config.
	tmpFile, err := os.CreateTemp("", "cq-config-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp config: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(rewritten); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp config: %w", err)
	}
	tmpFile.Close()

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Creating ConfigMap %s...\n", configMapName)
	}
	cmCmd := exec.CommandContext(ctx, "kubectl", "--context", kubeContext,
		"create", "configmap", configMapName,
		"--from-file", fmt.Sprintf("config.yaml=%s", tmpFile.Name()),
		"-n", namespace)
	if out, err := cmCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to create ConfigMap: %s: %w",
			strings.TrimSpace(string(out)), err)
	}

	// Inject LocalStack-compatible AWS credentials for k3d cluster.
	// Sidecars (e.g. S3 plugin) require these to talk to LocalStack.
	envMap := make(map[string]string, len(opts.Env)+3)
	envMap["AWS_ACCESS_KEY_ID"] = "test"
	envMap["AWS_SECRET_ACCESS_KEY"] = "test"
	envMap["AWS_DEFAULT_REGION"] = "us-east-1"
	for k, v := range opts.Env {
		envMap[k] = v // user overrides take precedence
	}

	// Step 5: Build and apply k8s Job with plugin sidecars.
	envYAML := buildJobEnvYAML(envMap)
	jobYAML := buildCloudQueryJobYAML(jobName, namespace, runID, cqImage, configMapName, envYAML, plugins)

	applyCmd := exec.CommandContext(ctx, "kubectl", "--context", kubeContext,
		"apply", "-f", "-")
	applyCmd.Stdin = strings.NewReader(jobYAML)
	if out, err := applyCmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create Job: %s: %w",
			strings.TrimSpace(string(out)), err)
	}

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Job %s created, waiting for pod...\n", jobName)
	}

	// Step 6: Wait for the Job's pod, then stream logs from all containers.
	if err := waitForJobPod(ctx, kubeContext, namespace, runID); err != nil {
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Warning: %v\n", err)
		}
	}

	logsCmd := exec.CommandContext(ctx, "kubectl", "--context", kubeContext,
		"logs", "--follow", "--all-containers",
		"-l", fmt.Sprintf("datakit.infoblox.dev/run-id=%s", runID),
		"-n", namespace)
	if opts.Output != nil {
		logsCmd.Stdout = opts.Output
		logsCmd.Stderr = opts.Output
	}
	logsCmd.Run() // blocks until main container exits

	// Step 7: Determine final Job status.
	succeeded := jobSucceeded(ctx, kubeContext, namespace, jobName)

	cleanup()

	now := time.Now()
	result.EndTime = &now
	result.Duration = now.Sub(result.StartTime)

	if succeeded {
		result.Status = contracts.RunStatusCompleted
	} else {
		result.Status = contracts.RunStatusFailed
		result.Error = "CloudQuery sync job failed"
		result.ExitCode = 1
		return result, fmt.Errorf("cloudquery sync failed")
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// CloudQuery config parsing & rewriting
// ---------------------------------------------------------------------------

// inspectPluginEntrypoint uses `docker inspect` to extract the binary path
// (first element of ENTRYPOINT) from a plugin image. This is needed because
// different plugin images package the binary differently:
//   - Destination plugins: ENTRYPOINT=["/entrypoint"]
//   - Source plugins:      ENTRYPOINT=["/cq-source-postgres","serve",...]
//
// By extracting just the binary, we can set `command: [<binary>]` in the k8s
// pod spec and always supply consistent `args: ["serve", "--address", ...]`.
func inspectPluginEntrypoint(ctx context.Context, image string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", image,
		"--format", "{{json .Config.Entrypoint}}")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect %s: %w", image, err)
	}
	var entrypoint []string
	if err := json.Unmarshal(bytes.TrimSpace(out), &entrypoint); err != nil {
		return "", fmt.Errorf("parsing entrypoint for %s: %w", image, err)
	}
	if len(entrypoint) == 0 {
		return "", nil
	}
	return entrypoint[0], nil
}

// rewriteCQConfigForGRPC takes raw config.yaml bytes and rewrites every
// `registry: docker` / `path: <image>` entry to `registry: grpc` /
// `path: localhost:<port>` so CloudQuery connects to sidecar containers in
// the same pod over gRPC instead of pulling OCI images via Docker.
func rewriteCQConfigForGRPC(configData []byte, plugins []cqPlugin) ([]byte, error) {
	lookup := make(map[string]cqPlugin, len(plugins))
	for _, p := range plugins {
		lookup[p.Name] = p
	}

	var docs []map[string]any
	decoder := yaml.NewDecoder(bytes.NewReader(configData))
	for {
		var doc map[string]any
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		spec, _ := doc["spec"].(map[string]any)
		if spec != nil {
			name, _ := spec["name"].(string)
			if p, ok := lookup[name]; ok {
				spec["registry"] = "grpc"
				spec["path"] = fmt.Sprintf("localhost:%d", p.Port)
			}
		}

		docs = append(docs, doc)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	for _, doc := range docs {
		if err := encoder.Encode(doc); err != nil {
			return nil, err
		}
	}
	encoder.Close()

	return buf.Bytes(), nil
}

// ---------------------------------------------------------------------------
// Kubernetes helpers
// ---------------------------------------------------------------------------

// loadK3dClusterName reads the cluster name from the dk config hierarchy.
// Returns empty string on any error.
func loadK3dClusterName() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cfgPath := filepath.Join(home, ".config", "dk", "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", err
	}
	var cfg struct {
		Dev struct {
			K3d struct {
				ClusterName string `yaml:"clusterName"`
			} `yaml:"k3d"`
		} `yaml:"dev"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return cfg.Dev.K3d.ClusterName, nil
}

// verifyCluster ensures the k3d cluster is running and the namespace exists.
func verifyCluster(ctx context.Context, kubeContext, namespace string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "--context", kubeContext,
		"get", "namespace", namespace, "-o", "name")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kubectl get namespace: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// buildJobEnvYAML returns the YAML fragment for env vars in a Job container.
func buildJobEnvYAML(envs map[string]string) string {
	if len(envs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("          env:\n")
	for k, v := range envs {
		sb.WriteString(fmt.Sprintf("            - name: %s\n              value: %q\n", k, v))
	}
	return sb.String()
}

// buildCloudQueryJobYAML generates the Kubernetes Job manifest for a CQ sync.
// Each plugin runs as a native sidecar (initContainer with restartPolicy: Always)
// serving gRPC on its assigned port. The CQ main container connects to the
// plugins over localhost — no Docker daemon required.
func buildCloudQueryJobYAML(jobName, namespace, runID, cqImage, configMapName, envYAML string, plugins []cqPlugin) string {
	// Build sidecar initContainers for each plugin.
	var sidecars strings.Builder
	if len(plugins) > 0 {
		sidecars.WriteString("      initContainers:\n")
		for _, p := range plugins {
			cmdLine := ""
			if p.Command != "" {
				cmdLine = fmt.Sprintf("          command: [%q]\n", p.Command)
			}
			sidecars.WriteString(fmt.Sprintf(`        - name: plugin-%s
          image: %s
          imagePullPolicy: IfNotPresent
          restartPolicy: Always
%s          args: ["serve", "--address", "0.0.0.0:%d", "--log-level", "info"]
%s`, p.Name, p.Image, cmdLine, p.Port, envYAML))
		}
	}

	return fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/managed-by: dk-cli
    datakit.infoblox.dev/runtime: cloudquery
spec:
  backoffLimit: 0
  template:
    metadata:
      labels:
        datakit.infoblox.dev/run-id: %s
        datakit.infoblox.dev/runtime: cloudquery
    spec:
      restartPolicy: Never
%s      containers:
        - name: cloudquery
          image: %s
          imagePullPolicy: IfNotPresent
          args: ["sync", "/config/config.yaml", "--log-console"]
%s          volumeMounts:
            - name: config
              mountPath: /config
              readOnly: true
      volumes:
        - name: config
          configMap:
            name: %s
`, jobName, namespace, runID, sidecars.String(), cqImage, envYAML, configMapName)
}

// waitForJobPod polls until a pod matching the run-id label is Running/Succeeded/Failed.
func waitForJobPod(ctx context.Context, kubeContext, namespace, runID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	deadline := time.After(120 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("timeout waiting for pod to start")
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "kubectl", "--context", kubeContext,
				"get", "pods",
				"-l", fmt.Sprintf("datakit.infoblox.dev/run-id=%s", runID),
				"-n", namespace,
				"-o", "jsonpath={.items[0].status.phase}")
			out, err := cmd.Output()
			if err != nil {
				continue
			}
			phase := strings.TrimSpace(string(out))
			if phase == "Running" || phase == "Succeeded" || phase == "Failed" {
				return nil
			}
		}
	}
}

// jobSucceeded polls Job status for up to 15 seconds, returning true once
// status.succeeded == 1. With native sidecars the Job controller may need a
// few seconds after the main container exits to mark the Job complete.
func jobSucceeded(ctx context.Context, kubeContext, namespace, jobName string) bool {
	deadline := time.After(15 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		cmd := exec.CommandContext(ctx, "kubectl", "--context", kubeContext,
			"get", "job", jobName,
			"-n", namespace,
			"-o", "jsonpath={.status.succeeded}")
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == "1" {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-deadline:
			return false
		case <-ticker.C:
		}
	}
}

// runDBT executes a dbt pipeline by invoking the dbt CLI directly instead of
// building a Docker image. dbt packages contain SQL/YAML transformations.
func (r *DockerRunner) runDBT(ctx context.Context, opts RunOptions, m manifest.Manifest) (*RunResult, error) {
	dbtBin, err := exec.LookPath("dbt")
	if err != nil {
		return nil, fmt.Errorf("dbt CLI not found in PATH: install it from https://docs.getdbt.com/docs/core/installation-overview\n  %w", err)
	}

	runID := GenerateRunID(m.GetName())
	result := &RunResult{
		RunID:     runID,
		Status:    contracts.RunStatusRunning,
		StartTime: time.Now(),
	}

	r.mu.Lock()
	r.runs[runID] = result
	r.mu.Unlock()

	if opts.DryRun {
		result.Status = contracts.RunStatusCompleted
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Dry run complete. Would run: %s run\n", dbtBin)
		}
		return result, nil
	}

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Running dbt (%s)...\n", m.GetName())
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, dbtBin, "run")
	cmd.Dir = opts.PackageDir

	// Merge environment: inherit current env, add opts env vars
	cmd.Env = os.Environ()
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if opts.Output != nil {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output
	}

	runErr := cmd.Run()

	now := time.Now()
	result.EndTime = &now
	result.Duration = now.Sub(result.StartTime)

	if runErr != nil {
		result.Status = contracts.RunStatusFailed
		result.Error = runErr.Error()
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	} else {
		result.Status = contracts.RunStatusCompleted
	}

	return result, runErr
}

// Stop stops a running pipeline.
func (r *DockerRunner) Stop(ctx context.Context, runID string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", runID)
	return cmd.Run()
}

// Logs streams logs from a pipeline run.
func (r *DockerRunner) Logs(ctx context.Context, runID string, follow bool, output io.Writer) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, runID)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

// Status returns the status of a pipeline run.
func (r *DockerRunner) Status(ctx context.Context, runID string) (*RunResult, error) {
	r.mu.RLock()
	result, ok := r.runs[runID]
	r.mu.RUnlock()

	if !ok {
		result = &RunResult{
			RunID: runID,
		}
	}

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", runID)
	out, err := cmd.Output()
	if err != nil {
		if result.Status == "" {
			result.Status = contracts.RunStatus("unknown")
		}
		return result, nil
	}

	status := strings.TrimSpace(string(out))
	switch status {
	case "running":
		result.Status = contracts.RunStatusRunning
	case "exited":
		result.Status = contracts.RunStatusCompleted
		exitCmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.ExitCode}}", runID)
		exitOut, _ := exitCmd.Output()
		if strings.TrimSpace(string(exitOut)) != "0" {
			result.Status = contracts.RunStatusFailed
		}
	case "created":
		result.Status = contracts.RunStatusPending
	default:
		result.Status = contracts.RunStatus(status)
	}

	return result, nil
}

// buildImage builds a Docker image from a Dockerfile.
func (r *DockerRunner) buildImage(ctx context.Context, dir, imageName string, output io.Writer) error {
	args := []string{"build", "-t", imageName, "."}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir

	if output != nil {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		go func() {
			scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
			for scanner.Scan() {
				fmt.Fprintln(output, scanner.Text())
			}
		}()

		return cmd.Wait()
	}

	return cmd.Run()
}

// buildEnvVarsFromPackage reads the package manifest and returns
// explicit environment variables defined in the spec.
func (r *DockerRunner) buildEnvVarsFromPackage(packageDir string) (map[string]string, error) {
	dkPath := filepath.Join(packageDir, "dk.yaml")
	dpData, err := os.ReadFile(dkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dk.yaml: %w", err)
	}

	m, kind, err := manifest.ParseManifest(dpData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dk.yaml: %w", err)
	}

	// Get explicit env vars from manifest
	explicitEnvs := EnvVarsFromManifest(m, kind)

	return explicitEnvs, nil
}

// isValidImageReference checks if an image reference is valid for docker run.
// Returns false if the image has empty components (e.g., "/foo:" from unexpanded vars).
func isValidImageReference(image string) bool {
	if image == "" {
		return false
	}

	// Check for leading slash (invalid: /foo:tag)
	if strings.HasPrefix(image, "/") {
		return false
	}

	// Check for empty tag (invalid: foo:)
	if strings.HasSuffix(image, ":") {
		return false
	}

	// Check for double slashes (invalid: registry//image)
	if strings.Contains(image, "//") {
		return false
	}

	// Check for empty segments between colons/slashes
	parts := strings.Split(image, "/")
	for _, part := range parts {
		if part == "" {
			return false
		}
	}

	return true
}

// buildVersionTag creates a version tag that includes git revision information.
// Format: <version>-<short-sha>[-dirty]
func buildVersionTag(baseVersion, packageDir string) string {
	gitVersion := getGitVersion(packageDir)
	if gitVersion == "" {
		return baseVersion
	}
	return fmt.Sprintf("%s-%s", baseVersion, gitVersion)
}

// getGitVersion returns the git short SHA and dirty status.
// Returns empty string if not in a git repository.
func getGitVersion(dir string) string {
	// Get short commit hash
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	sha := strings.TrimSpace(string(output))

	// Check if working directory is dirty
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err = cmd.Output()
	if err != nil {
		return sha
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		return sha + "-dirty"
	}
	return sha
}

// detectPipelineLanguage detects the programming language of a pipeline package.
// It checks both the legacy src/ directory layout and the new root-level layout.
func detectPipelineLanguage(packageDir string) string {
	srcDir := filepath.Join(packageDir, "src")

	// Check for Python (src/ layout first, then root)
	if _, err := os.Stat(filepath.Join(srcDir, "main.py")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(srcDir, "requirements.txt")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "main.py")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "requirements.txt")); err == nil {
		return "python"
	}

	// Check for Go (src/ layout first, then cmd/ layout, then root)
	if _, err := os.Stat(filepath.Join(srcDir, "main.go")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(srcDir, "go.mod")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "cmd", "main.go")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "main.go")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "go.mod")); err == nil {
		return "go"
	}

	// Default to Go
	return "go"
}

// hasSrcDir checks if the package directory has a legacy src/ subdirectory.
func hasSrcDir(packageDir string) bool {
	info, err := os.Stat(filepath.Join(packageDir, "src"))
	return err == nil && info.IsDir()
}

// readModulePath reads the module path from go.mod in the given directory.
// Returns the module path (e.g. "my-model") or empty string if go.mod is missing.
func readModulePath(packageDir string) string {
	data, err := os.ReadFile(filepath.Join(packageDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// detectGoBuildTarget determines the correct `go build` target path for a Go
// project. It inspects the directory structure to distinguish between:
//   - Cobra-style layout:  main.go at root, cmd/ is a library package → "."
//   - cmd/ entrypoint:     cmd/main.go is package main, root has no main.go → "./cmd"
//   - flat layout:         main.go at root, no cmd/ → "."
func detectGoBuildTarget(packageDir string) string {
	// Root main.go always wins — this is the standard Go / Cobra convention.
	if _, err := os.Stat(filepath.Join(packageDir, "main.go")); err == nil {
		return "."
	}
	// Check for a main.go inside cmd/ as the entrypoint.
	if _, err := os.Stat(filepath.Join(packageDir, "cmd", "main.go")); err == nil {
		return "./cmd"
	}
	// Fallback: build from root.
	return "."
}

// generateDockerfile generates a Dockerfile for the given language.
// It detects the project layout (src/, cmd/, or flat) and reads go.mod
// to determine the correct build target so imports resolve properly.
func generateDockerfile(lang, packageDir string) string {
	useSrcLayout := hasSrcDir(packageDir)

	switch lang {
	case "python":
		if useSrcLayout {
			return `# DK Pipeline Image (auto-generated)
ARG DK_BASE_IMAGE=python:3.11-slim

FROM python:3.11-slim AS builder
WORKDIR /build
COPY src/requirements.txt ./
RUN pip install --no-cache-dir --target=/deps -r requirements.txt || true

FROM ${DK_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /deps /app/deps
ENV PYTHONPATH=/app/deps
COPY src/ /app/src/
COPY dk.yaml /app/
ENTRYPOINT ["python", "/app/src/main.py"]
`
		}
		return `# DK Pipeline Image (auto-generated)
ARG DK_BASE_IMAGE=python:3.11-slim

FROM python:3.11-slim AS builder
WORKDIR /build
COPY requirements.txt ./
RUN pip install --no-cache-dir --target=/deps -r requirements.txt || true

FROM ${DK_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /deps /app/deps
ENV PYTHONPATH=/app/deps
COPY . /app/
COPY dk.yaml /app/
ENTRYPOINT ["python", "/app/main.py"]
`
	default: // go
		if useSrcLayout {
			return `# DK Pipeline Image (auto-generated)
ARG DK_BASE_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY src/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pipeline .

FROM ${DK_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /pipeline /app/pipeline
COPY dk.yaml /app/
ENTRYPOINT ["/app/pipeline"]
`
		}
		buildTarget := detectGoBuildTarget(packageDir)
		return fmt.Sprintf(`# DK Pipeline Image (auto-generated)
ARG DK_BASE_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pipeline %s

FROM ${DK_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /pipeline /app/pipeline
COPY dk.yaml /app/
ENTRYPOINT ["/app/pipeline"]
`, buildTarget)
	}
}

// buildImageWithDockerfile builds a Docker image using an external Dockerfile path.
func (r *DockerRunner) buildImageWithDockerfile(ctx context.Context, contextDir, dockerfilePath, imageName string, output io.Writer) error {
	args := []string{"build", "-t", imageName, "-f", dockerfilePath, "."}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = contextDir

	if output != nil {
		cmd.Stdout = output
		cmd.Stderr = output
	}

	return cmd.Run()
}
