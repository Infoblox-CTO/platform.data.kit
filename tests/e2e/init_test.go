package e2e

import (
	"path/filepath"
	"testing"
)

func TestInit_CreatesPackageStructure(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, err := runDPInDir(t, tmpDir, "init", "my-package")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	pkgDir := filepath.Join(tmpDir, "my-package")
	assertFileExists(t, pkgDir)
	assertFileExists(t, filepath.Join(pkgDir, "dp.yaml"))
}

func TestInit_ValidatesPackageName(t *testing.T) {
	skipIfShort(t)

	tests := []struct {
		name        string
		packageName string
		wantErr     bool
	}{
		{
			name:        "valid name",
			packageName: "valid-package",
			wantErr:     false,
		},
		{
			name:        "valid name with numbers",
			packageName: "package123",
			wantErr:     false,
		},
		{
			name:        "invalid name with spaces",
			packageName: "invalid package",
			wantErr:     true,
		},
		{
			name:        "invalid name with uppercase",
			packageName: "InvalidPackage",
			wantErr:     true,
		},
		{
			name:        "invalid name starting with number",
			packageName: "123package",
			wantErr:     true,
		},
		{
			name:        "invalid name with special chars",
			packageName: "package@name",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := createTempDir(t)

			result, err := runDPInDir(t, tmpDir, "init", tt.packageName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result.ExitCode == 0 {
					t.Errorf("expected error for package name %q, but got success", tt.packageName)
				}
			} else {
				if result.ExitCode != 0 {
					t.Errorf("expected success for package name %q, got exit code %d\nstderr: %s",
						tt.packageName, result.ExitCode, result.Stderr)
				}
			}
		})
	}
}

func TestInit_WithNamespaceFlag(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, err := runDPInDir(t, tmpDir, "init", "--namespace", "custom-namespace", "my-package")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	pkgDir := filepath.Join(tmpDir, "my-package")
	assertFileExists(t, filepath.Join(pkgDir, "dp.yaml"))
	assertFileContains(t, filepath.Join(pkgDir, "dp.yaml"), "namespace: custom-namespace")
}

func TestInit_WithOwnerFlag(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, err := runDPInDir(t, tmpDir, "init", "--owner", "platform-team", "my-package")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	pkgDir := filepath.Join(tmpDir, "my-package")
	assertFileExists(t, filepath.Join(pkgDir, "dp.yaml"))
	assertFileContains(t, filepath.Join(pkgDir, "dp.yaml"), "owner: platform-team")
}
