package e2e

import (
	"os"
	"strings"
	"testing"
)

func TestPromote_DryRunDefaultCell(t *testing.T) {
	skipIfShort(t)

	result, err := runDK(t, "promote", "my-pkg", "v1.0.0", "--to", "dev", "--dry-run")
	if err != nil {
		t.Fatalf("dk promote dry-run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "DRY RUN") {
		t.Errorf("output should contain DRY RUN, got: %s", result.Stdout)
	}

	// Default cell is c0
	if !strings.Contains(result.Stdout, "envs/dev/cells/c0/apps/my-pkg/values.yaml") {
		t.Errorf("output should contain envs/dev/cells/c0 path, got: %s", result.Stdout)
	}
}

func TestPromote_DryRunNamedCell(t *testing.T) {
	skipIfShort(t)

	result, err := runDK(t, "promote", "my-pkg", "v1.0.0", "--to", "prod", "--cell", "canary", "--dry-run")
	if err != nil {
		t.Fatalf("dk promote dry-run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "envs/prod/cells/canary/apps/my-pkg/values.yaml") {
		t.Errorf("output should contain envs/prod/cells/canary path, got: %s", result.Stdout)
	}
}

func TestPromote_MissingToFlag(t *testing.T) {
	skipIfShort(t)

	result, _ := runDK(t, "promote", "my-pkg", "v1.0.0", "--dry-run")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code when --to is missing")
	}
}

func TestRollback_DryRunDefaultCell(t *testing.T) {
	skipIfShort(t)

	result, err := runDK(t, "rollback", "my-pkg", "--to", "prod", "--to-version", "v0.9.0", "--dry-run")
	if err != nil {
		t.Fatalf("dk rollback dry-run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "DRY RUN") {
		t.Errorf("output should contain DRY RUN, got: %s", result.Stdout)
	}
}

func TestRollback_DryRunNamedCell(t *testing.T) {
	skipIfShort(t)

	result, err := runDK(t, "rollback", "my-pkg", "--to", "prod", "--cell", "canary", "--to-version", "v0.9.0", "--dry-run")
	if err != nil {
		t.Fatalf("dk rollback dry-run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "envs/prod/cells/canary/apps/my-pkg/values.yaml") {
		t.Errorf("output should contain canary cell path, got: %s", result.Stdout)
	}
}

func TestPromote_GiteaIntegration(t *testing.T) {
	skipIfShort(t)
	skipIfNoDocker(t)

	if os.Getenv("DK_E2E_PROMOTE") == "" {
		t.Skip("skipping Gitea integration test (set DK_E2E_PROMOTE=1)")
	}

	gi := startGitea(t, "test-org", "datakit")

	// Seed cell layout
	seedGiteaRepo(t, gi, map[string]string{
		"envs/dev/cells/c0/apps/.gitkeep": "",
	})

	// Set env vars for dk promote
	t.Setenv("GITHUB_TOKEN", gi.Token)
	t.Setenv("GITHUB_OWNER", gi.Org)
	t.Setenv("GITHUB_REPO", gi.Repo)
	t.Setenv("GITHUB_BASE_URL", gi.URL+"/api/v1")

	result, err := runDK(t, "promote", "my-pkg", "v1.0.0", "--to", "dev")
	if err != nil {
		t.Fatalf("dk promote failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstdout: %s\nstderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "Promotion PR created successfully") {
		t.Errorf("expected success message, got: %s", result.Stdout)
	}
}
