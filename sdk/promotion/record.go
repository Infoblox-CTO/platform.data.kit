// Package promotion provides services for promoting data packages between environments.
package promotion

import (
	"crypto/rand"
	"fmt"
	"time"
)

// RecordGenerator generates promotion records.
type RecordGenerator struct {
	// Initiator is the user initiating promotions (from git config or env).
	Initiator string
}

// NewRecordGenerator creates a new RecordGenerator.
func NewRecordGenerator() *RecordGenerator {
	return &RecordGenerator{
		Initiator: "unknown",
	}
}

// WithInitiator sets the initiator for generated records.
func (g *RecordGenerator) WithInitiator(initiator string) *RecordGenerator {
	g.Initiator = initiator
	return g
}

// Generate generates a new promotion record.
func (g *RecordGenerator) Generate(req *PromotionRequest, previousVersion string) *PromotionRecord {
	fromEnv := "registry"
	if previousVersion != "" {
		// Infer the source environment based on promotion path
		fromEnv = g.inferSourceEnv(req.TargetEnv)
	}

	return &PromotionRecord{
		ID:          g.generateID(),
		Package:     req.Package,
		Namespace:   req.Namespace,
		Version:     req.Version,
		Digest:      req.Digest,
		FromEnv:     fromEnv,
		ToEnv:       req.TargetEnv,
		Timestamp:   time.Now().UTC(),
		InitiatedBy: g.Initiator,
	}
}

// generateID generates a unique promotion ID.
func (g *RecordGenerator) generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("promo-%s-%x", time.Now().Format("20060102"), b)
}

// inferSourceEnv infers the source environment based on the target.
func (g *RecordGenerator) inferSourceEnv(target Environment) string {
	// Standard promotion path: dev -> int -> prod
	switch target {
	case EnvDev:
		return "registry" // First deployment
	case EnvInt:
		return string(EnvDev)
	case EnvProd:
		return string(EnvInt)
	default:
		return "unknown"
	}
}

// RecordFormatter formats promotion records for display.
type RecordFormatter struct{}

// NewRecordFormatter creates a new RecordFormatter.
func NewRecordFormatter() *RecordFormatter {
	return &RecordFormatter{}
}

// Format formats a promotion record as a string.
func (f *RecordFormatter) Format(record *PromotionRecord) string {
	return fmt.Sprintf(
		"Promotion %s\n"+
			"  Package:     %s\n"+
			"  Version:     %s\n"+
			"  Environment: %s -> %s\n"+
			"  Timestamp:   %s\n"+
			"  Initiated:   %s",
		record.ID,
		record.Package,
		record.Version,
		record.FromEnv,
		record.ToEnv,
		record.Timestamp.Format(time.RFC3339),
		record.InitiatedBy,
	)
}

// FormatCompact formats a promotion record compactly.
func (f *RecordFormatter) FormatCompact(record *PromotionRecord) string {
	return fmt.Sprintf(
		"%s: %s %s -> %s @ %s",
		record.ID,
		record.Package,
		record.FromEnv,
		record.ToEnv,
		record.Version,
	)
}

// RecordStore stores promotion records.
type RecordStore interface {
	// Save saves a promotion record.
	Save(record *PromotionRecord) error
	// Get retrieves a promotion record by ID.
	Get(id string) (*PromotionRecord, error)
	// List lists promotion records for a package.
	List(packageName string, limit int) ([]*PromotionRecord, error)
	// ListByEnv lists promotion records for an environment.
	ListByEnv(env Environment, limit int) ([]*PromotionRecord, error)
}

// InMemoryRecordStore is an in-memory implementation of RecordStore.
type InMemoryRecordStore struct {
	records map[string]*PromotionRecord
}

// NewInMemoryRecordStore creates a new InMemoryRecordStore.
func NewInMemoryRecordStore() *InMemoryRecordStore {
	return &InMemoryRecordStore{
		records: make(map[string]*PromotionRecord),
	}
}

// Save saves a promotion record.
func (s *InMemoryRecordStore) Save(record *PromotionRecord) error {
	s.records[record.ID] = record
	return nil
}

// Get retrieves a promotion record by ID.
func (s *InMemoryRecordStore) Get(id string) (*PromotionRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, fmt.Errorf("record not found: %s", id)
	}
	return record, nil
}

// List lists promotion records for a package.
func (s *InMemoryRecordStore) List(packageName string, limit int) ([]*PromotionRecord, error) {
	var result []*PromotionRecord
	for _, r := range s.records {
		if r.Package == packageName {
			result = append(result, r)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// ListByEnv lists promotion records for an environment.
func (s *InMemoryRecordStore) ListByEnv(env Environment, limit int) ([]*PromotionRecord, error) {
	var result []*PromotionRecord
	for _, r := range s.records {
		if r.ToEnv == env {
			result = append(result, r)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}
