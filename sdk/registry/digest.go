package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// DigestCalculator calculates digests for artifact content.
type DigestCalculator struct{}

// NewDigestCalculator creates a new digest calculator.
func NewDigestCalculator() *DigestCalculator {
	return &DigestCalculator{}
}

// CalculateDigest calculates the SHA256 digest of content.
func (d *DigestCalculator) CalculateDigest(content []byte) string {
	hash := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(hash[:])
}

// CalculateManifestDigest calculates the digest for an artifact manifest.
func (d *DigestCalculator) CalculateManifestDigest(artifact *Artifact) (string, error) {
	// Collect all layer digests
	var layerDigests []string
	for _, layer := range artifact.Layers {
		layerDigests = append(layerDigests, d.CalculateDigest(layer.Content))
	}

	// Sort for deterministic output
	sort.Strings(layerDigests)

	// Combine layer digests
	combined := ""
	for _, digest := range layerDigests {
		combined += digest
	}

	return d.CalculateDigest([]byte(combined)), nil
}

// VerifyDigest verifies that content matches an expected digest.
func (d *DigestCalculator) VerifyDigest(content []byte, expected string) bool {
	actual := d.CalculateDigest(content)
	return actual == expected
}

// ImmutabilityError is returned when attempting to overwrite an existing tag.
type ImmutabilityError struct {
	Reference      string
	ExistingDigest string
	Message        string
}

func (e *ImmutabilityError) Error() string {
	return fmt.Sprintf("immutability violation: %s - tag already exists with digest %s",
		e.Reference, e.ExistingDigest)
}

// CheckImmutability checks if a reference already exists and returns an error if so.
// This implements the tag immutability policy required by the platform.
func CheckImmutability(client Client, reference string) error {
	ctx := client.(*OrasClient) // Type assertion to access internal method
	exists, err := ctx.Exists(nil, reference)
	if err != nil {
		return fmt.Errorf("failed to check immutability: %w", err)
	}

	if exists {
		digest, _ := ctx.Resolve(nil, reference)
		return &ImmutabilityError{
			Reference:      reference,
			ExistingDigest: digest,
			Message:        "tag already exists and cannot be overwritten",
		}
	}

	return nil
}

// ParseReference parses an OCI reference into its components.
type ParsedReference struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
}

// ParseRef parses an OCI reference string.
func ParseRef(ref string) (*ParsedReference, error) {
	parsed := &ParsedReference{}

	// Simple parsing - in production would use a proper OCI reference parser
	// Format: registry/repository:tag or registry/repository@digest

	// Find @ for digest reference
	atIdx := -1
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '@' {
			atIdx = i
			break
		}
	}

	// Find : for tag reference
	colonIdx := -1
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == ':' {
			colonIdx = i
			break
		}
	}

	if atIdx > 0 {
		parsed.Digest = ref[atIdx+1:]
		ref = ref[:atIdx]
	} else if colonIdx > 0 {
		// Check if it's a port number or a tag
		hasSlash := false
		for i := colonIdx; i < len(ref); i++ {
			if ref[i] == '/' {
				hasSlash = true
				break
			}
		}
		if !hasSlash {
			parsed.Tag = ref[colonIdx+1:]
			ref = ref[:colonIdx]
		}
	}

	// Split registry from repository
	slashIdx := -1
	for i := 0; i < len(ref); i++ {
		if ref[i] == '/' {
			slashIdx = i
			break
		}
	}

	if slashIdx > 0 {
		parsed.Registry = ref[:slashIdx]
		parsed.Repository = ref[slashIdx+1:]
	} else {
		parsed.Repository = ref
	}

	return parsed, nil
}

// FormatReference formats a parsed reference back to a string.
func (p *ParsedReference) FormatReference() string {
	ref := ""
	if p.Registry != "" {
		ref = p.Registry + "/"
	}
	ref += p.Repository

	if p.Digest != "" {
		ref += "@" + p.Digest
	} else if p.Tag != "" {
		ref += ":" + p.Tag
	}

	return ref
}

// WithTag returns a new reference with the specified tag.
func (p *ParsedReference) WithTag(tag string) *ParsedReference {
	return &ParsedReference{
		Registry:   p.Registry,
		Repository: p.Repository,
		Tag:        tag,
	}
}

// WithDigest returns a new reference with the specified digest.
func (p *ParsedReference) WithDigest(digest string) *ParsedReference {
	return &ParsedReference{
		Registry:   p.Registry,
		Repository: p.Repository,
		Digest:     digest,
	}
}
