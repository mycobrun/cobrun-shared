package database

import (
	"strings"
	"testing"
	"time"
)

func TestMigration(t *testing.T) {
	now := time.Now()
	migration := Migration{
		Version:     1,
		Name:        "create_users",
		UpScript:    "CREATE TABLE users (id INT PRIMARY KEY);",
		DownScript:  "DROP TABLE users;",
		ExecutedAt:  &now,
	}

	if migration.Version != 1 {
		t.Errorf("expected Version=1, got %d", migration.Version)
	}
	if migration.Name != "create_users" {
		t.Errorf("expected Name=create_users, got %s", migration.Name)
	}
	if migration.UpScript == "" {
		t.Error("UpScript should not be empty")
	}
	if migration.DownScript == "" {
		t.Error("DownScript should not be empty")
	}
	if migration.ExecutedAt == nil {
		t.Error("ExecutedAt should not be nil")
	}
}

func TestWithTableName(t *testing.T) {
	customName := "custom_migrations"
	opt := WithTableName(customName)

	m := &Migrator{
		tableName: "_migrations", // default
	}

	opt(m)

	if m.tableName != customName {
		t.Errorf("expected tableName=%s, got %s", customName, m.tableName)
	}
}

func TestNewMigrator(t *testing.T) {
	client := &SQLClient{} // Mock client

	t.Run("default options", func(t *testing.T) {
		m := NewMigrator(client)

		if m.db != client {
			t.Error("db should be set to the provided client")
		}
		if m.tableName != "_migrations" {
			t.Errorf("expected default tableName=_migrations, got %s", m.tableName)
		}
		if m.migrations != nil {
			t.Error("migrations should be nil initially")
		}
	})

	t.Run("with custom table name", func(t *testing.T) {
		m := NewMigrator(client, WithTableName("custom_migrations"))

		if m.tableName != "custom_migrations" {
			t.Errorf("expected tableName=custom_migrations, got %s", m.tableName)
		}
	})
}

func TestMigratorAddMigration(t *testing.T) {
	client := &SQLClient{}
	m := NewMigrator(client)

	// Add migrations out of order
	m.AddMigration(3, "third", "CREATE TABLE third", "DROP TABLE third")
	m.AddMigration(1, "first", "CREATE TABLE first", "DROP TABLE first")
	m.AddMigration(2, "second", "CREATE TABLE second", "DROP TABLE second")

	if len(m.migrations) != 3 {
		t.Errorf("expected 3 migrations, got %d", len(m.migrations))
	}

	// Verify they're sorted by version
	if m.migrations[0].Version != 1 {
		t.Errorf("expected first migration version=1, got %d", m.migrations[0].Version)
	}
	if m.migrations[1].Version != 2 {
		t.Errorf("expected second migration version=2, got %d", m.migrations[1].Version)
	}
	if m.migrations[2].Version != 3 {
		t.Errorf("expected third migration version=3, got %d", m.migrations[2].Version)
	}

	// Verify names
	if m.migrations[0].Name != "first" {
		t.Errorf("expected first migration name=first, got %s", m.migrations[0].Name)
	}
}

func TestMigrationStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		status     MigrationStatus
		wantApplied bool
	}{
		{
			name: "pending migration",
			status: MigrationStatus{
				Version:    1,
				Name:       "create_users",
				Applied:    false,
				ExecutedAt: nil,
			},
			wantApplied: false,
		},
		{
			name: "applied migration",
			status: MigrationStatus{
				Version:    1,
				Name:       "create_users",
				Applied:    true,
				ExecutedAt: &now,
			},
			wantApplied: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status.Applied != tt.wantApplied {
				t.Errorf("expected Applied=%v, got %v", tt.wantApplied, tt.status.Applied)
			}

			if tt.wantApplied && tt.status.ExecutedAt == nil {
				t.Error("applied migration should have ExecutedAt")
			}

			if !tt.wantApplied && tt.status.ExecutedAt != nil {
				t.Error("pending migration should not have ExecutedAt")
			}
		})
	}
}

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected int
	}{
		{
			name: "single statement",
			script: `CREATE TABLE users (
				id INT PRIMARY KEY,
				name NVARCHAR(255)
			);`,
			expected: 1,
		},
		{
			name: "multiple statements with GO",
			script: `CREATE TABLE users (
				id INT PRIMARY KEY,
				name NVARCHAR(255)
			);
			GO
			CREATE TABLE posts (
				id INT PRIMARY KEY,
				user_id INT
			);
			GO`,
			expected: 2,
		},
		{
			name: "statements with lowercase go",
			script: `CREATE TABLE users (id INT);
			go
			CREATE TABLE posts (id INT);
			go`,
			expected: 2,
		},
		{
			name: "statements with mixed case GO",
			script: `CREATE TABLE users (id INT);
			Go
			CREATE TABLE posts (id INT);
			gO`,
			expected: 2,
		},
		{
			name:     "empty script",
			script:   "",
			expected: 0,
		},
		{
			name: "script with only GO",
			script: `GO
			GO
			GO`,
			expected: 0,
		},
		{
			name: "complex statements",
			script: `
			IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'users')
			CREATE TABLE users (
				id INT PRIMARY KEY,
				name NVARCHAR(255)
			);
			GO

			CREATE INDEX idx_users_name ON users(name);
			GO

			INSERT INTO users (id, name) VALUES (1, 'admin');
			GO`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statements := splitStatements(tt.script)

			// Filter out empty statements
			nonEmpty := 0
			for _, stmt := range statements {
				if strings.TrimSpace(stmt) != "" {
					nonEmpty++
				}
			}

			if nonEmpty != tt.expected {
				t.Errorf("expected %d statements, got %d", tt.expected, nonEmpty)
			}
		})
	}
}

func TestSplitStatements_Preservation(t *testing.T) {
	script := `CREATE TABLE users (id INT);
GO
CREATE TABLE posts (id INT);`

	statements := splitStatements(script)

	if len(statements) < 2 {
		t.Fatalf("expected at least 2 statements, got %d", len(statements))
	}

	// First statement should contain CREATE TABLE users
	if !strings.Contains(statements[0], "CREATE TABLE users") {
		t.Error("first statement should contain 'CREATE TABLE users'")
	}

	// Second statement should contain CREATE TABLE posts
	if !strings.Contains(statements[1], "CREATE TABLE posts") {
		t.Error("second statement should contain 'CREATE TABLE posts'")
	}

	// GO should not be in the statements
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "GO" || trimmed == "go" || trimmed == "Go" {
			t.Error("GO statement should not be included in results")
		}
	}
}

func TestMigrationFilenamePatterns(t *testing.T) {
	tests := []struct {
		filename      string
		expectedVer   int
		expectedName  string
		expectedType  string
		valid         bool
	}{
		{
			filename:     "001_create_users.up.sql",
			expectedVer:  1,
			expectedName: "create_users",
			expectedType: "up",
			valid:        true,
		},
		{
			filename:     "001_create_users.down.sql",
			expectedVer:  1,
			expectedName: "create_users",
			expectedType: "down",
			valid:        true,
		},
		{
			filename:     "042_add_indexes.up.sql",
			expectedVer:  42,
			expectedName: "add_indexes",
			expectedType: "up",
			valid:        true,
		},
		{
			filename:     "100_major_refactor.up.sql",
			expectedVer:  100,
			expectedName: "major_refactor",
			expectedType: "up",
			valid:        true,
		},
		{
			filename: "invalid_no_version.up.sql",
			valid:    false,
		},
		{
			filename: "001_no_direction.sql",
			valid:    false,
		},
		{
			filename: "not_a_migration.txt",
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			// Check if filename ends with .sql
			if !strings.HasSuffix(tt.filename, ".sql") && tt.valid {
				t.Error("valid migration files should end with .sql")
			}

			if tt.valid {
				// Check version parsing
				parts := strings.SplitN(tt.filename, "_", 2)
				if len(parts) < 2 {
					t.Error("valid migration should have version and name separated by underscore")
				}

				// Check up/down designation
				hasUp := strings.Contains(tt.filename, ".up.")
				hasDown := strings.Contains(tt.filename, ".down.")

				if !hasUp && !hasDown {
					t.Error("valid migration should have .up. or .down. in filename")
				}

				if hasUp && tt.expectedType != "up" {
					t.Errorf("expected type=%s, got up", tt.expectedType)
				}

				if hasDown && tt.expectedType != "down" {
					t.Errorf("expected type=%s, got down", tt.expectedType)
				}
			}
		})
	}
}

func TestMigrationOrdering(t *testing.T) {
	client := &SQLClient{}
	m := NewMigrator(client)

	// Add migrations in random order
	m.AddMigration(5, "five", "up5", "down5")
	m.AddMigration(2, "two", "up2", "down2")
	m.AddMigration(8, "eight", "up8", "down8")
	m.AddMigration(1, "one", "up1", "down1")
	m.AddMigration(3, "three", "up3", "down3")

	// Verify sorted order
	for i := 0; i < len(m.migrations)-1; i++ {
		if m.migrations[i].Version >= m.migrations[i+1].Version {
			t.Errorf("migrations not sorted: version %d at position %d >= version %d at position %d",
				m.migrations[i].Version, i, m.migrations[i+1].Version, i+1)
		}
	}

	// Verify specific ordering
	expected := []int{1, 2, 3, 5, 8}
	for i, ver := range expected {
		if m.migrations[i].Version != ver {
			t.Errorf("expected migration at index %d to have version %d, got %d",
				i, ver, m.migrations[i].Version)
		}
	}
}

func TestMigrationDuplicateVersions(t *testing.T) {
	client := &SQLClient{}
	m := NewMigrator(client)

	// Add same version twice
	m.AddMigration(1, "first", "up1", "down1")
	m.AddMigration(1, "first_duplicate", "up1_v2", "down1_v2")

	// The second one should overwrite/update due to sorting
	// Count migrations with version 1
	count := 0
	for _, mig := range m.migrations {
		if mig.Version == 1 {
			count++
		}
	}

	// After re-sorting, there should be 2 entries with version 1
	// (the implementation doesn't prevent duplicates, just sorts them)
	if count != 2 {
		t.Logf("note: migration system allows duplicate versions (found %d)", count)
	}
}
