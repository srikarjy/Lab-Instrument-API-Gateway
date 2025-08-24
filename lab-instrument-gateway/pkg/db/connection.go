package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/yourorg/lab-gateway/pkg/config"
	"github.com/yourorg/lab-gateway/pkg/logger"
)

// ConnectionManager manages database connections with pooling and health checks
type ConnectionManager struct {
	db     *sql.DB
	config *config.DatabaseConfig
	logger *logger.Logger
}

// ConnectionStats holds connection pool statistics
type ConnectionStats struct {
	OpenConnections     int
	InUseConnections    int
	IdleConnections     int
	WaitCount           int64
	WaitDuration        time.Duration
	MaxIdleClosed       int64
	MaxIdleTimeClosed   int64
	MaxLifetimeClosed   int64
}

// NewConnectionManager creates a new database connection manager
func NewConnectionManager(cfg *config.DatabaseConfig, log *logger.Logger) (*ConnectionManager, error) {
	cm := &ConnectionManager{
		config: cfg,
		logger: log,
	}

	if err := cm.connect(); err != nil {
		return nil, fmt.Errorf("failed to establish database connection: %w", err)
	}

	return cm, nil
}

// connect establishes the database connection with retry logic
func (cm *ConnectionManager) connect() error {
	dsn := cm.buildDSN()
	
	var db *sql.DB
	var err error
	
	// Retry connection with exponential backoff
	maxRetries := 5
	baseDelay := time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			cm.logger.WithError(err).Errorf("Failed to open database connection (attempt %d/%d)", attempt+1, maxRetries)
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
				cm.logger.Infof("Retrying database connection in %v", delay)
				time.Sleep(delay)
				continue
			}
			return fmt.Errorf("failed to open database after %d attempts: %w", maxRetries, err)
		}

		// Test the connection
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = db.PingContext(ctx)
		cancel()
		
		if err != nil {
			db.Close()
			cm.logger.WithError(err).Errorf("Failed to ping database (attempt %d/%d)", attempt+1, maxRetries)
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<attempt)
				cm.logger.Infof("Retrying database connection in %v", delay)
				time.Sleep(delay)
				continue
			}
			return fmt.Errorf("failed to ping database after %d attempts: %w", maxRetries, err)
		}

		break
	}

	// Configure connection pool
	cm.configureConnectionPool(db)
	
	cm.db = db
	cm.logger.Info("Database connection established successfully")
	
	return nil
}

// buildDSN constructs the database connection string
func (cm *ConnectionManager) buildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cm.config.Host,
		cm.config.Port,
		cm.config.User,
		cm.config.Password,
		cm.config.Name,
		cm.config.SSLMode,
	)
}

// configureConnectionPool sets up connection pool parameters for high concurrency
func (cm *ConnectionManager) configureConnectionPool(db *sql.DB) {
	// Set maximum number of open connections (for 1000+ concurrent connections)
	db.SetMaxOpenConns(100)
	
	// Set maximum number of idle connections
	db.SetMaxIdleConns(25)
	
	// Set maximum lifetime of connections
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Set maximum idle time for connections
	db.SetConnMaxIdleTime(1 * time.Minute)
	
	cm.logger.Info("Database connection pool configured for high concurrency")
}

// GetDB returns the database connection
func (cm *ConnectionManager) GetDB() *sql.DB {
	return cm.db
}

// HealthCheck performs a database health check
func (cm *ConnectionManager) HealthCheck(ctx context.Context) error {
	if cm.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Ping with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := cm.db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test with a simple query
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	var result int
	err := cm.db.QueryRowContext(queryCtx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database query returned unexpected result: %d", result)
	}

	return nil
}

// GetStats returns connection pool statistics
func (cm *ConnectionManager) GetStats() ConnectionStats {
	if cm.db == nil {
		return ConnectionStats{}
	}

	stats := cm.db.Stats()
	return ConnectionStats{
		OpenConnections:     stats.OpenConnections,
		InUseConnections:    stats.InUse,
		IdleConnections:     stats.Idle,
		WaitCount:           stats.WaitCount,
		WaitDuration:        stats.WaitDuration,
		MaxIdleClosed:       stats.MaxIdleClosed,
		MaxIdleTimeClosed:   stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:   stats.MaxLifetimeClosed,
	}
}

// BeginTx starts a new transaction with context
func (cm *ConnectionManager) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if cm.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	
	tx, err := cm.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	return tx, nil
}

// ExecContext executes a query with context
func (cm *ConnectionManager) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if cm.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	
	result, err := cm.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	
	return result, nil
}

// QueryContext executes a query that returns rows with context
func (cm *ConnectionManager) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if cm.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	
	rows, err := cm.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	
	return rows, nil
}

// QueryRowContext executes a query that returns a single row with context
func (cm *ConnectionManager) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if cm.db == nil {
		// Return a row that will return an error when scanned
		return &sql.Row{}
	}
	
	return cm.db.QueryRowContext(ctx, query, args...)
}

// PrepareContext prepares a statement with context
func (cm *ConnectionManager) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if cm.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	
	stmt, err := cm.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	
	return stmt, nil
}

// Close gracefully closes the database connection
func (cm *ConnectionManager) Close() error {
	if cm.db == nil {
		return nil
	}

	cm.logger.Info("Closing database connection...")
	
	// Close the database connection
	if err := cm.db.Close(); err != nil {
		cm.logger.WithError(err).Error("Error closing database connection")
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	cm.logger.Info("Database connection closed successfully")
	return nil
}

// WaitForConnection waits for database to become available
func (cm *ConnectionManager) WaitForConnection(ctx context.Context, maxWait time.Duration) error {
	cm.logger.Info("Waiting for database connection...")
	
	timeout := time.After(maxWait)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for database connection after %v", maxWait)
		case <-ticker.C:
			if err := cm.HealthCheck(ctx); err == nil {
				cm.logger.Info("Database connection is ready")
				return nil
			}
			cm.logger.Debug("Database not ready yet, retrying...")
		}
	}
}

// IsConnected checks if the database connection is active
func (cm *ConnectionManager) IsConnected() bool {
	if cm.db == nil {
		return false
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	return cm.HealthCheck(ctx) == nil
}