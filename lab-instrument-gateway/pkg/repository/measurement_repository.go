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

// measurementRepository implements MeasurementRepository interface
type measurementRepository struct {
	db     *db.ConnectionManager
	logger *logger.Logger
}

// NewMeasurementRepository creates a new measurement repository
func NewMeasurementRepository(db *db.ConnectionManager, logger *logger.Logger) MeasurementRepository {
	return &measurementRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new measurement
func (r *measurementRepository) Create(ctx context.Context, measurement *models.Measurement) error {
	if err := measurement.Validate(); err != nil {
		return fmt.Errorf("measurement validation failed: %w", err)
	}

	measurement.SetDefaults()

	query := `
		INSERT INTO measurements (id, device_id, timestamp, type, value, unit, quality, metadata, batch_id, sequence_number, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, err := marshalJSON(measurement.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		measurement.ID,
		measurement.DeviceID,
		measurement.Timestamp,
		measurement.Type,
		measurement.Value,
		measurement.Unit,
		measurement.Quality,
		metadataJSON,
		measurement.BatchID,
		measurement.SequenceNumber,
		measurement.CreatedAt,
	)

	if err != nil {
		r.logger.WithField("device_id", measurement.DeviceID).WithError(err).Error("Failed to create measurement")
		return fmt.Errorf("failed to create measurement: %w", err)
	}

	return nil
}

// GetByID retrieves a measurement by ID
func (r *measurementRepository) GetByID(ctx context.Context, id string) (*models.Measurement, error) {
	query := `
		SELECT id, device_id, timestamp, type, value, unit, quality, metadata, batch_id, sequence_number, created_at
		FROM measurements
		WHERE id = $1
	`

	measurement := &models.Measurement{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&measurement.ID,
		&measurement.DeviceID,
		&measurement.Timestamp,
		&measurement.Type,
		&measurement.Value,
		&measurement.Unit,
		&measurement.Quality,
		&metadataJSON,
		&measurement.BatchID,
		&measurement.SequenceNumber,
		&measurement.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("measurement not found: %s", id)
		}
		r.logger.WithError(err).Error("Failed to get measurement")
		return nil, fmt.Errorf("failed to get measurement: %w", err)
	}

	if err := unmarshalJSON(metadataJSON, &measurement.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return measurement, nil
}

// Delete deletes a measurement by ID
func (r *measurementRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM measurements WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete measurement")
		return fmt.Errorf("failed to delete measurement: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("measurement not found: %s", id)
	}

	return nil
}

// CreateBulk creates multiple measurements in a single transaction for high-throughput scenarios
func (r *measurementRepository) CreateBulk(ctx context.Context, measurements []*models.Measurement) (*BulkResult, error) {
	if len(measurements) == 0 {
		return &BulkResult{}, nil
	}

	result := &BulkResult{}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO measurements (id, device_id, timestamp, type, value, unit, quality, metadata, batch_id, sequence_number, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, measurement := range measurements {
		if err := measurement.Validate(); err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("measurement validation failed: %w", err))
			continue
		}

		measurement.SetDefaults()

		metadataJSON, err := marshalJSON(measurement.Metadata)
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("failed to marshal metadata: %w", err))
			continue
		}

		_, err = stmt.ExecContext(ctx,
			measurement.ID,
			measurement.DeviceID,
			measurement.Timestamp,
			measurement.Type,
			measurement.Value,
			measurement.Unit,
			measurement.Quality,
			metadataJSON,
			measurement.BatchID,
			measurement.SequenceNumber,
			measurement.CreatedAt,
		)

		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Errorf("failed to insert measurement: %w", err))
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
	}).Info("Bulk measurement creation completed")

	return result, nil
}

// CreateBatch creates a batch of measurements with shared metadata
func (r *measurementRepository) CreateBatch(ctx context.Context, batch *models.MeasurementBatch) error {
	if len(batch.Measurements) == 0 {
		return nil
	}

	// Set batch metadata on all measurements
	for i := range batch.Measurements {
		batch.Measurements[i].DeviceID = batch.DeviceID
		batch.Measurements[i].BatchID = &batch.BatchID
		if batch.Measurements[i].Timestamp.IsZero() {
			batch.Measurements[i].Timestamp = batch.Timestamp
		}
	}

	// Convert []Measurement to []*Measurement
	measurements := make([]*models.Measurement, len(batch.Measurements))
	for i := range batch.Measurements {
		measurements[i] = &batch.Measurements[i]
	}
	
	bulkResult, err := r.CreateBulk(ctx, measurements)
	if err != nil {
		return fmt.Errorf("failed to create measurement batch: %w", err)
	}

	if bulkResult.FailureCount > 0 {
		r.logger.WithFields(map[string]interface{}{
			"batch_id":      batch.BatchID,
			"failure_count": bulkResult.FailureCount,
			"errors":        len(bulkResult.Errors),
		}).Warn("Some measurements in batch failed")
	}

	return nil
}

// List retrieves measurements with filtering and pagination
func (r *measurementRepository) List(ctx context.Context, filter MeasurementFilter) ([]*models.Measurement, error) {
	query, args := r.buildListQuery(filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to list measurements")
		return nil, fmt.Errorf("failed to list measurements: %w", err)
	}
	defer rows.Close()

	var measurements []*models.Measurement
	for rows.Next() {
		measurement := &models.Measurement{}
		var metadataJSON []byte

		err := rows.Scan(
			&measurement.ID,
			&measurement.DeviceID,
			&measurement.Timestamp,
			&measurement.Type,
			&measurement.Value,
			&measurement.Unit,
			&measurement.Quality,
			&metadataJSON,
			&measurement.BatchID,
			&measurement.SequenceNumber,
			&measurement.CreatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan measurement")
			continue
		}

		if err := unmarshalJSON(metadataJSON, &measurement.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal measurement metadata")
			measurement.Metadata = make(map[string]interface{})
		}

		measurements = append(measurements, measurement)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating measurement rows: %w", err)
	}

	return measurements, nil
}

// Count returns the total number of measurements matching the filter
func (r *measurementRepository) Count(ctx context.Context, filter MeasurementFilter) (int64, error) {
	query, args := r.buildCountQuery(filter)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count measurements")
		return 0, fmt.Errorf("failed to count measurements: %w", err)
	}

	return count, nil
}

// GetByTimeRange retrieves measurements within a time range
func (r *measurementRepository) GetByTimeRange(ctx context.Context, deviceID string, startTime, endTime time.Time) ([]*models.Measurement, error) {
	query := `
		SELECT id, device_id, timestamp, type, value, unit, quality, metadata, batch_id, sequence_number, created_at
		FROM measurements
		WHERE device_id = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	rows, err := r.db.QueryContext(ctx, query, deviceID, startTime, endTime)
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to get measurements by time range")
		return nil, fmt.Errorf("failed to get measurements by time range: %w", err)
	}
	defer rows.Close()

	var measurements []*models.Measurement
	for rows.Next() {
		measurement := &models.Measurement{}
		var metadataJSON []byte

		err := rows.Scan(
			&measurement.ID,
			&measurement.DeviceID,
			&measurement.Timestamp,
			&measurement.Type,
			&measurement.Value,
			&measurement.Unit,
			&measurement.Quality,
			&metadataJSON,
			&measurement.BatchID,
			&measurement.SequenceNumber,
			&measurement.CreatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan measurement")
			continue
		}

		if err := unmarshalJSON(metadataJSON, &measurement.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal measurement metadata")
			measurement.Metadata = make(map[string]interface{})
		}

		measurements = append(measurements, measurement)
	}

	return measurements, nil
}

// GetLatest retrieves the latest measurement for a device and type
func (r *measurementRepository) GetLatest(ctx context.Context, deviceID string, measurementType string) (*models.Measurement, error) {
	query := `
		SELECT id, device_id, timestamp, type, value, unit, quality, metadata, batch_id, sequence_number, created_at
		FROM measurements
		WHERE device_id = $1 AND type = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	measurement := &models.Measurement{}
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, deviceID, measurementType).Scan(
		&measurement.ID,
		&measurement.DeviceID,
		&measurement.Timestamp,
		&measurement.Type,
		&measurement.Value,
		&measurement.Unit,
		&measurement.Quality,
		&metadataJSON,
		&measurement.BatchID,
		&measurement.SequenceNumber,
		&measurement.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no measurements found for device %s and type %s", deviceID, measurementType)
		}
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to get latest measurement")
		return nil, fmt.Errorf("failed to get latest measurement: %w", err)
	}

	if err := unmarshalJSON(metadataJSON, &measurement.Metadata); err != nil {
		r.logger.WithError(err).Error("Failed to unmarshal measurement metadata")
		measurement.Metadata = make(map[string]interface{})
	}

	return measurement, nil
}

// GetLatestByDevice retrieves the latest measurements for a device (one per type)
func (r *measurementRepository) GetLatestByDevice(ctx context.Context, deviceID string, limit int) ([]*models.Measurement, error) {
	query := `
		SELECT DISTINCT ON (type) id, device_id, timestamp, type, value, unit, quality, metadata, batch_id, sequence_number, created_at
		FROM measurements
		WHERE device_id = $1
		ORDER BY type, timestamp DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, deviceID, limit)
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to get latest measurements by device")
		return nil, fmt.Errorf("failed to get latest measurements by device: %w", err)
	}
	defer rows.Close()

	var measurements []*models.Measurement
	for rows.Next() {
		measurement := &models.Measurement{}
		var metadataJSON []byte

		err := rows.Scan(
			&measurement.ID,
			&measurement.DeviceID,
			&measurement.Timestamp,
			&measurement.Type,
			&measurement.Value,
			&measurement.Unit,
			&measurement.Quality,
			&metadataJSON,
			&measurement.BatchID,
			&measurement.SequenceNumber,
			&measurement.CreatedAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan measurement")
			continue
		}

		if err := unmarshalJSON(metadataJSON, &measurement.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal measurement metadata")
			measurement.Metadata = make(map[string]interface{})
		}

		measurements = append(measurements, measurement)
	}

	return measurements, nil
}

// Aggregate performs data aggregation on measurements
func (r *measurementRepository) Aggregate(ctx context.Context, req AggregationRequest) ([]*AggregationResult, error) {
	query, args := r.buildAggregationQuery(req)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to aggregate measurements")
		return nil, fmt.Errorf("failed to aggregate measurements: %w", err)
	}
	defer rows.Close()

	var results []*AggregationResult
	for rows.Next() {
		result := &AggregationResult{}
		var metadataJSON []byte

		err := rows.Scan(
			&result.DeviceID,
			&result.Type,
			&result.Timestamp,
			&result.Value,
			&result.Count,
			&metadataJSON,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan aggregation result")
			continue
		}

		if err := unmarshalJSON(metadataJSON, &result.Metadata); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal aggregation metadata")
			result.Metadata = make(map[string]interface{})
		}

		results = append(results, result)
	}

	return results, nil
}

// GetStatistics retrieves statistical information for measurements
func (r *measurementRepository) GetStatistics(ctx context.Context, filter MeasurementFilter) (*models.MeasurementStats, error) {
	query, args := r.buildStatsQuery(filter)

	stats := &models.MeasurementStats{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.Count,
		&stats.MinValue,
		&stats.MaxValue,
		&stats.AvgValue,
		&stats.EarliestTime,
		&stats.LatestTime,
		&stats.GoodQuality,
		&stats.BadQuality,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &models.MeasurementStats{}, nil
		}
		r.logger.WithError(err).Error("Failed to get measurement statistics")
		return nil, fmt.Errorf("failed to get measurement statistics: %w", err)
	}

	return stats, nil
}

// DeleteOlderThan removes measurements older than the specified threshold
func (r *measurementRepository) DeleteOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	query := `DELETE FROM measurements WHERE timestamp < $1`

	result, err := r.db.ExecContext(ctx, query, threshold)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete old measurements")
		return 0, fmt.Errorf("failed to delete old measurements: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"deleted":   rowsAffected,
		"threshold": threshold,
	}).Info("Old measurements deleted")

	return rowsAffected, nil
}

// DeleteByDevice removes all measurements for a specific device
func (r *measurementRepository) DeleteByDevice(ctx context.Context, deviceID string) (int64, error) {
	query := `DELETE FROM measurements WHERE device_id = $1`

	result, err := r.db.ExecContext(ctx, query, deviceID)
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to delete measurements by device")
		return 0, fmt.Errorf("failed to delete measurements by device: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.WithField("device_id", deviceID).WithFields(map[string]interface{}{
		"deleted": rowsAffected,
	}).Info("Device measurements deleted")

	return rowsAffected, nil
}

// Helper methods for building queries

// buildListQuery constructs the SQL query for listing measurements with filters
func (r *measurementRepository) buildListQuery(filter MeasurementFilter) (string, []interface{}) {
	query := `
		SELECT id, device_id, timestamp, type, value, unit, quality, metadata, batch_id, sequence_number, created_at
		FROM measurements
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
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Types))
		argIndex++
	}

	if len(filter.Qualities) > 0 {
		qualityStrings := make([]string, len(filter.Qualities))
		for i, quality := range filter.Qualities {
			qualityStrings[i] = string(quality)
		}
		conditions = append(conditions, fmt.Sprintf("quality = ANY($%d)", argIndex))
		args = append(args, pq.Array(qualityStrings))
		argIndex++
	}

	if filter.BatchID != nil {
		conditions = append(conditions, fmt.Sprintf("batch_id = $%d", argIndex))
		args = append(args, *filter.BatchID)
		argIndex++
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	orderBy := "timestamp"
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

// buildCountQuery constructs the SQL query for counting measurements with filters
func (r *measurementRepository) buildCountQuery(filter MeasurementFilter) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM measurements"

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
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Types))
		argIndex++
	}

	if len(filter.Qualities) > 0 {
		qualityStrings := make([]string, len(filter.Qualities))
		for i, quality := range filter.Qualities {
			qualityStrings[i] = string(quality)
		}
		conditions = append(conditions, fmt.Sprintf("quality = ANY($%d)", argIndex))
		args = append(args, pq.Array(qualityStrings))
		argIndex++
	}

	if filter.BatchID != nil {
		conditions = append(conditions, fmt.Sprintf("batch_id = $%d", argIndex))
		args = append(args, *filter.BatchID)
		argIndex++
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return query, args
}

// buildAggregationQuery constructs the SQL query for data aggregation
func (r *measurementRepository) buildAggregationQuery(req AggregationRequest) (string, []interface{}) {
	var aggregateFunc string
	switch req.AggregationType {
	case "avg":
		aggregateFunc = "AVG(value)"
	case "min":
		aggregateFunc = "MIN(value)"
	case "max":
		aggregateFunc = "MAX(value)"
	case "sum":
		aggregateFunc = "SUM(value)"
	case "count":
		aggregateFunc = "COUNT(*)"
	default:
		aggregateFunc = "AVG(value)"
	}

	query := fmt.Sprintf(`
		SELECT 
			device_id,
			type,
			date_trunc('hour', timestamp) as timestamp,
			%s as value,
			COUNT(*) as count,
			'{}'::jsonb as metadata
		FROM measurements
	`, aggregateFunc)

	var conditions []string
	var args []interface{}
	argIndex := 1

	if len(req.DeviceIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("device_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(req.DeviceIDs))
		argIndex++
	}

	if len(req.Types) > 0 {
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(req.Types))
		argIndex++
	}

	if req.TimeRange.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *req.TimeRange.StartTime)
		argIndex++
	}

	if req.TimeRange.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *req.TimeRange.EndTime)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY device_id, type, date_trunc('hour', timestamp) ORDER BY timestamp DESC"

	return query, args
}

// buildStatsQuery constructs the SQL query for measurement statistics
func (r *measurementRepository) buildStatsQuery(filter MeasurementFilter) (string, []interface{}) {
	query := `
		SELECT 
			COUNT(*) as total_count,
			MIN(value) as min_value,
			MAX(value) as max_value,
			AVG(value) as avg_value,
			MIN(timestamp) as earliest_timestamp,
			MAX(timestamp) as latest_timestamp,
			COUNT(CASE WHEN quality = 'good' THEN 1 END) as good_quality_count,
			COUNT(CASE WHEN quality = 'bad' THEN 1 END) as bad_quality_count
		FROM measurements
	`

	var conditions []string
	var args []interface{}
	argIndex := 1

	if len(filter.DeviceIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("device_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.DeviceIDs))
		argIndex++
	}

	if len(filter.Types) > 0 {
		conditions = append(conditions, fmt.Sprintf("type = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Types))
		argIndex++
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filter.StartTime)
		argIndex++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filter.EndTime)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return query, args
}