package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourorg/lab-gateway/pkg/logger"
)

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db             *sql.DB
	logger         *logger.Logger
	migrationsPath string
}

// Migration represents a database migration
type Migration struct {
	Version   int
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
	Checksum  string
}

// MigrationStatus represents the status of migrations
type MigrationStatus struct {
	CurrentVersion int
	PendingCount   int
	AppliedCount   int
	Migrations     []Migration
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *sql.DB, migrationsPath string, log *logger.Logger) *MigrationRunner {
	return &MigrationRunner{
		db:             db,
		logger:         log,
		migrationsPath: migrationsPath,
	}
}

// Initialize creates the migrations table if it doesn't exist
func (mr *MigrationRunner) Initialize(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			checksum VARCHAR(64) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at 
		ON schema_migrations(applied_at);
	`
	
	_, err := mr.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations table: %w", err)
	}
	
	mr.logger.Info("Migrations table initialized")
	return nil
}