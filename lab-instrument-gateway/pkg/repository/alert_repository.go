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

// alertRepository implements AlertRepository interface
type alertRepository struct {
	db     *db.ConnectionManager
	logger *logger.Logger
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *db.ConnectionManager, logger *logger.Logger) AlertRepository {
	return &alertRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new alert
func (r *alertRepository) Create(ctx context.Context, alert *models.Alert) error {
	if err := alert.Validate(); err != nil {
		return fmt.Errorf("alert validation failed: %w", err)
	}

	alert.SetDefaults()

	query := `
		INSERT INTO alerts (id, device_id, type, severity, message, metadata, acknowledged, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	metadataJSON, err := marshalJSON(alert.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		alert.ID,
		alert.DeviceID,
		alert.Type,
		alert.Severity,
		alert.Message,
		metadataJSON,
		alert.Acknowledged,
		alert.CreatedAt,
	)

	if err != nil {
		deviceID := ""
		if alert.DeviceID != nil {
			deviceID = *alert.DeviceID
		}
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to create alert")
		return fmt.Errorf("failed to create alert: %w", err)
	}

	deviceID := ""
	if alert.DeviceID != nil {
		deviceID = *alert.DeviceID
	}
	r.logger.WithField("device_id", deviceID).WithFields(map[string]interface{}{
		"alert_id": alert.ID,
		"type":     alert.Type,
		"severity": alert.Severity,
	}).Info("Alert created successfully")

	return nil
}

// GetByID retrieves an alert by ID
func (r *alertRepository) GetByID(ctx context.Context, id string) (*models.Alert, error) {
	query := `
		SELECT id, device_id, type, severity, message, metadata, acknowledged, acknowledged_by, 
		       acknowledged_at, resolved_at, created_at
		FROM alerts
		WHERE id = $1
	`

	alert := &models.Alert{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&alert.ID,
		&alert.DeviceID,
		&alert.Type,
		&alert.Severity,
		&alert.Message,
		&metadataJSON,
		&alert.Acknowledged,
		&alert.AcknowledgedBy,
		&alert.AcknowledgedAt,
		&alert.ResolvedAt,
		&alert.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert not found: %s", id)
		}
		r.logger.WithError(err).Error("Failed to get alert")
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	if err := unmarshalJSON(metadataJSON, &alert.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return alert, nil
}

// Update updates an existing alert
func (r *alertRepository) Update(ctx context.Context, alert *models.Alert) error {
	if err := alert.Validate(); err != nil {
		return fmt.Errorf("alert validation failed: %w", err)
	}

	query := `
		UPDATE alerts 
		SET type = $2, severity = $3, message = $4, metadata = $5, acknowledged = $6, 
		    acknowledged_by = $7, acknowledged_at = $8, resolved_at = $9
		WHERE id = $1
	`

	metadataJSON, err := marshalJSON(alert.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		alert.ID,
		alert.Type,
		alert.Severity,
		alert.Message,
		metadataJSON,
		alert.Acknowledged,
		alert.AcknowledgedBy,
		alert.AcknowledgedAt,
		alert.ResolvedAt,
	)

	if err != nil {
		deviceID := ""
		if alert.DeviceID != nil {
			deviceID = *alert.DeviceID
		}
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to update alert")
		return fmt.Errorf("failed to update alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found: %s", alert.ID)
	}

	deviceID := ""
	if alert.DeviceID != nil {
		deviceID = *alert.DeviceID
	}
	r.logger.WithField("device_id", deviceID).WithFields(map[string]interface{}{
		"alert_id": alert.ID,
		"severity": alert.Severity,
	}).Info("Alert updated successfully")

	return nil
}

// Delete removes an alert
func (r *alertRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM alerts WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete alert")
		return fmt.Errorf("failed to delete alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found: %s", id)
	}

	r.logger.WithFields(map[string]interface{}{
		"alert_id": id,
	}).Info("Alert deleted successfully")

	return nil
}

// List retrieves alerts with filtering and pagination
func (r *alertRepository) List(ctx context.Context, filter AlertFilter) ([]*models.Alert, error) {
	query, args := r.buildListQuery(filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to list alerts")
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		alert := &models.Alert{}
		var metadataJSON []byte

		err := rows.Scan(
			&alert.ID,
			&alert.DeviceID,
			&alert.Type,
			&alert.Severity,
			&alert.Message,
			&metadataJSON,
			&alert.Acknowledged,
			&alert.AcknowledgedBy,
			&alert.AcknowledgedAt,
			&alert.ResolvedAt,
			&alert.CreatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan alert")
			continue
		}

		if err := unmarshalJSON(metadataJSON, &alert.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal alert metadata")
			alert.Metadata = make(map[string]interface{})
		}

		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alert rows: %w", err)
	}

	return alerts, nil
}

// Count returns the total number of alerts matching the filter
func (r *alertRepository) Count(ctx context.Context, filter AlertFilter) (int64, error) {
	query, args := r.buildCountQuery(filter)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count alerts")
		return 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	return count, nil
}

// Acknowledge acknowledges an alert
func (r *alertRepository) Acknowledge(ctx context.Context, alertID string, acknowledgedBy string) error {
	query := `
		UPDATE alerts 
		SET acknowledged = true, acknowledged_by = $2, acknowledged_at = $3
		WHERE id = $1 AND acknowledged = false
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, alertID, acknowledgedBy, now)
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"alert_id":        alertID,
			"acknowledged_by": acknowledgedBy,
		}).WithError(err).Error("Failed to acknowledge alert")
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found or already acknowledged: %s", alertID)
	}

	r.logger.WithFields(map[string]interface{}{
		"alert_id":        alertID,
		"acknowledged_by": acknowledgedBy,
	}).Info("Alert acknowledged")

	return nil
}

// Resolve resolves an alert
func (r *alertRepository) Resolve(ctx context.Context, alertID string) error {
	query := `
		UPDATE alerts 
		SET resolved_at = $2
		WHERE id = $1 AND resolved_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, alertID, now)
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"alert_id": alertID,
		}).WithError(err).Error("Failed to resolve alert")
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert not found or already resolved: %s", alertID)
	}

	r.logger.WithFields(map[string]interface{}{
		"alert_id": alertID,
	}).Info("Alert resolved")

	return nil
}

// GetUnacknowledged retrieves all unacknowledged alerts
func (r *alertRepository) GetUnacknowledged(ctx context.Context) ([]*models.Alert, error) {
	query := `
		SELECT id, device_id, type, severity, message, metadata, acknowledged, acknowledged_by, 
		       acknowledged_at, resolved_at, created_at
		FROM alerts
		WHERE acknowledged = false
		ORDER BY severity DESC, created_at DESC
	`

	return r.executeQuery(ctx, query)
}

// GetUnresolved retrieves all unresolved alerts
func (r *alertRepository) GetUnresolved(ctx context.Context) ([]*models.Alert, error) {
	query := `
		SELECT id, device_id, type, severity, message, metadata, acknowledged, acknowledged_by, 
		       acknowledged_at, resolved_at, created_at
		FROM alerts
		WHERE resolved_at IS NULL
		ORDER BY severity DESC, created_at DESC
	`

	return r.executeQuery(ctx, query)
}

// GetCriticalAlerts retrieves all critical alerts
func (r *alertRepository) GetCriticalAlerts(ctx context.Context) ([]*models.Alert, error) {
	query := `
		SELECT id, device_id, type, severity, message, metadata, acknowledged, acknowledged_by, 
		       acknowledged_at, resolved_at, created_at
		FROM alerts
		WHERE severity = 'critical' AND resolved_at IS NULL
		ORDER BY created_at DESC
	`

	return r.executeQuery(ctx, query)
}

// GetAlertStats retrieves alert statistics for a time range
func (r *alertRepository) GetAlertStats(ctx context.Context, timeRange TimeRangeFilter) (map[models.AlertSeverity]int64, error) {
	query := `
		SELECT severity, COUNT(*) as count
		FROM alerts
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if timeRange.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *timeRange.StartTime)
		argIndex++
	}

	if timeRange.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *timeRange.EndTime)
		argIndex++
	}

	query += " GROUP BY severity"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get alert statistics")
		return nil, fmt.Errorf("failed to get alert statistics: %w", err)
	}
	defer rows.Close()

	stats := make(map[models.AlertSeverity]int64)
	for rows.Next() {
		var severity models.AlertSeverity
		var count int64

		err := rows.Scan(&severity, &count)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan alert stats")
			continue
		}

		stats[severity] = count
	}

	return stats, nil
}

// GetAlertsByDevice retrieves alerts for a specific device
func (r *alertRepository) GetAlertsByDevice(ctx context.Context, deviceID string, limit int) ([]*models.Alert, error) {
	query := `
		SELECT id, device_id, type, severity, message, metadata, acknowledged, acknowledged_by, 
		       acknowledged_at, resolved_at, created_at
		FROM alerts
		WHERE device_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, deviceID, limit)
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to get alerts by device")
		return nil, fmt.Errorf("failed to get alerts by device: %w", err)
	}
	defer rows.Close()

	return r.scanAlerts(rows)
}

// DeleteResolvedOlderThan removes resolved alerts older than the specified threshold
func (r *alertRepository) DeleteResolvedOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	query := `
		DELETE FROM alerts 
		WHERE resolved_at IS NOT NULL AND resolved_at < $1
	`

	result, err := r.db.ExecContext(ctx, query, threshold)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete old resolved alerts")
		return 0, fmt.Errorf("failed to delete old resolved alerts: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		r.logger.WithFields(map[string]interface{}{
			"deleted":   rowsAffected,
			"threshold": threshold,
		}).Info("Old resolved alerts deleted")
	}

	return rowsAffected, nil
}

// Helper methods

// executeQuery executes a query and returns alerts
func (r *alertRepository) executeQuery(ctx context.Context, query string, args ...interface{}) ([]*models.Alert, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to execute alert query")
		return nil, fmt.Errorf("failed to execute alert query: %w", err)
	}
	defer rows.Close()

	return r.scanAlerts(rows)
}

// scanAlerts scans rows into alert objects
func (r *alertRepository) scanAlerts(rows *sql.Rows) ([]*models.Alert, error) {
	var alerts []*models.Alert
	for rows.Next() {
		alert := &models.Alert{}
		var metadataJSON []byte

		err := rows.Scan(
			&alert.ID,
			&alert.DeviceID,
			&alert.Type,
			&alert.Severity,
			&alert.Message,
			&metadataJSON,
			&alert.Acknowledged,
			&alert.AcknowledgedBy,
			&alert.AcknowledgedAt,
			&alert.ResolvedAt,
			&alert.CreatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan alert")
			continue
		}

		if err := unmarshalJSON(metadataJSON, &alert.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal alert metadata")
			alert.Metadata = make(map[string]interface{})
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// buildListQuery constructs the SQL query for listing alerts with filters
func (r *alertRepository) buildListQuery(filter AlertFilter) (string, []interface{}) {
	query := `
		SELECT id, device_id, type, severity, message, metadata, acknowledged, acknowledged_by, 
		       acknowledged_at, resolved_at, created_at
		FROM alerts
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add WHERE conditions
	if len(filter.DeviceIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("device_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.DeviceIDs))
		argIndex++
	}

	if len(filter.Types) > 0 {
		typeStrings := make([]string, len(filter.Types))
		for i, alertType := range filter.Types {
			typeStrings[i] = string(alertType)
		}
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(typeStrings))
		argIndex++
	}

	if len(filter.Severities) > 0 {
		severityStrings := make([]string, len(filter.Severities))
		for i, severity := range filter.Severities {
			severityStrings[i] = string(severity)
		}
		conditions = append(conditions, fmt.Sprintf("severity = ANY($%d)", argIndex))
		args = append(args, pq.Array(severityStrings))
		argIndex++
	}

	if filter.Acknowledged != nil {
		conditions = append(conditions, fmt.Sprintf("acknowledged = $%d", argIndex))
		args = append(args, *filter.Acknowledged)
		argIndex++
	}

	if filter.Resolved != nil {
		if *filter.Resolved {
			conditions = append(conditions, "resolved_at IS NOT NULL")
		} else {
			conditions = append(conditions, "resolved_at IS NULL")
		}
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndTime)
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

// buildCountQuery constructs the SQL query for counting alerts with filters
func (r *alertRepository) buildCountQuery(filter AlertFilter) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM alerts"

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add WHERE conditions (same as buildListQuery but without ORDER BY, LIMIT, OFFSET)
	if len(filter.DeviceIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("device_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.DeviceIDs))
		argIndex++
	}

	if len(filter.Types) > 0 {
		typeStrings := make([]string, len(filter.Types))
		for i, alertType := range filter.Types {
			typeStrings[i] = string(alertType)
		}
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(typeStrings))
		argIndex++
	}

	if len(filter.Severities) > 0 {
		severityStrings := make([]string, len(filter.Severities))
		for i, severity := range filter.Severities {
			severityStrings[i] = string(severity)
		}
		conditions = append(conditions, fmt.Sprintf("severity = ANY($%d)", argIndex))
		args = append(args, pq.Array(severityStrings))
		argIndex++
	}

	if filter.Acknowledged != nil {
		conditions = append(conditions, fmt.Sprintf("acknowledged = $%d", argIndex))
		args = append(args, *filter.Acknowledged)
		argIndex++
	}

	if filter.Resolved != nil {
		if *filter.Resolved {
			conditions = append(conditions, "resolved_at IS NOT NULL")
		} else {
			conditions = append(conditions, "resolved_at IS NULL")
		}
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return query, args
}