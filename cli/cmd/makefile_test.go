package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncMakefileCommon_CreatesFiles(t *testing.T) {
	// Given a directory with a pyproject.toml (Python CloudQuery project)
	// and no .datakit/ directory, syncMakefileCommon should create
	// .datakit/Makefile.common and .datakit/Makefile.common.sha256
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644)

	updated, err := syncMakefileCommon(tmpDir)
	if err != nil {
		t.Fatalf("syncMakefileCommon() error: %v", err)
	}
	if !updated {
		t.Error("expected updated=true for fresh directory")
	}

	// Verify files exist
	commonPath := filepath.Join(tmpDir, ".datakit", "Makefile.common")
	data, err := os.ReadFile(commonPath)
	if err != nil {
		t.Fatalf("expected .datakit/Makefile.common to exist: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "DO NOT EDIT") {
		t.Error("Makefile.common should contain DO NOT EDIT warning")
	}
	if !strings.Contains(content, "pytest") {
		t.Error("Python Makefile.common should contain pytest target")
	}

	hashPath := filepath.Join(tmpDir, ".datakit", "Makefile.common.sha256")
	hashData, err := os.ReadFile(hashPath)
	if err != nil {
		t.Fatalf("expected hash file to exist: %v", err)
	}
	if len(strings.TrimSpace(string(hashData))) != 64 {
		t.Errorf("expected 64-char SHA-256 hash, got %q", strings.TrimSpace(string(hashData)))
	}
}

func TestSyncMakefileCommon_GoLanguage(t *testing.T) {
	// Given a directory with a go.mod (Go CloudQuery project)
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.25\n"), 0644)

	updated, err := syncMakefileCommon(tmpDir)
	if err != nil {
		t.Fatalf("syncMakefileCommon() error: %v", err)
	}
	if !updated {
		t.Error("expected updated=true for fresh directory")
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".datakit", "Makefile.common"))
	if err != nil {
		t.Fatalf("expected .datakit/Makefile.common to exist: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "go test") {
		t.Error("Go Makefile.common should contain go test target")
	}
	if !strings.Contains(content, "go fmt") {
		t.Error("Go Makefile.common should contain go fmt target")
	}
}

func TestSyncMakefileCommon_SkipsWhenCurrent(t *testing.T) {
	// First call creates, second call with same version should skip
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.25\n"), 0644)

	updated1, err := syncMakefileCommon(tmpDir)
	if err != nil {
		t.Fatalf("first syncMakefileCommon() error: %v", err)
	}
	if !updated1 {
		t.Error("expected updated=true on first call")
	}

	updated2, err := syncMakefileCommon(tmpDir)
	if err != nil {
		t.Fatalf("second syncMakefileCommon() error: %v", err)
	}
	if updated2 {
		t.Error("expected updated=false on second call (file is current)")
	}
}

func TestSyncMakefileCommon_UpdatesWhenStale(t *testing.T) {
	// Simulate a stale file by writing different content
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.25\n"), 0644)

	// Create initial version
	updated, err := syncMakefileCommon(tmpDir)
	if err != nil {
		t.Fatalf("syncMakefileCommon() error: %v", err)
	}
	if !updated {
		t.Fatal("expected initial update")
	}

	// Tamper with the content
	commonPath := filepath.Join(tmpDir, ".datakit", "Makefile.common")
	os.WriteFile(commonPath, []byte("# tampered content\n"), 0644)

	// Sync should detect stale content and update
	updated, err = syncMakefileCommon(tmpDir)
	if err != nil {
		t.Fatalf("syncMakefileCommon() error: %v", err)
	}
	if !updated {
		t.Error("expected updated=true for tampered file")
	}

	// Verify it was restored
	data, _ := os.ReadFile(commonPath)
	if !strings.Contains(string(data), "DO NOT EDIT") {
		t.Error("file should be restored to managed content")
	}
}

func TestSyncMakefileCommon_RecoversMissingHashFile(t *testing.T) {
	// If the hash file is deleted but the content is still correct,
	// syncMakefileCommon should just recreate the hash file
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.25\n"), 0644)

	// Create initial version
	syncMakefileCommon(tmpDir)

	// Delete just the hash file
	hashPath := filepath.Join(tmpDir, ".datakit", "Makefile.common.sha256")
	os.Remove(hashPath)

	// Should not report updated (content matches)
	updated, err := syncMakefileCommon(tmpDir)
	if err != nil {
		t.Fatalf("syncMakefileCommon() error: %v", err)
	}
	if updated {
		t.Error("expected updated=false when content is already current (just hash file missing)")
	}

	// Hash file should be recreated
	if _, err := os.Stat(hashPath); os.IsNotExist(err) {
		t.Error("expected hash file to be recreated")
	}
}
