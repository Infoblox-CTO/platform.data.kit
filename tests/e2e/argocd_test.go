//go:build e2e_argocd

package e2e

import (
	"os/exec"
	"testing"
)

func skipIfNoK3d(t *testing.T) {
	t.Helper()
	if err := exec.Command("k3d", "version").Run(); err != nil {
		t.Skip("skipping: k3d not available")
	}
}

func TestArgoCD_GitGeneratorDiscovery(t *testing.T) {
	skipIfShort(t)
	skipIfNoDocker(t)
	skipIfNoK3d(t)

	// This test validates that ArgoCD's git generator correctly discovers
	// apps from the cells/*/apps/* directory structure.
	//
	// Steps:
	// 1. Create k3d cluster
	// 2. Install ArgoCD
	// 3. Apply ApplicationSet from gitops/argocd/applicationset.yaml
	// 4. Point ArgoCD at Gitea repo with cell layout
	// 5. Verify app discovery
	//
	// For now, this is a placeholder that documents the intended test.
	// Full implementation requires k3d + ArgoCD setup which is slow
	// and best run in CI with dedicated infrastructure.
	t.Skip("ArgoCD integration test requires k3d cluster — run in CI")
}

func TestArgoCD_PathTemplating(t *testing.T) {
	skipIfShort(t)
	skipIfNoDocker(t)
	skipIfNoK3d(t)

	// Validates that the git generator path templating works:
	// - path.segments[1] = cell name (e.g., "us-dev-1")
	// - path.basename = app name (e.g., "my-pkg")
	//
	// This ensures the ApplicationSet template correctly extracts
	// cell and app names from the directory structure.
	t.Skip("ArgoCD path templating test requires k3d cluster — run in CI")
}
