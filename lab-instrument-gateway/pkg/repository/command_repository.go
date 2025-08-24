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

// commandRepository implements CommandRepository interface
type commandRepository struct {
	db     *db.ConnectionManager
	logger *logger.Logger
}

// NewCommandRepository creates a new command repository
func NewCommandRepository(db *db.ConnectionManager, logger *logger.Logger) CommandRepository {
	return &commandRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new command
func (r *commandRepository) Create(ctx context.Context, command *models.Command) error {
	if err := command.Validate(); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	command.SetDefaults()

	query := `
		INSERT INTO commands (id, command_id, device_id, type, parameters, status, priority, timeout_seconds, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	parametersJSON, err := marshalJSON(command.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		command.ID,
		command.CommandID,
		command.DeviceID,
		command.Type,
		parametersJSON,
		command.Status,
		command.Priority,
		command.TimeoutSeconds,
		command.CreatedAt,
		command.UpdatedAt,
		command.ExpiresAt,
	)

	if err != nil {
		r.logger.WithField("device_id", command.DeviceID).WithError(err).Error("Failed to create command")
		return fmt.Errorf("failed to create command: %w", err)
	}

	r.logger.WithField("device_id", command.DeviceID).WithFields(map[string]interface{}{
		"command_id": command.CommandID,
		"type":       command.Type,
	}).Info("Command created successfully")

	return nil
}

// GetByID retrieves a command by ID
func (r *commandRepository) GetByID(ctx context.Context, id string) (*models.Command, error) {
	query := `
		SELECT id, command_id, device_id, type, parameters, status, priority, timeout_seconds, 
		       result, error_message, executed_at, created_at, updated_at, expires_at
		FROM commands
		WHERE id = $1
	`

	command := &models.Command{}
	var parametersJSON, resultJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&command.ID,
		&command.CommandID,
		&command.DeviceID,
		&command.Type,
		&parametersJSON,
		&command.Status,
		&command.Priority,
		&command.TimeoutSeconds,
		&resultJSON,
		&command.ErrorMessage,
		&command.ExecutedAt,
		&command.CreatedAt,
		&command.UpdatedAt,
		&command.ExpiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("command not found: %s", id)
		}
		r.logger.WithError(err).Error("Failed to get command")
		return nil, fmt.Errorf("failed to get command: %w", err)
	}

	if err := unmarshalJSON(parametersJSON, &command.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	if len(resultJSON) > 0 {
		if err := unmarshalJSON(resultJSON, &command.Result); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal command result")
			command.Result = make(map[string]interface{})
		}
	} else {
		command.Result = make(map[string]interface{})
	}

	return command, nil
}

// GetByCommandID retrieves a command by command ID
func (r *commandRepository) GetByCommandID(ctx context.Context, commandID string) (*models.Command, error) {
	query := `
		SELECT id, command_id, device_id, type, parameters, status, priority, timeout_seconds, 
		       result, error_message, executed_at, created_at, updated_at, expires_at
		FROM commands
		WHERE command_id = $1
	`

	command := &models.Command{}
	var parametersJSON, resultJSON []byte

	err := r.db.QueryRowContext(ctx, query, commandID).Scan(
		&command.ID,
		&command.CommandID,
		&command.DeviceID,
		&command.Type,
		&parametersJSON,
		&command.Status,
		&command.Priority,
		&command.TimeoutSeconds,
		&resultJSON,
		&command.ErrorMessage,
		&command.ExecutedAt,
		&command.CreatedAt,
		&command.UpdatedAt,
		&command.ExpiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("command not found: %s", commandID)
		}
		r.logger.WithError(err).Error("Failed to get command by command ID")
		return nil, fmt.Errorf("failed to get command by command ID: %w", err)
	}

	if err := unmarshalJSON(parametersJSON, &command.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	if len(resultJSON) > 0 {
		if err := unmarshalJSON(resultJSON, &command.Result); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal command result")
			command.Result = make(map[string]interface{})
		}
	} else {
		command.Result = make(map[string]interface{})
	}

	return command, nil
}

// Update updates an existing command
func (r *commandRepository) Update(ctx context.Context, command *models.Command) error {
	if err := command.Validate(); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	command.UpdatedAt = time.Now()

	query := `
		UPDATE commands 
		SET type = $2, parameters = $3, status = $4, priority = $5, timeout_seconds = $6, 
		    result = $7, error_message = $8, executed_at = $9, updated_at = $10, expires_at = $11
		WHERE id = $1
	`

	parametersJSON, err := marshalJSON(command.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	resultJSON, err := marshalJSON(command.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		command.ID,
		command.Type,
		parametersJSON,
		command.Status,
		command.Priority,
		command.TimeoutSeconds,
		resultJSON,
		command.ErrorMessage,
		command.ExecutedAt,
		command.UpdatedAt,
		command.ExpiresAt,
	)

	if err != nil {
		r.logger.WithField("device_id", command.DeviceID).WithError(err).Error("Failed to update command")
		return fmt.Errorf("failed to update command: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("command not found: %s", command.ID)
	}

	r.logger.WithField("device_id", command.DeviceID).WithFields(map[string]interface{}{
		"command_id": command.CommandID,
		"status":     command.Status,
	}).Info("Command updated successfully")

	return nil
}

// Delete removes a command
func (r *commandRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM commands WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete command")
		return fmt.Errorf("failed to delete command: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("command not found: %s", id)
	}

	r.logger.WithFields(map[string]interface{}{
		"command_id": id,
	}).Info("Command deleted successfully")

	return nil
}

// List retrieves commands with filtering and pagination
func (r *commandRepository) List(ctx context.Context, filter CommandFilter) ([]*models.Command, error) {
	query, args := r.buildListQuery(filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithError(err).Error("Failed to list commands")
		return nil, fmt.Errorf("failed to list commands: %w", err)
	}
	defer rows.Close()

	return r.scanCommands(rows)
}

// Count returns the total number of commands matching the filter
func (r *commandRepository) Count(ctx context.Context, filter CommandFilter) (int64, error) {
	query, args := r.buildCountQuery(filter)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count commands")
		return 0, fmt.Errorf("failed to count commands: %w", err)
	}

	return count, nil
}

// GetPendingCommands retrieves pending commands for a device
func (r *commandRepository) GetPendingCommands(ctx context.Context, deviceID string) ([]*models.Command, error) {
	query := `
		SELECT id, command_id, device_id, type, parameters, status, priority, timeout_seconds, 
		       result, error_message, executed_at, created_at, updated_at, expires_at
		FROM commands
		WHERE device_id = $1 AND status = 'pending' AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to get pending commands")
		return nil, fmt.Errorf("failed to get pending commands: %w", err)
	}
	defer rows.Close()

	return r.scanCommands(rows)
}

// GetExecutingCommands retrieves executing commands for a device
func (r *commandRepository) GetExecutingCommands(ctx context.Context, deviceID string) ([]*models.Command, error) {
	query := `
		SELECT id, command_id, device_id, type, parameters, status, priority, timeout_seconds, 
		       result, error_message, executed_at, created_at, updated_at, expires_at
		FROM commands
		WHERE device_id = $1 AND status = 'executing'
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to get executing commands")
		return nil, fmt.Errorf("failed to get executing commands: %w", err)
	}
	defer rows.Close()

	return r.scanCommands(rows)
}

// UpdateStatus updates the status of a command
func (r *commandRepository) UpdateStatus(ctx context.Context, commandID string, status models.CommandStatus) error {
	query := `
		UPDATE commands 
		SET status = $2, updated_at = $3
		WHERE command_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, commandID, status, time.Now())
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"command_id": commandID,
			"status":     status,
		}).WithError(err).Error("Failed to update command status")
		return fmt.Errorf("failed to update command status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("command not found: %s", commandID)
	}

	r.logger.WithFields(map[string]interface{}{
		"command_id": commandID,
		"status":     status,
	}).Info("Command status updated")

	return nil
}

// GetExpiredCommands retrieves commands that have expired
func (r *commandRepository) GetExpiredCommands(ctx context.Context) ([]*models.Command, error) {
	query := `
		SELECT id, command_id, device_id, type, parameters, status, priority, timeout_seconds, 
		       result, error_message, executed_at, created_at, updated_at, expires_at
		FROM commands
		WHERE expires_at IS NOT NULL AND expires_at <= NOW() AND status IN ('pending', 'executing')
		ORDER BY expires_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get expired commands")
		return nil, fmt.Errorf("failed to get expired commands: %w", err)
	}
	defer rows.Close()

	return r.scanCommands(rows)
}

// MarkExpiredAsTimeout marks expired commands as timed out
func (r *commandRepository) MarkExpiredAsTimeout(ctx context.Context) (int64, error) {
	query := `
		UPDATE commands 
		SET status = 'timeout', error_message = 'Command expired', updated_at = NOW()
		WHERE expires_at IS NOT NULL AND expires_at <= NOW() AND status IN ('pending', 'executing')
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		r.logger.WithError(err).Error("Failed to mark expired commands as timeout")
		return 0, fmt.Errorf("failed to mark expired commands as timeout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		r.logger.WithFields(map[string]interface{}{
			"expired_count": rowsAffected,
		}).Info("Expired commands marked as timeout")
	}

	return rowsAffected, nil
}

// DeleteCompletedOlderThan removes completed commands older than the specified threshold
func (r *commandRepository) DeleteCompletedOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	query := `
		DELETE FROM commands 
		WHERE created_at < $1 AND status IN ('completed', 'failed', 'timeout')
	`

	result, err := r.db.ExecContext(ctx, query, threshold)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete old completed commands")
		return 0, fmt.Errorf("failed to delete old completed commands: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		r.logger.WithFields(map[string]interface{}{
			"deleted":   rowsAffected,
			"threshold": threshold,
		}).Info("Old completed commands deleted")
	}

	return rowsAffected, nil
}

// GetCommandStats retrieves command statistics for a device
func (r *commandRepository) GetCommandStats(ctx context.Context, deviceID string, timeRange TimeRangeFilter) (map[models.CommandStatus]int64, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM commands
		WHERE device_id = $1
	`

	args := []interface{}{deviceID}
	argIndex := 2

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

	query += " GROUP BY status"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.WithField("device_id", deviceID).WithError(err).Error("Failed to get command statistics")
		return nil, fmt.Errorf("failed to get command statistics: %w", err)
	}
	defer rows.Close()

	stats := make(map[models.CommandStatus]int64)
	for rows.Next() {
		var status models.CommandStatus
		var count int64

		err := rows.Scan(&status, &count)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan command stats")
			continue
		}

		stats[status] = count
	}

	return stats, nil
}

// Helper methods

// scanCommands scans rows into command objects
func (r *commandRepository) scanCommands(rows *sql.Rows) ([]*models.Command, error) {
	var commands []*models.Command
	for rows.Next() {
		command := &models.Command{}
		var parametersJSON, resultJSON []byte

		err := rows.Scan(
			&command.ID,
			&command.CommandID,
			&command.DeviceID,
			&command.Type,
			&parametersJSON,
			&command.Status,
			&command.Priority,
			&command.TimeoutSeconds,
			&resultJSON,
			&command.ErrorMessage,
			&command.ExecutedAt,
			&command.CreatedAt,
			&command.UpdatedAt,
			&command.ExpiresAt,
		)

		if err != nil {
			r.logger.WithError(err).Error("Failed to scan command")
			continue
		}

		if err := unmarshalJSON(parametersJSON, &command.Parameters); err != nil {
			r.logger.WithError(err).Error("Failed to unmarshal command parameters")
			command.Parameters = make(map[string]interface{})
		}

		if len(resultJSON) > 0 {
			if err := unmarshalJSON(resultJSON, &command.Result); err != nil {
				r.logger.WithError(err).Error("Failed to unmarshal command result")
				command.Result = make(map[string]interface{})
			}
		} else {
			command.Result = make(map[string]interface{})
		}

		commands = append(commands, command)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating command rows: %w", err)
	}

	return commands, nil
}

// buildListQuery constructs the SQL query for listing commands with filters
func (r *commandRepository) buildListQuery(filter CommandFilter) (string, []interface{}) {
	query := `
		SELECT id, command_id, device_id, type, parameters, status, priority, timeout_seconds, 
		       result, error_message, executed_at, created_at, updated_at, expires_at
		FROM commands
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

	if len(filter.Statuses) > 0 {
		statusStrings := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			statusStrings[i] = string(status)
		}
		conditions = append(conditions, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statusStrings))
		argIndex++
	}

	if len(filter.Priorities) > 0 {
		conditions = append(conditions, fmt.Sprintf("priority = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Priorities))
		argIndex++
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

// buildCountQuery constructs the SQL query for counting commands with filters
func (r *commandRepository) buildCountQuery(filter CommandFilter) (string, []interface{}) {
	query := "SELECT COUNT(*) FROM commands"

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

	if len(filter.Statuses) > 0 {
		statusStrings := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			statusStrings[i] = string(status)
		}
		conditions = append(conditions, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statusStrings))
		argIndex++
	}

	if len(filter.Priorities) > 0 {
		conditions = append(conditions, fmt.Sprintf("priority = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Priorities))
		argIndex++
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