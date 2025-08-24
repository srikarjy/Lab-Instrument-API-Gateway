package repository

import (
	"context"
	"fmt"

	"github.com/yourorg/lab-gateway/pkg/db"
	"github.com/yourorg/lab-gateway/pkg/logger"
)

// repositoryManager implements RepositoryManager interface
type repositoryManager struct {
	db     *db.ConnectionManager
	logger *logger.Logger

	deviceRepo      DeviceRepository
	measurementRepo MeasurementRepository
	commandRepo     CommandRepository
	alertRepo       AlertRepository
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(db *db.ConnectionManager, logger *logger.Logger) RepositoryManager {
	return &repositoryManager{
		db:              db,
		logger:          logger,
		deviceRepo:      NewDeviceRepository(db, logger),
		measurementRepo: NewMeasurementRepository(db, logger),
		commandRepo:     NewCommandRepository(db, logger),
		alertRepo:       NewAlertRepository(db, logger),
	}
}

// Device returns the device repository
func (rm *repositoryManager) Device() DeviceRepository {
	return rm.deviceRepo
}

// Measurement returns the measurement repository
func (rm *repositoryManager) Measurement() MeasurementRepository {
	return rm.measurementRepo
}

// Command returns the command repository
func (rm *repositoryManager) Command() CommandRepository {
	return rm.commandRepo
}

// Alert returns the alert repository
func (rm *repositoryManager) Alert() AlertRepository {
	return rm.alertRepo
}

// WithTransaction executes a function within a database transaction
func (rm *repositoryManager) WithTransaction(ctx context.Context, fn func(ctx context.Context, repos RepositoryManager) error) error {
	tx, err := rm.db.BeginTx(ctx, nil)
	if err != nil {
		rm.logger.WithError(err).Error("Failed to begin transaction")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// For now, just use the same repository manager within the transaction
	// TODO: Implement proper transactional repositories
	if err := fn(ctx, rm); err != nil {
		rm.logger.WithError(err).Error("Transaction function failed")
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		rm.logger.WithError(err).Error("Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	rm.logger.Debug("Transaction committed successfully")
	return nil
}

// HealthCheck performs a health check on all repositories
func (rm *repositoryManager) HealthCheck(ctx context.Context) error {
	// Check database connectivity
	if err := rm.db.HealthCheck(ctx); err != nil {
		rm.logger.WithError(err).Error("Database health check failed")
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check database statistics
	stats := rm.db.GetStats()
	rm.logger.WithFields(map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":          stats.InUseConnections,
		"idle":            stats.IdleConnections,
	}).Debug("Database connection pool status")

	// Verify we can execute a simple query
	var result int
	err := rm.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		rm.logger.WithError(err).Error("Database query health check failed")
		return fmt.Errorf("database query health check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database query returned unexpected result: %d", result)
	}

	rm.logger.Debug("Repository manager health check passed")
	return nil
}

// Close closes all repository resources
func (rm *repositoryManager) Close() error {
	rm.logger.Info("Closing repository manager")
	
	if err := rm.db.Close(); err != nil {
		rm.logger.WithError(err).Error("Failed to close database connection")
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	rm.logger.Info("Repository manager closed successfully")
	return nil
}

// TODO: Implement transactional repositories when needed
// For now, transactions are handled at the individual repository level