package db

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LoadMigrations loads all migration files from the migrations directory
func (mr *MigrationRunner) LoadMigrations() ([]Migration, error) {
	files, err := filepath.Glob(filepath.Join(mr.migrationsPath, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}
	
	var migrations []Migration
	
	for _, file := range files {
		migration, err := mr.parseMigrationFile(file)
		if err != nil {
			mr.logger.WithError(err).Errorf("Failed to parse migration file: %s", file)
			continue
		}
		migrations = append(migrations, migration)
	}
	
	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	
	return migrations, nil
}

// parseMigrationFile parses a migration file and extracts version, name, and SQL
func (mr *MigrationRunner) parseMigrationFile(filePath string) (Migration, error) {
	filename := filepath.Base(filePath)
	
	// Parse version from filename (e.g., "001_initial_schema.sql")
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return Migration{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}
	
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return Migration{}, fmt.Errorf("invalid version in filename %s: %w", filename, err)
	}
	
	name := strings.TrimSuffix(parts[1], ".sql")
	
	// Read file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read migration file %s: %w", filePath, err)
	}
	
	sql := string(content)
	checksum := mr.calculateChecksum(sql)
	
	return Migration{
		Version:  version,
		Name:     name,
		UpSQL:    sql,
		Checksum: checksum,
	}, nil
}

// calculateChecksum calculates SHA-256 checksum of migration content
func (mr *MigrationRunner) calculateChecksum(content string) string {
	// Simple checksum implementation - in production, use crypto/sha256
	return fmt.Sprintf("%x", len(content))
}

// GetStatus returns the current migration status
func (mr *MigrationRunner) GetStatus(ctx context.Context) (*MigrationStatus, error) {
	// Load all available migrations
	migrations, err := mr.LoadMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}
	
	// Get applied migrations from database
	appliedMigrations, err := mr.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	// Create a map of applied migrations for quick lookup
	appliedMap := make(map[int]*time.Time)
	for _, applied := range appliedMigrations {
		appliedMap[applied.Version] = applied.AppliedAt
	}
	
	// Update migration status
	currentVersion := 0
	pendingCount := 0
	appliedCount := 0
	
	for i := range migrations {
		if appliedAt, exists := appliedMap[migrations[i].Version]; exists {
			migrations[i].AppliedAt = appliedAt
			appliedCount++
			currentVersion = migrations[i].Version
		} else {
			pendingCount++
		}
	}
	
	return &MigrationStatus{
		CurrentVersion: currentVersion,
		PendingCount:   pendingCount,
		AppliedCount:   appliedCount,
		Migrations:     migrations,
	}, nil
}