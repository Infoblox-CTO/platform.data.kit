// Package auth provides authentication handling for DK CLI.
package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/pkg/githubauth"
)

// DockerConfig represents the Docker config.json structure.
type DockerConfig struct {
	Auths       map[string]AuthEntry `json:"auths,omitempty"`
	CredHelpers map[string]string    `json:"credHelpers,omitempty"`
}

// AuthEntry represents an auth entry in Docker config.
type AuthEntry struct {
	Auth          string `json:"auth,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Email         string `json:"email,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

// Credentials represents parsed registry credentials.
type Credentials struct {
	Username string
	Password string
	Token    string
}

// RegistryAuth provides authentication for OCI registries.
type RegistryAuth struct {
	config     *DockerConfig
	configPath string
}

// NewRegistryAuth creates a new registry auth handler.
func NewRegistryAuth() (*RegistryAuth, error) {
	configPath := getDockerConfigPath()
	auth := &RegistryAuth{
		configPath: configPath,
	}

	if _, err := os.Stat(configPath); err == nil {
		config, err := loadDockerConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load docker config: %w", err)
		}
		auth.config = config
	}

	return auth, nil
}

// GetCredentials retrieves credentials for a registry.
func (r *RegistryAuth) GetCredentials(registry string) (*Credentials, error) {
	// First, check environment variables
	if creds := r.getCredentialsFromEnv(registry); creds != nil {
		return creds, nil
	}

	// Then check docker config
	if r.config != nil {
		if creds := r.getCredentialsFromConfig(registry); creds != nil {
			return creds, nil
		}
	}

	return nil, nil // No credentials found
}

// getCredentialsFromEnv gets credentials from environment variables.
func (r *RegistryAuth) getCredentialsFromEnv(registry string) *Credentials {
	// Check for registry-specific environment variables
	// Format: DK_REGISTRY_<HOSTNAME>_USERNAME
	// or generic: DK_REGISTRY_USERNAME
	envPrefix := "DK_REGISTRY"

	// Try generic first
	username := os.Getenv(envPrefix + "_USERNAME")
	password := os.Getenv(envPrefix + "_PASSWORD")
	token := os.Getenv(envPrefix + "_TOKEN")

	if username != "" || token != "" {
		return &Credentials{
			Username: username,
			Password: password,
			Token:    token,
		}
	}

	// Check for well-known registry environment variables
	switch {
	case strings.Contains(registry, "ghcr.io"):
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			return &Credentials{
				Username: "oauth2",
				Token:    token,
			}
		}
		// Fallback: try cached githubauth token (does NOT trigger device flow)
		if org, err := githubauth.DetectOrg(); err == nil {
			if tok, err := githubauth.LoadToken(org); err == nil && tok != nil {
				return &Credentials{
					Username: "oauth2",
					Token:    tok.AccessToken,
				}
			}
		}
	case strings.Contains(registry, "docker.io") || strings.Contains(registry, "hub.docker.com"):
		if username := os.Getenv("DOCKER_USERNAME"); username != "" {
			return &Credentials{
				Username: username,
				Password: os.Getenv("DOCKER_PASSWORD"),
			}
		}
	case strings.Contains(registry, "gcr.io") || strings.Contains(registry, "pkg.dev"):
		if token := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); token != "" {
			// Read service account JSON
			data, err := os.ReadFile(token)
			if err == nil {
				return &Credentials{
					Username: "_json_key",
					Password: string(data),
				}
			}
		}
	case strings.Contains(registry, "ecr.aws") || strings.Contains(registry, "amazonaws.com"):
		// ECR uses AWS credentials - would need AWS SDK
		// For now, rely on docker config
		return nil
	}

	return nil
}

// getCredentialsFromConfig gets credentials from docker config.
func (r *RegistryAuth) getCredentialsFromConfig(registry string) *Credentials {
	if r.config == nil || r.config.Auths == nil {
		return nil
	}

	// Look for exact match first
	if entry, ok := r.config.Auths[registry]; ok {
		return r.parseAuthEntry(entry)
	}

	// Try with https://
	if entry, ok := r.config.Auths["https://"+registry]; ok {
		return r.parseAuthEntry(entry)
	}

	// Try with https:// and /v2/
	if entry, ok := r.config.Auths["https://"+registry+"/v2/"]; ok {
		return r.parseAuthEntry(entry)
	}

	// Try to match partial hostname
	for key, entry := range r.config.Auths {
		if strings.Contains(key, registry) {
			return r.parseAuthEntry(entry)
		}
	}

	return nil
}

// parseAuthEntry parses a docker config auth entry.
func (r *RegistryAuth) parseAuthEntry(entry AuthEntry) *Credentials {
	creds := &Credentials{}

	// Check for token-based auth
	if entry.IdentityToken != "" {
		creds.Token = entry.IdentityToken
		return creds
	}

	if entry.RegistryToken != "" {
		creds.Token = entry.RegistryToken
		return creds
	}

	// Check for username/password
	if entry.Username != "" {
		creds.Username = entry.Username
		creds.Password = entry.Password
		return creds
	}

	// Parse auth field (base64 encoded username:password)
	if entry.Auth != "" {
		decoded, err := base64.StdEncoding.DecodeString(entry.Auth)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				creds.Username = parts[0]
				creds.Password = parts[1]
				return creds
			}
		}
	}

	return creds
}

// getDockerConfigPath returns the path to docker config.json.
func getDockerConfigPath() string {
	// Check DOCKER_CONFIG environment variable
	if configDir := os.Getenv("DOCKER_CONFIG"); configDir != "" {
		return filepath.Join(configDir, "config.json")
	}

	// Check XDG config
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		path := filepath.Join(xdgConfig, "containers", "auth.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to ~/.docker/config.json
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".docker", "config.json")
}

// loadDockerConfig loads and parses docker config.json.
func loadDockerConfig(path string) (*DockerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config DockerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Login stores credentials for a registry.
func (r *RegistryAuth) Login(registry, username, password string) error {
	if r.config == nil {
		r.config = &DockerConfig{
			Auths: make(map[string]AuthEntry),
		}
	}

	// Encode credentials
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	r.config.Auths[registry] = AuthEntry{
		Auth: auth,
	}

	return r.saveConfig()
}

// Logout removes credentials for a registry.
func (r *RegistryAuth) Logout(registry string) error {
	if r.config == nil || r.config.Auths == nil {
		return nil
	}

	delete(r.config.Auths, registry)
	delete(r.config.Auths, "https://"+registry)

	return r.saveConfig()
}

// saveConfig saves the docker config to disk.
func (r *RegistryAuth) saveConfig() error {
	data, err := json.MarshalIndent(r.config, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	configDir := filepath.Dir(r.configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	return os.WriteFile(r.configPath, data, 0600)
}
