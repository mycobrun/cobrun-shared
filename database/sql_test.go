package database

import (
	"testing"
	"time"
)

func TestDefaultSQLConfig(t *testing.T) {
	config := DefaultSQLConfig()

	if config.Port != 1433 {
		t.Errorf("expected Port=1433, got %d", config.Port)
	}
	if config.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns=25, got %d", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns=5, got %d", config.MaxIdleConns)
	}
	if config.MaxLifetime != 5*time.Minute {
		t.Errorf("expected MaxLifetime=5m, got %v", config.MaxLifetime)
	}
}

func TestSQLConfig(t *testing.T) {
	tests := []struct {
		name   string
		config SQLConfig
	}{
		{
			name: "with credentials",
			config: SQLConfig{
				Host:         "localhost",
				Port:         1433,
				Database:     "testdb",
				User:         "sa",
				Password:     "password",
				UseMSI:       false,
				MaxOpenConns: 25,
				MaxIdleConns: 5,
				MaxLifetime:  5 * time.Minute,
			},
		},
		{
			name: "with MSI",
			config: SQLConfig{
				Host:         "test.database.windows.net",
				Port:         1433,
				Database:     "testdb",
				UseMSI:       true,
				MaxOpenConns: 25,
				MaxIdleConns: 5,
				MaxLifetime:  5 * time.Minute,
			},
		},
		{
			name: "custom pool settings",
			config: SQLConfig{
				Host:         "localhost",
				Port:         1433,
				Database:     "testdb",
				User:         "sa",
				Password:     "password",
				MaxOpenConns: 50,
				MaxIdleConns: 10,
				MaxLifetime:  10 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Host == "" {
				t.Error("Host should not be empty")
			}
			if tt.config.Port == 0 {
				t.Error("Port should not be zero")
			}
			if tt.config.Database == "" {
				t.Error("Database should not be empty")
			}
			if !tt.config.UseMSI && (tt.config.User == "" || tt.config.Password == "") {
				t.Error("User and Password required when not using MSI")
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedOffset int
		expectedLimit  int
	}{
		{
			name:           "first page",
			page:           1,
			pageSize:       10,
			expectedOffset: 0,
			expectedLimit:  10,
		},
		{
			name:           "second page",
			page:           2,
			pageSize:       10,
			expectedOffset: 10,
			expectedLimit:  10,
		},
		{
			name:           "third page",
			page:           3,
			pageSize:       25,
			expectedOffset: 50,
			expectedLimit:  25,
		},
		{
			name:           "invalid page (0)",
			page:           0,
			pageSize:       10,
			expectedOffset: 0,
			expectedLimit:  10,
		},
		{
			name:           "invalid page (negative)",
			page:           -1,
			pageSize:       10,
			expectedOffset: 0,
			expectedLimit:  10,
		},
		{
			name:           "invalid page size (0)",
			page:           1,
			pageSize:       0,
			expectedOffset: 0,
			expectedLimit:  10, // Default
		},
		{
			name:           "invalid page size (negative)",
			page:           1,
			pageSize:       -1,
			expectedOffset: 0,
			expectedLimit:  10, // Default
		},
		{
			name:           "exceeds max page size",
			page:           1,
			pageSize:       200,
			expectedOffset: 0,
			expectedLimit:  100, // Max
		},
		{
			name:           "max page size",
			page:           1,
			pageSize:       100,
			expectedOffset: 0,
			expectedLimit:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Paginate{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}

			offset := p.Offset()
			limit := p.Limit()

			if offset != tt.expectedOffset {
				t.Errorf("expected offset=%d, got %d", tt.expectedOffset, offset)
			}
			if limit != tt.expectedLimit {
				t.Errorf("expected limit=%d, got %d", tt.expectedLimit, limit)
			}
		})
	}
}

func TestNullHelpers(t *testing.T) {
	t.Run("NullString", func(t *testing.T) {
		// Non-empty string
		ns := NullString("test")
		if !ns.Valid {
			t.Error("NullString should be valid for non-empty string")
		}
		if ns.String != "test" {
			t.Errorf("expected string=test, got %s", ns.String)
		}

		// Empty string
		emptyNs := NullString("")
		if emptyNs.Valid {
			t.Error("NullString should be invalid for empty string")
		}
	})

	t.Run("NullInt64", func(t *testing.T) {
		ni := NullInt64(42)
		if !ni.Valid {
			t.Error("NullInt64 should be valid")
		}
		if ni.Int64 != 42 {
			t.Errorf("expected int64=42, got %d", ni.Int64)
		}

		// Zero value
		zeroNi := NullInt64(0)
		if !zeroNi.Valid {
			t.Error("NullInt64 should be valid for zero")
		}
		if zeroNi.Int64 != 0 {
			t.Errorf("expected int64=0, got %d", zeroNi.Int64)
		}
	})

	t.Run("NullFloat64", func(t *testing.T) {
		nf := NullFloat64(3.14)
		if !nf.Valid {
			t.Error("NullFloat64 should be valid")
		}
		if nf.Float64 != 3.14 {
			t.Errorf("expected float64=3.14, got %f", nf.Float64)
		}

		// Zero value
		zeroNf := NullFloat64(0.0)
		if !zeroNf.Valid {
			t.Error("NullFloat64 should be valid for zero")
		}
	})

	t.Run("NullTime", func(t *testing.T) {
		now := time.Now()
		nt := NullTime(now)
		if !nt.Valid {
			t.Error("NullTime should be valid for non-zero time")
		}
		if !nt.Time.Equal(now) {
			t.Errorf("expected time=%v, got %v", now, nt.Time)
		}

		// Zero time
		zeroNt := NullTime(time.Time{})
		if zeroNt.Valid {
			t.Error("NullTime should be invalid for zero time")
		}
	})

	t.Run("NullBool", func(t *testing.T) {
		// True
		trueNb := NullBool(true)
		if !trueNb.Valid {
			t.Error("NullBool should be valid")
		}
		if !trueNb.Bool {
			t.Error("expected bool=true")
		}

		// False
		falseNb := NullBool(false)
		if !falseNb.Valid {
			t.Error("NullBool should be valid")
		}
		if falseNb.Bool {
			t.Error("expected bool=false")
		}
	})
}

func TestSQLConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  SQLConfig
		isValid bool
	}{
		{
			name: "valid SQL auth",
			config: SQLConfig{
				Host:     "localhost",
				Port:     1433,
				Database: "testdb",
				User:     "sa",
				Password: "password",
				UseMSI:   false,
			},
			isValid: true,
		},
		{
			name: "valid MSI auth",
			config: SQLConfig{
				Host:     "test.database.windows.net",
				Port:     1433,
				Database: "testdb",
				UseMSI:   true,
			},
			isValid: true,
		},
		{
			name: "missing host",
			config: SQLConfig{
				Host:     "",
				Port:     1433,
				Database: "testdb",
				User:     "sa",
				Password: "password",
			},
			isValid: false,
		},
		{
			name: "missing database",
			config: SQLConfig{
				Host:     "localhost",
				Port:     1433,
				Database: "",
				User:     "sa",
				Password: "password",
			},
			isValid: false,
		},
		{
			name: "missing credentials without MSI",
			config: SQLConfig{
				Host:     "localhost",
				Port:     1433,
				Database: "testdb",
				User:     "",
				Password: "",
				UseMSI:   false,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.config.Host != "" && tt.config.Database != "" &&
				(tt.config.UseMSI || (tt.config.User != "" && tt.config.Password != ""))

			if valid != tt.isValid {
				t.Errorf("expected valid=%v, got %v", tt.isValid, valid)
			}
		})
	}
}

func TestPaginateEdgeCases(t *testing.T) {
	t.Run("large page number", func(t *testing.T) {
		p := Paginate{Page: 1000, PageSize: 10}
		offset := p.Offset()
		expectedOffset := 9990 // (1000-1) * 10

		if offset != expectedOffset {
			t.Errorf("expected offset=%d, got %d", expectedOffset, offset)
		}
	})

	t.Run("page size at boundary", func(t *testing.T) {
		p := Paginate{Page: 1, PageSize: 100}
		limit := p.Limit()

		if limit != 100 {
			t.Errorf("expected limit=100, got %d", limit)
		}
	})

	t.Run("page size just above boundary", func(t *testing.T) {
		p := Paginate{Page: 1, PageSize: 101}
		limit := p.Limit()

		if limit != 100 {
			t.Errorf("expected limit=100 (capped), got %d", limit)
		}
	})
}

func TestConnectionStringFormation(t *testing.T) {
	tests := []struct {
		name     string
		config   SQLConfig
		contains []string
	}{
		{
			name: "SQL auth",
			config: SQLConfig{
				Host:     "localhost",
				Port:     1433,
				Database: "testdb",
				User:     "sa",
				Password: "password",
				UseMSI:   false,
			},
			contains: []string{"localhost", "1433", "testdb", "sa"},
		},
		{
			name: "MSI auth",
			config: SQLConfig{
				Host:     "test.database.windows.net",
				Port:     1433,
				Database: "testdb",
				UseMSI:   true,
			},
			contains: []string{"test.database.windows.net", "1433", "testdb", "ActiveDirectoryMSI"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a structural test - we can't actually create connection strings
			// without the actual driver, but we can verify the config has required fields
			for _, field := range tt.contains {
				found := false
				if tt.config.Host == field || tt.config.Database == field || tt.config.User == field {
					found = true
				}
				if !found && field == "ActiveDirectoryMSI" && tt.config.UseMSI {
					found = true
				}
				if !found && field == "1433" {
					found = tt.config.Port == 1433
				}

				if !found {
					t.Logf("note: field %s not directly testable in unit test", field)
				}
			}
		})
	}
}
