package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/yourorg/lab-gateway/pkg/config"
	"github.com/yourorg/lab-gateway/pkg/db"
	"github.com/yourorg/lab-gateway/pkg/logger"
)

func main() {
	var (
		migrationsPath = flag.String("path", "./migrations", "Path to migration files")
		action         = flag.String("action", "up", "Migration action: up, down, status, validate")
		timeout        = flag.Duration("timeout", 30*time.Second, "Migration timeout")
	)
	flag.Parse()

	// Load configuration
	cfg := config.Load()
	
	// Initialize logger
	logger := logger.NewDefaultLogger()
	
	// Create connection manager
	cm, err := db.NewConnectionManager(&cfg.Database, logger)
	if err != nil {
		logger.Fatalf("Failed to create connection manager: %v", err)
	}
	defer cm.Close()
	
	// Wait for database to be ready
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	
	if err := cm.WaitForConnection(ctx, *timeout); err != nil {
		logger.Fatalf("Database not ready: %v", err)
	}
	
	// Create migration runner
	migrator := db.NewMigrationRunner(cm.GetDB(), *migrationsPath, logger)
	
	// Initialize migrations table
	if err := migrator.Initialize(ctx); err != nil {
		logger.Fatalf("Failed to initialize migrations: %v", err)
	}
	
	// Execute migration action
	switch *action {
	case "up":
		if err := migrator.Up(ctx); err != nil {
			logger.Fatalf("Migration up failed: %v", err)
		}
		logger.Info("Migrations completed successfully")
		
	case "status":
		status, err := migrator.GetStatus(ctx)
		if err != nil {
			logger.Fatalf("Failed to get migration status: %v", err)
		}
		
		fmt.Printf("Migration Status:\n")
		fmt.Printf("  Current Version: %d\n", status.CurrentVersion)
		fmt.Printf("  Applied: %d\n", status.AppliedCount)
		fmt.Printf("  Pending: %d\n", status.PendingCount)
		fmt.Printf("\nMigrations:\n")
		
		for _, migration := range status.Migrations {
			status := "PENDING"
			if migration.AppliedAt != nil {
				status = fmt.Sprintf("APPLIED (%s)", migration.AppliedAt.Format("2006-01-02 15:04:05"))
			}
			fmt.Printf("  %03d_%s: %s\n", migration.Version, migration.Name, status)
		}
		
	case "validate":
		if err := migrator.ValidateIntegrity(ctx); err != nil {
			logger.Fatalf("Migration validation failed: %v", err)
		}
		logger.Info("Migration integrity validation passed")
		
	default:
		logger.Fatalf("Unknown action: %s", *action)
	}
}