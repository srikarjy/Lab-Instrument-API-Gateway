package db

import (
	"context"
	"testing"
	"time"

	"github.com/yourorg/lab-gateway/pkg/config"
	"github.com/yourorg/lab-gateway/pkg/logger"
)

func TestConnectionManager(t *testing.T) {
	// Test configuration
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "lab_instruments_test",
		User:     "user",
		Password: "password",
		SSLMode:  "disable",
	}
	
	log := logger.NewDefaultLogger()
	
	t.Run("NewConnectionManager", func(t *testing.T) {
		// This test requires a running PostgreSQL instance
		// Skip if not available
		t.Skip("Requires running PostgreSQL instance")
		
		cm, err := NewConnectionManager(cfg, log)
		if err != nil {
			t.Fatalf("Failed to create connection manager: %v", err)
		}
		defer cm.Close()
		
		if cm.db == nil {
			t.Error("Database connection is nil")
		}
	})
	
	t.Run("HealthCheck", func(t *testing.T) {
		t.Skip("Requires running PostgreSQL instance")
		
		cm, err := NewConnectionManager(cfg, log)
		if err != nil {
			t.Fatalf("Failed to create connection manager: %v", err)
		}
		defer cm.Close()
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err = cm.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})
	
	t.Run("GetStats", func(t *testing.T) {
		t.Skip("Requires running PostgreSQL instance")
		
		cm, err := NewConnectionManager(cfg, log)
		if err != nil {
			t.Fatalf("Failed to create connection manager: %v", err)
		}
		defer cm.Close()
		
		stats := cm.GetStats()
		if stats.OpenConnections < 0 {
			t.Error("Invalid connection stats")
		}
	})
}