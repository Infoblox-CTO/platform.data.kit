// Package runs provides run tracking and history for DataKit.
package runs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

// Store defines the interface for run record storage.
type Store interface {
	// Create creates a new run record.
	Create(ctx context.Context, run *RunRecord) error
	// Update updates an existing run record.
	Update(ctx context.Context, run *RunRecord) error
	// Get retrieves a run record by ID.
	Get(ctx context.Context, id string) (*RunRecord, error)
	// List lists run records with optional filters.
	List(ctx context.Context, filter *RunFilter, limit, offset int) ([]*RunRecord, int, error)
	// Delete deletes a run record by ID.
	Delete(ctx context.Context, id string) error
}

// InMemoryStore is an in-memory implementation of Store.
type InMemoryStore struct {
	mu   sync.RWMutex
	runs map[string]*RunRecord
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		runs: make(map[string]*RunRecord),
	}
}

// Create implements Store.
func (s *InMemoryStore) Create(ctx context.Context, run *RunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.ID]; exists {
		return fmt.Errorf("run already exists: %s", run.ID)
	}

	s.runs[run.ID] = run
	return nil
}

// Update implements Store.
func (s *InMemoryStore) Update(ctx context.Context, run *RunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.ID]; !exists {
		return fmt.Errorf("run not found: %s", run.ID)
	}

	s.runs[run.ID] = run
	return nil
}

// Get implements Store.
func (s *InMemoryStore) Get(ctx context.Context, id string) (*RunRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, exists := s.runs[id]
	if !exists {
		return nil, fmt.Errorf("run not found: %s", id)
	}

	return run, nil
}

// List implements Store.
func (s *InMemoryStore) List(ctx context.Context, filter *RunFilter, limit, offset int) ([]*RunRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*RunRecord
	for _, run := range s.runs {
		if filter != nil {
			if filter.Package != "" && run.Package != filter.Package {
				continue
			}
			if filter.Namespace != "" && run.Namespace != filter.Namespace {
				continue
			}
			if filter.Environment != "" && run.Environment != filter.Environment {
				continue
			}
			if filter.Status != "" && run.Status != filter.Status {
				continue
			}
			if filter.Since != nil && run.StartTime.Before(*filter.Since) {
				continue
			}
			if filter.Until != nil && run.StartTime.After(*filter.Until) {
				continue
			}
		}
		result = append(result, run)
	}

	total := len(result)

	// Apply offset and limit
	if offset > len(result) {
		return nil, total, nil
	}
	result = result[offset:]
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result, total, nil
}

// Delete implements Store.
func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[id]; !exists {
		return fmt.Errorf("run not found: %s", id)
	}

	delete(s.runs, id)
	return nil
}

// PostgresStore is a PostgreSQL implementation of Store.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgresStore.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Create implements Store.
func (s *PostgresStore) Create(ctx context.Context, run *RunRecord) error {
	query := `
		INSERT INTO runs (id, package, namespace, version, environment, status, start_time, records_processed, bytes_processed, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := s.db.ExecContext(ctx, query,
		run.ID, run.Package, run.Namespace, run.Version, run.Environment,
		run.Status, run.StartTime, run.RecordsProcessed, run.BytesProcessed, run.ErrorMessage,
	)
	return err
}

// Update implements Store.
func (s *PostgresStore) Update(ctx context.Context, run *RunRecord) error {
	query := `
		UPDATE runs SET
			status = $2,
			end_time = $3,
			duration_ms = $4,
			records_processed = $5,
			bytes_processed = $6,
			error_message = $7
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query,
		run.ID, run.Status, run.EndTime, run.DurationMs,
		run.RecordsProcessed, run.BytesProcessed, run.ErrorMessage,
	)
	return err
}

// Get implements Store.
func (s *PostgresStore) Get(ctx context.Context, id string) (*RunRecord, error) {
	query := `
		SELECT id, package, namespace, version, environment, status, start_time, end_time, duration_ms, records_processed, bytes_processed, error_message
		FROM runs WHERE id = $1
	`
	run := &RunRecord{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&run.ID, &run.Package, &run.Namespace, &run.Version, &run.Environment,
		&run.Status, &run.StartTime, &run.EndTime, &run.DurationMs,
		&run.RecordsProcessed, &run.BytesProcessed, &run.ErrorMessage,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", id)
	}
	return run, err
}

// List implements Store.
func (s *PostgresStore) List(ctx context.Context, filter *RunFilter, limit, offset int) ([]*RunRecord, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter != nil {
		if filter.Package != "" {
			conditions = append(conditions, fmt.Sprintf("package = $%d", argIdx))
			args = append(args, filter.Package)
			argIdx++
		}
		if filter.Namespace != "" {
			conditions = append(conditions, fmt.Sprintf("namespace = $%d", argIdx))
			args = append(args, filter.Namespace)
			argIdx++
		}
		if filter.Environment != "" {
			conditions = append(conditions, fmt.Sprintf("environment = $%d", argIdx))
			args = append(args, filter.Environment)
			argIdx++
		}
		if filter.Status != "" {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, filter.Status)
			argIdx++
		}
		if filter.Since != nil {
			conditions = append(conditions, fmt.Sprintf("start_time >= $%d", argIdx))
			args = append(args, *filter.Since)
			argIdx++
		}
		if filter.Until != nil {
			conditions = append(conditions, fmt.Sprintf("start_time <= $%d", argIdx))
			args = append(args, *filter.Until)
			argIdx++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM runs %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get results
	query := fmt.Sprintf(`
		SELECT id, package, namespace, version, environment, status, start_time, end_time, duration_ms, records_processed, bytes_processed, error_message
		FROM runs %s
		ORDER BY start_time DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []*RunRecord
	for rows.Next() {
		run := &RunRecord{}
		if err := rows.Scan(
			&run.ID, &run.Package, &run.Namespace, &run.Version, &run.Environment,
			&run.Status, &run.StartTime, &run.EndTime, &run.DurationMs,
			&run.RecordsProcessed, &run.BytesProcessed, &run.ErrorMessage,
		); err != nil {
			return nil, 0, err
		}
		runs = append(runs, run)
	}

	return runs, total, rows.Err()
}

// Delete implements Store.
func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM runs WHERE id = $1"
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", id)
	}
	return nil
}

// MigrateSchema creates the runs table if it doesn't exist.
func (s *PostgresStore) MigrateSchema(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS runs (
			id VARCHAR(64) PRIMARY KEY,
			package VARCHAR(255) NOT NULL,
			namespace VARCHAR(255),
			version VARCHAR(64) NOT NULL,
			environment VARCHAR(32) NOT NULL,
			status VARCHAR(32) NOT NULL,
			start_time TIMESTAMP WITH TIME ZONE NOT NULL,
			end_time TIMESTAMP WITH TIME ZONE,
			duration_ms BIGINT,
			records_processed BIGINT DEFAULT 0,
			bytes_processed BIGINT DEFAULT 0,
			error_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_runs_package ON runs(package);
		CREATE INDEX IF NOT EXISTS idx_runs_environment ON runs(environment);
		CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
		CREATE INDEX IF NOT EXISTS idx_runs_start_time ON runs(start_time DESC);
	`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Ensure implementations satisfy Store interface.
var _ Store = (*InMemoryStore)(nil)
var _ Store = (*PostgresStore)(nil)
