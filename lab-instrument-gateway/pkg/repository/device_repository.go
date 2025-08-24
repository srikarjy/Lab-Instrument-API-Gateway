package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/yourorg/lab-gateway/pkg/db"
	"github.com/yourorg/lab-gateway/pkg/logger"
	"github.com/yourorg/lab-gateway/pkg/models"
)

// deviceRepository implements DeviceRepository interface
type deviceRepository struct {
	db     *db.ConnectionManager
	logger *logger.Logger
}

// NewDeviceRepository creates a new device repository
func NewDeviceRepository(db *db.ConnectionManager, logger *logger.Logger) DeviceRepository {
	return &deviceRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new device
func (r *deviceRepository) Create(ctx context.Context, device *models.Device) error {
	if err := device.Validate(); err != nil {
		return fmt.Errorf("device validation failed: %w", err)
	}

	query := `
		INSERT INTO devices (id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	now := time.Now()
	if device.RegisteredAt.IsZero() {
		device.RegisteredAt = now
	}
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	device.UpdatedAt = now

	metadataJSON, err := marshalJSON(device.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		device.ID,
		device.Name,
		device.Type,
		device.Version,
		device.Status,
		metadataJSON,
		pq.Array(device.Capabilities),
		device.LastSeen,
		device.RegisteredAt,
		device.CreatedAt,
		device.UpdatedAt,
	)

	if err != nil {
		r.logger.WithField("device_id", device.ID).WithError(err).Error("Failed to create device")
		return fmt.Errorf("failed to create device: %w", err)
	}

	r.logger.WithField("device_id", device.ID).Info("Device created successfully")
	return nil
}

// GetByID retrieves a device by ID
func (r *deviceRepository) GetByID(ctx context.Context, id string) (*models.Device, error) {
	query := `
		SELECT id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at
		FROM devices
		WHERE id = $1
	`

	device := &models.Device{}
	var metadataJSON []byte
	var capabilities pq.StringArray

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&device.ID,
		&device.Name,
		&device.Type,
		&device.Version,
		&device.Status,
		&metadataJSON,
		&capabilities,
		&device.LastSeen,
		&device.RegisteredAt,
		&device.CreatedAt,
		&device.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("device not found: %s", id)
		}
		r.logger.WithField("device_id", id).WithError(err).Error("Failed to get device")
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	device.Capabilities = []string(capabilities)

	if err := unmarshalJSON(metadataJSON, &device.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return device, nil
}

// Update updates an existing device
func (r *deviceRepository) Update(ctx context.Context, device *models.Device) error {
	if err := device.Validate(); err != nil {
		return fmt.Errorf("device validation failed: %w", err)
	}

	query := `
		UPDATE devices 
		SET name = $2, type = $3, version = $4, status = $5, metadata = $6, capabilities = $7, last_seen = $8, updated_at = $9
		WHERE id = $1
	`

	device.UpdatedAt = time.Now()

	metadataJSON, err := marshalJSON(device.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		device.ID,
		device.Name,
		device.Type,
		device.Version,
		device.Status,
		metadataJSON,
		pq.Array(device.Capabilities),
		device.LastSeen,
		device.UpdatedAt,
	)

	if err != nil {
		r.logger.WithField("device_id", device.ID).WithError(err).Error("Failed to update device")
		return fmt.Errorf("failed to update device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device not found: %s", device.ID)
	}

	r.logger.WithField("device_id", device.ID).Info("Device updated successfully")
	return nil
}

// Delete removes a device
func (r *deviceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithField("device_id", id).WithError(err).Error("Failed to delete device")
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device not found: %s", id)
	}

	r.logger.WithField("device_id", id).Info("Device deleted successfully")
	return nil
}

// CreateBulk creates multiple devices in a single transaction
func (r *deviceRepository) CreateBulk(ctx context.Context, devices []*models.Device) (*BulkResult, error) {
	if len(devices) == 0 {
		return &BulkResult{}, nil
	}

	result := &BulkResult{}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO devices (id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, device := range devices {
		if err := device.Validate(); err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s validation failed: %w", device.ID, err))
			continue
		}

		if device.RegisteredAt.IsZero() {
			device.RegisteredAt = now
		}
		if device.CreatedAt.IsZero() {
			device.CreatedAt = now
		}
		device.UpdatedAt = now

		metadataJSON, err := marshalJSON(device.Metadata)
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s metadata marshal failed: %w", device.ID, err))
			continue
		}

		_, err = stmt.ExecContext(ctx,
			device.ID,
			device.Name,
			device.Type,
			device.Version,
			device.Status,
			metadataJSON,
			pq.Array(device.Capabilities),
			device.LastSeen,
			device.RegisteredAt,
			device.CreatedAt,
			device.UpdatedAt,
		)

		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s insert failed: %w", device.ID, err))
			continue
		}

		result.SuccessCount++
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"success_count": result.SuccessCount,
		"failure_count": result.FailureCount,
	}).Info("Bulk device creation completed")

	return result, nil
}

// UpdateBulk updates multiple devices in a single transaction
func (r *deviceRepository) UpdateBulk(ctx context.Context, devices []*models.Device) (*BulkResult, error) {
	if len(devices) == 0 {
		return &BulkResult{}, nil
	}

	result := &BulkResult{}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE devices 
		SET name = $2, type = $3, version = $4, status = $5, metadata = $6, capabilities = $7, last_seen = $8, updated_at = $9
		WHERE id = $1
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, device := range devices {
		if err := device.Validate(); err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s validation failed: %w", device.ID, err))
			continue
		}

		device.UpdatedAt = time.Now()

		metadataJSON, err := marshalJSON(device.Metadata)
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s metadata marshal failed: %w", device.ID, err))
			continue
		}

		sqlResult, err := stmt.ExecContext(ctx,
			device.ID,
			device.Name,
			device.Type,
			device.Version,
			device.Status,
			metadataJSON,
			pq.Array(device.Capabilities),
			device.LastSeen,
			device.UpdatedAt,
		)

		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s update failed: %w", device.ID, err))
			continue
		}

		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s rows affected check failed: %w", device.ID, err))
			continue
		}

		if rowsAffected == 0 {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("device %s not found", device.ID))
			continue
		}

		result.SuccessCount++
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"success_count": result.SuccessCount,
		"failure_count": result.FailureCount,
	}).Info("Bulk device update completed")

	return result, nil
}

// List retrieves devices with filtering and pagination
func (r *deviceRepository) List(ctx context.Context, filter DeviceFilter) ([]*models.Device, error) {
	query, args := r.buildListQuery(filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to list devices")
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device := &models.Device{}
		var metadataJSON []byte
		var capabilities pq.StringArray

		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Type,
			&device.Version,
			&device.Status,
			&metadataJSON,
			&capabilities,
			&device.LastSeen,
			&device.RegisteredAt,
			&device.CreatedAt,
			&device.UpdatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan device")
			continue
		}

		device.Capabilities = []string(capabilities)

		if err := unmarshalJSON(metadataJSON, &device.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal device metadata")
			device.Metadata = make(map[string]interface{})
		}

		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating device rows: %w", err)
	}

	return devices, nil
}

// Count returns the total number of devices matching the filter
func (r *deviceRepository) Count(ctx context.Context, filter DeviceFilter) (int64, error) {
	query, args := r.buildCountQuery(filter)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count devices")
		return 0, fmt.Errorf("failed to count devices: %w", err)
	}

	return count, nil
}

// UpdateStatus updates device status
func (r *deviceRepository) UpdateStatus(ctx context.Context, deviceID string, status models.DeviceStatus) error {
	query := `
		UPDATE devices 
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, deviceID, status, time.Now())
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to update device status")
		return fmt.Errorf("failed to update device status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	r.logger.WithField("device_id", deviceID).WithFields(map[string]interface{}{
		"status": status,
	}).Info("Device status updated")

	return nil
}

// UpdateLastSeen updates the last seen timestamp
func (r *deviceRepository) UpdateLastSeen(ctx context.Context, deviceID string, timestamp time.Time) error {
	query := `
		UPDATE devices 
		SET last_seen = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, deviceID, timestamp, time.Now())
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to update device last seen")
		return fmt.Errorf("failed to update device last seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	return nil
}

// SearchByMetadata searches devices by metadata fields
func (r *deviceRepository) SearchByMetadata(ctx context.Context, metadata map[string]interface{}) ([]*models.Device, error) {
	if len(metadata) == 0 {
		return []*models.Device{}, nil
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	for key, value := range metadata {
		conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at
		FROM devices
		WHERE %s
		ORDER BY created_at DESC
	`, strings.Join(conditions, " AND "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to search devices by metadata")
		return nil, fmt.Errorf("failed to search devices by metadata: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device := &models.Device{}
		var metadataJSON []byte
		var capabilities pq.StringArray

		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Type,
			&device.Version,
			&device.Status,
			&metadataJSON,
			&capabilities,
			&device.LastSeen,
			&device.RegisteredAt,
			&device.CreatedAt,
			&device.UpdatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan device")
			continue
		}

		device.Capabilities = []string(capabilities)

		if err := unmarshalJSON(metadataJSON, &device.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal device metadata")
			device.Metadata = make(map[string]interface{})
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// GetByCapability retrieves devices with a specific capability
func (r *deviceRepository) GetByCapability(ctx context.Context, capability string) ([]*models.Device, error) {
	query := `
		SELECT id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at
		FROM devices
		WHERE $1 = ANY(capabilities)
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, capability)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get devices by capability")
		return nil, fmt.Errorf("failed to get devices by capability: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device := &models.Device{}
		var metadataJSON []byte
		var capabilities pq.StringArray

		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Type,
			&device.Version,
			&device.Status,
			&metadataJSON,
			&capabilities,
			&device.LastSeen,
			&device.RegisteredAt,
			&device.CreatedAt,
			&device.UpdatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan device")
			continue
		}

		device.Capabilities = []string(capabilities)

		if err := unmarshalJSON(metadataJSON, &device.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal device metadata")
			device.Metadata = make(map[string]interface{})
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// GetOnlineDevices retrieves all online devices
func (r *deviceRepository) GetOnlineDevices(ctx context.Context) ([]*models.Device, error) {
	query := `
		SELECT id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at
		FROM devices
		WHERE status = 'online'
		ORDER BY last_seen DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get online devices")
		return nil, fmt.Errorf("failed to get online devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device := &models.Device{}
		var metadataJSON []byte
		var capabilities pq.StringArray

		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Type,
			&device.Version,
			&device.Status,
			&metadataJSON,
			&capabilities,
			&device.LastSeen,
			&device.RegisteredAt,
			&device.CreatedAt,
			&device.UpdatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan device")
			continue
		}

		device.Capabilities = []string(capabilities)

		if err := unmarshalJSON(metadataJSON, &device.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal device metadata")
			device.Metadata = make(map[string]interface{})
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// GetOfflineDevices retrieves devices that have been offline for longer than threshold
func (r *deviceRepository) GetOfflineDevices(ctx context.Context, threshold time.Duration) ([]*models.Device, error) {
	cutoffTime := time.Now().Add(-threshold)

	query := `
		SELECT id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at
		FROM devices
		WHERE (last_seen IS NULL OR last_seen < $1) AND status != 'maintenance'
		ORDER BY last_seen ASC NULLS FIRST
	`

	rows, err := r.db.QueryContext(ctx, query, cutoffTime)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get offline devices")
		return nil, fmt.Errorf("failed to get offline devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device := &models.Device{}
		var metadataJSON []byte
		var capabilities pq.StringArray

		err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Type,
			&device.Version,
			&device.Status,
			&metadataJSON,
			&capabilities,
			&device.LastSeen,
			&device.RegisteredAt,
			&device.CreatedAt,
			&device.UpdatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan device")
			continue
		}

		device.Capabilities = []string(capabilities)

		if err := unmarshalJSON(metadataJSON, &device.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal device metadata")
			device.Metadata = make(map[string]interface{})
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// Helper methods for building queries

// buildListQuery constructs the SQL query for listing devices with filters
func (r *deviceRepository) buildListQuery(filter DeviceFilter) (string, []interface{}) {
	query := `
		SELECT id, name, type, version, status, metadata, capabilities, last_seen, registered_at, created_at, updated_at
		FROM devices
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add WHERE conditions
	if len(filter.DeviceIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.DeviceIDs))
		argIndex++
	}

	if len(filter.Types) > 0 {
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Types))
		argIndex++
	}

	if len(filter.Statuses) > 0 {
		statusStrings := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			statusStrings[i] = string(status)
		}
		conditions = append(conditions, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statusStrings))
		argIndex++
	}

	if len(filter.Capabilities) > 0 {
		for _, capability := range filter.Capabilities {
			conditions = append(conditions, fmt.Sprintf("$%d = ANY(capabilities)", argIndex))
			args = append(args, capability)
			argIndex++
		}
	}

	if filter.LastSeenAfter != nil {
		conditions = append(conditions, fmt.Sprintf("last_seen > $%d", argIndex))
		args = append(args, *filter.LastSeenAfter)
		argIndex++
	}

	if filter.LastSeenBefore != nil {
		conditions = append(conditions, fmt.Sprintf("last_seen < $%d", argIndex))
		args = append(args, *filter.LastSeenBefore)
		argIndex++
	}

	// Add metadata filters
	for key, value := range filter.MetadataFilters {
		conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	orderBy := "created_at"
	if filter.SortBy != "" {
		orderBy = filter.SortBy
	}
	order := "DESC"
	if filter.Order != "" {
		order = strings.ToUpper(filter.Order)
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

	// Add LIMIT and OFFSET
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}

	return query, args
}

// buildCountQuery constructs the SQL query for counting devices with filters
func (r *deviceRepository) buildCountQuery(filter DeviceFilter) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM devices"

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add WHERE conditions (same as buildListQuery but without ORDER BY, LIMIT, OFFSET)
	if len(filter.DeviceIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.DeviceIDs))
		argIndex++
	}

	if len(filter.Types) > 0 {
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Types))
		argIndex++
	}

	if len(filter.Statuses) > 0 {
		statusStrings := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			statusStrings[i] = string(status)
		}
		conditions = append(conditions, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statusStrings))
		argIndex++
	}

	if len(filter.Capabilities) > 0 {
		for _, capability := range filter.Capabilities {
			conditions = append(conditions, fmt.Sprintf("$%d = ANY(capabilities)", argIndex))
			args = append(args, capability)
			argIndex++
		}
	}

	if filter.LastSeenAfter != nil {
		conditions = append(conditions, fmt.Sprintf("last_seen > $%d", argIndex))
		args = append(args, *filter.LastSeenAfter)
		argIndex++
	}

	if filter.LastSeenBefore != nil {
		conditions = append(conditions, fmt.Sprintf("last_seen < $%d", argIndex))
		args = append(args, *filter.LastSeenBefore)
		argIndex++
	}

	// Add metadata filters
	for key, value := range filter.MetadataFilters {
		conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return query, args
}