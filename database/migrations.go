// Package database provides database migration utilities.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a single migration.
type Migration struct {
	Version     int
	Name        string
	UpScript    string
	DownScript  string
	ExecutedAt  *time.Time
}

// Migrator handles database migrations.
type Migrator struct {
	db           *SQLClient
	tableName    string
	migrations   []Migration
}

// MigratorOption configures the migrator.
type MigratorOption func(*Migrator)

// WithTableName sets the migrations tracking table name.
func WithTableName(name string) MigratorOption {
	return func(m *Migrator) {
		m.tableName = name
	}
}

// NewMigrator creates a new migrator.
func NewMigrator(db *SQLClient, opts ...MigratorOption) *Migrator {
	m := &Migrator{
		db:        db,
		tableName: "_migrations",
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// LoadFromFS loads migrations from an embedded filesystem.
// Expected structure: migrations/001_create_users.up.sql, migrations/001_create_users.down.sql
func (m *Migrator) LoadFromFS(fsys embed.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrationsMap := make(map[int]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse filename: 001_create_users.up.sql or 001_create_users.down.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		content, err := fs.ReadFile(fsys, filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		if _, ok := migrationsMap[version]; !ok {
			migrationsMap[version] = &Migration{Version: version}
		}

		migration := migrationsMap[version]

		if strings.Contains(name, ".up.") {
			migration.UpScript = string(content)
			// Extract name from filename
			nameWithoutExt := strings.TrimSuffix(parts[1], ".up.sql")
			migration.Name = nameWithoutExt
		} else if strings.Contains(name, ".down.") {
			migration.DownScript = string(content)
		}
	}

	// Convert map to sorted slice
	m.migrations = make([]Migration, 0, len(migrationsMap))
	for _, migration := range migrationsMap {
		m.migrations = append(m.migrations, *migration)
	}
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	return nil
}

// AddMigration adds a migration programmatically.
func (m *Migrator) AddMigration(version int, name, up, down string) {
	m.migrations = append(m.migrations, Migration{
		Version:    version,
		Name:       name,
		UpScript:   up,
		DownScript: down,
	})
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
}

// Initialize creates the migrations tracking table.
func (m *Migrator) Initialize(ctx context.Context) error {
	query := fmt.Sprintf(`
		IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='%s' AND xtype='U')
		CREATE TABLE %s (
			version INT PRIMARY KEY,
			name NVARCHAR(255) NOT NULL,
			executed_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
		)
	`, m.tableName, m.tableName)

	_, err := m.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// Status returns the migration status.
func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := m.Initialize(ctx); err != nil {
		return nil, err
	}

	// Get executed migrations
	executed := make(map[int]time.Time)
	query := fmt.Sprintf("SELECT version, executed_at FROM %s", m.tableName)
	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		var executedAt time.Time
		if err := rows.Scan(&version, &executedAt); err != nil {
			return nil, err
		}
		executed[version] = executedAt
	}

	// Build status list
	statuses := make([]MigrationStatus, len(m.migrations))
	for i, migration := range m.migrations {
		status := MigrationStatus{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: false,
		}
		if t, ok := executed[migration.Version]; ok {
			status.Applied = true
			status.ExecutedAt = &t
		}
		statuses[i] = status
	}

	return statuses, nil
}

// MigrationStatus represents the status of a migration.
type MigrationStatus struct {
	Version    int
	Name       string
	Applied    bool
	ExecutedAt *time.Time
}

// Up runs all pending migrations.
func (m *Migrator) Up(ctx context.Context) (int, error) {
	if err := m.Initialize(ctx); err != nil {
		return 0, err
	}

	statuses, err := m.Status(ctx)
	if err != nil {
		return 0, err
	}

	applied := 0
	for i, status := range statuses {
		if status.Applied {
			continue
		}

		migration := m.migrations[i]
		if migration.UpScript == "" {
			return applied, fmt.Errorf("migration %d has no up script", migration.Version)
		}

		if err := m.runMigration(ctx, migration, true); err != nil {
			return applied, fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}

		applied++
	}

	return applied, nil
}

// UpTo runs migrations up to and including the specified version.
func (m *Migrator) UpTo(ctx context.Context, version int) (int, error) {
	if err := m.Initialize(ctx); err != nil {
		return 0, err
	}

	statuses, err := m.Status(ctx)
	if err != nil {
		return 0, err
	}

	applied := 0
	for i, status := range statuses {
		if status.Version > version {
			break
		}
		if status.Applied {
			continue
		}

		migration := m.migrations[i]
		if err := m.runMigration(ctx, migration, true); err != nil {
			return applied, fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}

		applied++
	}

	return applied, nil
}

// Down rolls back the last migration.
func (m *Migrator) Down(ctx context.Context) error {
	if err := m.Initialize(ctx); err != nil {
		return err
	}

	statuses, err := m.Status(ctx)
	if err != nil {
		return err
	}

	// Find last applied migration
	for i := len(statuses) - 1; i >= 0; i-- {
		if statuses[i].Applied {
			migration := m.migrations[i]
			if migration.DownScript == "" {
				return fmt.Errorf("migration %d has no down script", migration.Version)
			}

			return m.runMigration(ctx, migration, false)
		}
	}

	return nil // No migrations to roll back
}

// DownTo rolls back migrations down to but not including the specified version.
func (m *Migrator) DownTo(ctx context.Context, version int) (int, error) {
	if err := m.Initialize(ctx); err != nil {
		return 0, err
	}

	statuses, err := m.Status(ctx)
	if err != nil {
		return 0, err
	}

	rolledBack := 0
	for i := len(statuses) - 1; i >= 0; i-- {
		if !statuses[i].Applied || statuses[i].Version <= version {
			continue
		}

		migration := m.migrations[i]
		if err := m.runMigration(ctx, migration, false); err != nil {
			return rolledBack, fmt.Errorf("rollback of migration %d failed: %w", migration.Version, err)
		}

		rolledBack++
	}

	return rolledBack, nil
}

// Reset rolls back all migrations.
func (m *Migrator) Reset(ctx context.Context) (int, error) {
	return m.DownTo(ctx, 0)
}

// Refresh resets and re-runs all migrations.
func (m *Migrator) Refresh(ctx context.Context) error {
	if _, err := m.Reset(ctx); err != nil {
		return err
	}
	_, err := m.Up(ctx)
	return err
}

func (m *Migrator) runMigration(ctx context.Context, migration Migration, isUp bool) error {
	return m.db.WithTransaction(ctx, func(tx *Transaction) error {
		var script string
		if isUp {
			script = migration.UpScript
		} else {
			script = migration.DownScript
		}

		// Split script by GO statements for SQL Server
		statements := splitStatements(script)
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			if _, err := tx.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("failed to execute statement: %w", err)
			}
		}

		// Update migrations table
		if isUp {
			insertQuery := fmt.Sprintf(
				"INSERT INTO %s (version, name) VALUES (@p1, @p2)",
				m.tableName,
			)
			if _, err := tx.Exec(ctx, insertQuery, migration.Version, migration.Name); err != nil {
				return fmt.Errorf("failed to record migration: %w", err)
			}
		} else {
			deleteQuery := fmt.Sprintf(
				"DELETE FROM %s WHERE version = @p1",
				m.tableName,
			)
			if _, err := tx.Exec(ctx, deleteQuery, migration.Version); err != nil {
				return fmt.Errorf("failed to remove migration record: %w", err)
			}
		}

		return nil
	})
}

// splitStatements splits SQL script by GO statements.
func splitStatements(script string) []string {
	lines := strings.Split(script, "\n")
	var statements []string
	var current strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "GO") {
			if current.Len() > 0 {
				statements = append(statements, current.String())
				current.Reset()
			}
		} else {
			current.WriteString(line)
			current.WriteString("\n")
		}
	}

	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}

// Version returns the current migration version.
func (m *Migrator) Version(ctx context.Context) (int, error) {
	if err := m.Initialize(ctx); err != nil {
		return 0, err
	}

	query := fmt.Sprintf("SELECT MAX(version) FROM %s", m.tableName)
	row := m.db.QueryRow(ctx, query)

	var version sql.NullInt64
	if err := row.Scan(&version); err != nil {
		return 0, err
	}

	if !version.Valid {
		return 0, nil
	}

	return int(version.Int64), nil
}
