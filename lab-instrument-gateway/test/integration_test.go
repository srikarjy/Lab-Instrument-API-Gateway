package test

import (
	"context"
	"testing"
	"time"

	"github.com/yourorg/lab-gateway/pkg/config"
	"github.com/yourorg/lab-gateway/pkg/db"
	"github.com/yourorg/lab-gateway/pkg/logger"
	"github.com/yourorg/lab-gateway/pkg/models"
)

// TestDatabaseIntegration tests the complete database setup
func TestDatabaseIntegration(t *testing.T) {
	// Skip if no database available
	t.Skip("Integration test requires running PostgreSQL database")
	
	// Test configuration
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "lab_instruments",
		User:     "user",
		Password: "password",
		SSLMode:  "disable",
	}
	
	log := logger.NewDefaultLogger()
	
	// Create connection manager
	cm, err := db.NewConnectionManager(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer cm.Close()
	
	// Test health check
	ctx := context.Background()
	if err := cm.HealthCheck(ctx); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	
	// Test migration runner
	migrator := db.NewMigrationRunner(cm.GetDB(), "../migrations", log)
	
	// Initialize migrations
	if err := migrator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize migrations: %v", err)
	}
	
	// Get migration status
	status, err := migrator.GetStatus(ctx)
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}
	
	t.Logf("Migration status: Current=%d, Applied=%d, Pending=%d", 
		status.CurrentVersion, status.AppliedCount, status.PendingCount)
	
	// Test model validation
	device := &models.Device{
		ID:           "test-device-001",
		Name:         "Test Device",
		Type:         "test",
		Version:      "1.0.0",
		Status:       models.DeviceStatusOffline,
		Capabilities: []string{"measurement", "control"},
	}
	
	if err := device.Validate(); err != nil {
		t.Errorf("Device validation failed: %v", err)
	}
	
	measurement := &models.Measurement{
		DeviceID:  "test-device-001",
		Type:      "temperature",
		Value:     25.5,
		Unit:      "celsius",
		Quality:   models.QualityGood,
		Timestamp: time.Now(),
	}
	
	if err := measurement.Validate(); err != nil {
		t.Errorf("Measurement validation failed: %v", err)
	}
	
	command := &models.Command{
		DeviceID:       "test-device-001",
		CommandID:      "cmd-001",
		Type:           "calibrate",
		TimeoutSeconds: 30,
		Parameters:     map[string]interface{}{"mode": "auto"},
	}
	command.SetDefaults()
	
	if err := command.Validate(); err != nil {
		t.Errorf("Command validation failed: %v", err)
	}
	
	alert := &models.Alert{
		DeviceID: &device.ID,
		Type:     models.AlertTypeDeviceOffline,
		Severity: models.AlertSeverityWarning,
		Message:  "Device went offline",
	}
	alert.SetDefaults()
	
	if err := alert.Validate(); err != nil {
		t.Errorf("Alert validation failed: %v", err)
	}
	
	t.Log("All database integration tests passed")
}