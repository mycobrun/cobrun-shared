// Package database provides database client utilities.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/microsoft/go-mssqldb" // SQL Server driver
	_ "github.com/microsoft/go-mssqldb/azuread" // Azure AD auth
)

// SQLConfig holds Azure SQL configuration.
type SQLConfig struct {
	Host         string
	Port         int
	Database     string
	User         string
	Password     string
	UseMSI       bool // Use Managed Service Identity
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// DefaultSQLConfig returns sensible defaults.
func DefaultSQLConfig() SQLConfig {
	return SQLConfig{
		Port:         1433,
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		MaxLifetime:  5 * time.Minute,
	}
}

// SQLClient wraps a SQL database connection.
type SQLClient struct {
	db     *sql.DB
	config SQLConfig
}

// NewSQLClient creates a new Azure SQL client.
func NewSQLClient(ctx context.Context, config SQLConfig) (*SQLClient, error) {
	var connStr string

	if config.UseMSI {
		// Use Azure AD authentication with Managed Identity
		connStr = fmt.Sprintf(
			"sqlserver://%s:%d?database=%s&fedauth=ActiveDirectoryMSI",
			config.Host, config.Port, config.Database,
		)
	} else {
		// Use SQL authentication
		connStr = fmt.Sprintf(
			"sqlserver://%s:%s@%s:%d?database=%s&encrypt=true&trustservercertificate=false",
			config.User, config.Password, config.Host, config.Port, config.Database,
		)
	}

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.MaxLifetime)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLClient{
		db:     db,
		config: config,
	}, nil
}

// DB returns the underlying sql.DB instance.
func (c *SQLClient) DB() *sql.DB {
	return c.db
}

// Ping checks the database connection.
func (c *SQLClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Close closes the database connection.
func (c *SQLClient) Close() error {
	return c.db.Close()
}

// Exec executes a query without returning results.
func (c *SQLClient) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

// Query executes a query and returns rows.
func (c *SQLClient) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query and returns a single row.
func (c *SQLClient) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// Transaction represents a database transaction.
type Transaction struct {
	tx *sql.Tx
}

// Begin starts a new transaction.
func (c *SQLClient) Begin(ctx context.Context) (*Transaction, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Transaction{tx: tx}, nil
}

// Commit commits the transaction.
func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *Transaction) Rollback() error {
	return t.tx.Rollback()
}

// Exec executes a query in the transaction.
func (t *Transaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// Query executes a query in the transaction.
func (t *Transaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// QueryRow executes a query in the transaction and returns a single row.
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

// WithTransaction executes a function within a transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, it's committed.
func (c *SQLClient) WithTransaction(ctx context.Context, fn func(*Transaction) error) error {
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

// Paginate helps build paginated queries.
type Paginate struct {
	Page     int
	PageSize int
}

// Offset returns the SQL offset.
func (p Paginate) Offset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	return (p.Page - 1) * p.PageSize
}

// Limit returns the SQL limit.
func (p Paginate) Limit() int {
	if p.PageSize < 1 {
		return 10 // Default
	}
	if p.PageSize > 100 {
		return 100 // Max
	}
	return p.PageSize
}

// Stats returns database statistics.
func (c *SQLClient) Stats() sql.DBStats {
	return c.db.Stats()
}

// NullString creates a sql.NullString.
func NullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// NullInt64 creates a sql.NullInt64.
func NullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: true}
}

// NullFloat64 creates a sql.NullFloat64.
func NullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: f, Valid: true}
}

// NullTime creates a sql.NullTime.
func NullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// NullBool creates a sql.NullBool.
func NullBool(b bool) sql.NullBool {
	return sql.NullBool{Bool: b, Valid: true}
}

// Retry-enabled operations for production resilience

// ExecWithRetry executes a query with retry logic.
func (c *SQLClient) ExecWithRetry(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	err := RetrySQLOperation(ctx, func() error {
		var execErr error
		result, execErr = c.db.ExecContext(ctx, query, args...)
		return execErr
	})
	return result, err
}

// QueryWithRetry executes a query with retry logic and returns rows.
func (c *SQLClient) QueryWithRetry(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	err := RetrySQLOperation(ctx, func() error {
		var queryErr error
		rows, queryErr = c.db.QueryContext(ctx, query, args...)
		return queryErr
	})
	return rows, err
}

// QueryRowWithRetry executes a query with retry and returns a scanner.
// Note: This returns a custom wrapper since sql.Row cannot be retried after creation.
func (c *SQLClient) QueryRowWithRetry(ctx context.Context, query string, args ...interface{}) *RetryableRow {
	return &RetryableRow{
		client: c,
		ctx:    ctx,
		query:  query,
		args:   args,
	}
}

// RetryableRow wraps a row query with retry capability.
type RetryableRow struct {
	client *SQLClient
	ctx    context.Context
	query  string
	args   []interface{}
}

// Scan executes the query with retry and scans the result.
func (r *RetryableRow) Scan(dest ...interface{}) error {
	return RetrySQLOperation(r.ctx, func() error {
		row := r.client.db.QueryRowContext(r.ctx, r.query, r.args...)
		return row.Scan(dest...)
	})
}

// PingWithRetry checks the database connection with retry logic.
func (c *SQLClient) PingWithRetry(ctx context.Context) error {
	return RetrySQLOperation(ctx, func() error {
		return c.db.PingContext(ctx)
	})
}

// WithTransactionRetry executes a function within a transaction with retry logic.
// The entire transaction is retried on transient failures.
func (c *SQLClient) WithTransactionRetry(ctx context.Context, fn func(*Transaction) error) error {
	return RetrySQLOperation(ctx, func() error {
		return c.WithTransaction(ctx, fn)
	})
}

// PrepareWithRetry prepares a statement with retry logic.
func (c *SQLClient) PrepareWithRetry(ctx context.Context, query string) (*sql.Stmt, error) {
	var stmt *sql.Stmt
	err := RetrySQLOperation(ctx, func() error {
		var prepErr error
		stmt, prepErr = c.db.PrepareContext(ctx, query)
		return prepErr
	})
	return stmt, err
}
