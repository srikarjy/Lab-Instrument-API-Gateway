package db

import (
	"context"
	"fmt"
)

// getAppliedMigrations retrieves applied migrations from the database
func (mr *MigrationRunner) getAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `
		SELECT version, name, checksum, applied_at 
		FROM schema_migrations 
		ORDER BY version
	`
	
	rows, err := mr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()
	
	var migrations []Migration
	for rows.Next() {
		var migration Migration
		err := rows.Scan(
			&migration.Version,
			&migration.Name,
			&migration.Checksum,
			&migration.AppliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		migrations = append(migrations, migration)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}
	
	return migrations, nil
}

// Up runs all pending migrations
func (mr *MigrationRunner) Up(ctx context.Context) error {
	status, err := mr.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}
	
	if status.PendingCount == 0 {
		mr.logger.Info("No pending migrations to apply")
		return nil
	}
	
	mr.logger.Infof("Applying %d pending migrations", status.PendingCount)
	
	for _, migration := range status.Migrations {
		if migration.AppliedAt != nil {
			continue // Skip already applied migrations
		}
		
		if err := mr.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}
	
	mr.logger.Info("All migrations applied successfully")
	return nil
}

// applyMigration applies a single migration within a transaction
func (mr *MigrationRunner) applyMigration(ctx context.Context, migration Migration) error {
	mr.logger.Infof("Applying migration %d: %s", migration.Version, migration.Name)
	
	// Start transaction
	tx, err := mr.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Execute migration SQL
	_, err = tx.ExecContext(ctx, migration.UpSQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}
	
	// Record migration in schema_migrations table
	_, err = tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (version, name, checksum, applied_at)
		VALUES ($1, $2, $3, NOW())
	`, migration.Version, migration.Name, migration.Checksum)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}
	
	mr.logger.Infof("Migration %d applied successfully", migration.Version)
	return nil
}

// ValidateIntegrity validates that applied migrations match their checksums
func (mr *MigrationRunner) ValidateIntegrity(ctx context.Context) error {
	mr.logger.Info("Validating migration integrity...")
	
	// Load file migrations
	fileMigrations, err := mr.LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load file migrations: %w", err)
	}
	
	// Get applied migrations
	appliedMigrations, err := mr.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	// Create maps for comparison
	fileMap := make(map[int]Migration)
	for _, m := range fileMigrations {
		fileMap[m.Version] = m
	}
	
	// Validate each applied migration
	for _, applied := range appliedMigrations {
		fileMigration, exists := fileMap[applied.Version]
		if !exists {
			return fmt.Errorf("applied migration %d not found in migration files", applied.Version)
		}
		
		if fileMigration.Checksum != applied.Checksum {
			return fmt.Errorf("checksum mismatch for migration %d: file=%s, db=%s", 
				applied.Version, fileMigration.Checksum, applied.Checksum)
		}
	}
	
	mr.logger.Info("Migration integrity validation passed")
	return nil
}