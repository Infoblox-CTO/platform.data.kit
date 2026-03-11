package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCertsDir(t *testing.T) {
	dir, err := CertsDir()
	if err != nil {
		t.Fatalf("CertsDir() error: %v", err)
	}

	if dir == "" {
		t.Fatal("CertsDir() returned empty string")
	}

	// Should end with .config/dk/certs
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}

	// Directory should exist
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("CertsDir() directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected directory, got file")
	}
}

func TestCertPaths(t *testing.T) {
	cert, key, err := CertPaths()
	if err != nil {
		t.Fatalf("CertPaths() error: %v", err)
	}

	if cert == "" || key == "" {
		t.Fatal("CertPaths() returned empty paths")
	}

	if filepath.Base(cert) != certFileName {
		t.Errorf("expected cert filename %q, got %q", certFileName, filepath.Base(cert))
	}
	if filepath.Base(key) != keyFileName {
		t.Errorf("expected key filename %q, got %q", keyFileName, filepath.Base(key))
	}

	// Both should be in the same directory
	if filepath.Dir(cert) != filepath.Dir(key) {
		t.Errorf("cert and key are in different directories: %q vs %q", filepath.Dir(cert), filepath.Dir(key))
	}
}

func TestHasCerts_NoCerts(t *testing.T) {
	// Unless the developer has actually run mkcert, certs shouldn't exist
	// This test is best-effort — it may pass even if certs exist
	// The important thing is it doesn't crash
	_ = HasCerts()
}

func TestMkcertAvailable(t *testing.T) {
	// Just verify it doesn't crash — result depends on environment
	_ = MkcertAvailable()
}
