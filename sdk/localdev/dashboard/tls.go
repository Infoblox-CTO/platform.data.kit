package dashboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	certFileName = "localtest.me+1.pem"
	keyFileName  = "localtest.me+1-key.pem"
)

// CertFileNames returns cert/key filenames for a given domain.
func CertFileNames(domain string) (string, string) {
	return domain + "+1.pem", domain + "+1-key.pem"
}

// CertsDir returns the directory for storing TLS certificates (~/.config/dk/certs/),
// creating it if it doesn't exist.
func CertsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	dir := filepath.Join(home, ".config", "dk", "certs")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create certs directory: %w", err)
	}

	return dir, nil
}

// CertPaths returns the paths to the TLS certificate and key files.
func CertPaths() (cert string, key string, err error) {
	dir, err := CertsDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(dir, certFileName), filepath.Join(dir, keyFileName), nil
}

// HasCerts returns true if both the certificate and key files exist.
func HasCerts() bool {
	cert, key, err := CertPaths()
	if err != nil {
		return false
	}

	if _, err := os.Stat(cert); err != nil {
		return false
	}
	if _, err := os.Stat(key); err != nil {
		return false
	}
	return true
}

// MkcertAvailable returns true if mkcert is on PATH.
func MkcertAvailable() bool {
	_, err := exec.LookPath("mkcert")
	return err == nil
}

// EnsureCerts generates TLS certificates for localtest.me if they don't already exist.
// Uses mkcert to create locally-trusted development certificates.
//
// If mkcert is not installed, returns empty paths and nil error — the caller
// should fall back to HTTP.
//
// If certs already exist, returns their paths without regenerating.
func EnsureCerts() (cert string, key string, err error) {
	return EnsureCertsForDomain(DefaultDomain)
}

// EnsureCertsForDomain generates TLS certificates for the given domain and its
// wildcard (e.g. "mydev.test" and "*.mydev.test").
func EnsureCertsForDomain(domain string) (cert string, key string, err error) {
	dir, err := CertsDir()
	if err != nil {
		return "", "", err
	}

	certName, keyName := CertFileNames(domain)
	certPath := filepath.Join(dir, certName)
	keyPath := filepath.Join(dir, keyName)

	// If certs already exist, reuse them
	if fileExists(certPath) && fileExists(keyPath) {
		return certPath, keyPath, nil
	}

	// Check if mkcert is available
	if !MkcertAvailable() {
		return "", "", nil
	}

	// Install the local CA into the system trust store (idempotent)
	installCmd := exec.Command("mkcert", "-install")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return "", "", fmt.Errorf("mkcert -install failed: %w", err)
	}

	// Generate cert for domain and *.domain
	genCmd := exec.Command("mkcert",
		"-cert-file", certPath,
		"-key-file", keyPath,
		domain,
		"*."+domain,
	)
	genCmd.Stdout = os.Stdout
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		return "", "", fmt.Errorf("mkcert cert generation failed: %w", err)
	}

	return certPath, keyPath, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
