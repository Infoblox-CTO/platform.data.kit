package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/cli/internal/templates"
)

const makefileCommonHashFile = ".datakit/Makefile.common.sha256"

// syncMakefileCommon ensures .datakit/Makefile.common is up-to-date in the
// given package directory. It compares the SHA-256 hash of the on-disk file
// against the hash of the freshly rendered template. If they differ (or the
// file is missing), it writes the new content and hash file.
//
// Returns true if the file was updated, false if it was already current.
func syncMakefileCommon(dir string) (updated bool, err error) {
	lang := detectCloudQueryLanguage(dir)

	renderer, err := templates.NewRenderer()
	if err != nil {
		return false, fmt.Errorf("failed to create renderer: %w", err)
	}

	content, expectedHash, err := renderer.RenderMakefileCommon(lang, Version)
	if err != nil {
		return false, fmt.Errorf("failed to render Makefile.common: %w", err)
	}

	commonPath := filepath.Join(dir, ".datakit", "Makefile.common")
	hashPath := filepath.Join(dir, makefileCommonHashFile)

	// Check the actual file content hash — this catches both version changes
	// AND manual tampering.
	if existingContent, readErr := os.ReadFile(commonPath); readErr == nil {
		h := sha256.Sum256(existingContent)
		actualHash := hex.EncodeToString(h[:])
		if actualHash == expectedHash {
			// Content is current. Ensure the hash file also exists.
			if _, statErr := os.Stat(hashPath); os.IsNotExist(statErr) {
				_ = os.WriteFile(hashPath, []byte(expectedHash+"\n"), 0644)
			}
			return false, nil
		}
	}

	// File is missing, stale, or tampered — write fresh content
	datakitDir := filepath.Join(dir, ".datakit")
	if err := os.MkdirAll(datakitDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create .datakit directory: %w", err)
	}

	if err := os.WriteFile(commonPath, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("failed to write Makefile.common: %w", err)
	}

	if err := os.WriteFile(hashPath, []byte(expectedHash+"\n"), 0644); err != nil {
		return false, fmt.Errorf("failed to write hash file: %w", err)
	}

	return true, nil
}
