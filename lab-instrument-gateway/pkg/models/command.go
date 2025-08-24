package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// CommandStatus represents the status of a command
type CommandStatus string

const (
	CommandStatusUnknown   CommandStatus = "unknown"
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusExecuting CommandStatus = "executing"
	CommandStatusCompleted CommandStatus = "completed"
	CommandStatusFailed    CommandStatus = "failed"
	CommandStatusTimeout   CommandStatus = "timeout"
	CommandStatusCancelled CommandStatus = "cancelled"
)

// Command represents a command sent to a device
type Command struct {
	ID              string                 `json:"id" db:"id"`
	DeviceID        string                 `json:"device_id" db:"device_id"`
	CommandID       string                 `json:"command_id" db:"command_id"`
	Type            string                 `json:"type" db:"type"`
	Parameters      map[string]interface{} `json:"parameters" db:"parameters"`
	Status          CommandStatus          `json:"status" db:"status"`
	Priority        int                    `json:"priority" db:"priority"`
	TimeoutSeconds  int                    `json:"timeout_seconds" db:"timeout_seconds"`
	Result          map[string]interface{} `json:"result" db:"result"`
	ErrorMessage    *string                `json:"error_message" db:"error_message"`
	SubmittedAt     time.Time              `json:"submitted_at" db:"submitted_at"`
	ExecutedAt      *time.Time             `json:"executed_at" db:"executed_at"`
	CompletedAt     *time.Time             `json:"completed_at" db:"completed_at"`
	ExpiresAt       *time.Time             `json:"expires_at" db:"expires_at"`
	ExecutionTimeMs *float64               `json:"execution_time_ms" db:"execution_time_ms"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success         bool                   `json:"success"`
	Message         string                 `json:"message"`
	Data            map[string]interface{} `json:"data"`
	ExecutedAt      time.Time              `json:"executed_at"`
	ExecutionTimeMs float64                `json:"execution_time_ms"`
}

// Validate validates the command data
func (c *Command) Validate() error {
	if c.DeviceID == "" {
		return fmt.Errorf("device ID is required")
	}
	
	if c.CommandID == "" {
		return fmt.Errorf("command ID is required")
	}
	
	if c.Type == "" {
		return fmt.Errorf("command type is required")
	}
	
	if c.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout seconds must be positive")
	}
	
	// Validate status
	validStatuses := map[CommandStatus]bool{
		CommandStatusUnknown:   true,
		CommandStatusPending:   true,
		CommandStatusExecuting: true,
		CommandStatusCompleted: true,
		CommandStatusFailed:    true,
		CommandStatusTimeout:   true,
		CommandStatusCancelled: true,
	}
	
	if !validStatuses[c.Status] {
		return fmt.Errorf("invalid command status: %s", c.Status)
	}
	
	return nil
}

// IsCompleted returns true if the command has completed (success or failure)
func (c *Command) IsCompleted() bool {
	return c.Status == CommandStatusCompleted || 
		   c.Status == CommandStatusFailed || 
		   c.Status == CommandStatusTimeout || 
		   c.Status == CommandStatusCancelled
}

// IsExpired returns true if the command has expired
func (c *Command) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// CanExecute returns true if the command can be executed
func (c *Command) CanExecute() bool {
	return c.Status == CommandStatusPending && !c.IsExpired()
}

// SetDefaults sets default values for the command
func (c *Command) SetDefaults() {
	if c.Status == "" {
		c.Status = CommandStatusPending
	}
	
	if c.Priority == 0 {
		c.Priority = 1
	}
	
	if c.TimeoutSeconds == 0 {
		c.TimeoutSeconds = 30
	}
	
	now := time.Now()
	if c.SubmittedAt.IsZero() {
		c.SubmittedAt = now
	}
	
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	
	c.UpdatedAt = now
	
	if c.Parameters == nil {
		c.Parameters = make(map[string]interface{})
	}
	
	if c.Result == nil {
		c.Result = make(map[string]interface{})
	}
	
	// Set expiration time if not set
	if c.ExpiresAt == nil {
		expiresAt := c.SubmittedAt.Add(time.Duration(c.TimeoutSeconds) * time.Second)
		c.ExpiresAt = &expiresAt
	}
}

// StartExecution marks the command as executing
func (c *Command) StartExecution() {
	c.Status = CommandStatusExecuting
	now := time.Now()
	c.ExecutedAt = &now
	c.UpdatedAt = now
}

// CompleteExecution marks the command as completed with result
func (c *Command) CompleteExecution(result CommandResult) {
	if result.Success {
		c.Status = CommandStatusCompleted
	} else {
		c.Status = CommandStatusFailed
		if result.Message != "" {
			c.ErrorMessage = &result.Message
		}
	}
	
	now := time.Now()
	c.CompletedAt = &now
	c.UpdatedAt = now
	
	// Calculate execution time
	if c.ExecutedAt != nil {
		executionTime := now.Sub(*c.ExecutedAt).Seconds() * 1000 // Convert to milliseconds
		c.ExecutionTimeMs = &executionTime
	}
	
	// Store result data
	if result.Data != nil {
		c.Result = result.Data
	}
}

// MarkTimeout marks the command as timed out
func (c *Command) MarkTimeout() {
	c.Status = CommandStatusTimeout
	now := time.Now()
	c.CompletedAt = &now
	c.UpdatedAt = now
	
	errorMsg := "Command execution timed out"
	c.ErrorMessage = &errorMsg
}

// Cancel marks the command as cancelled
func (c *Command) Cancel(reason string) {
	c.Status = CommandStatusCancelled
	now := time.Now()
	c.CompletedAt = &now
	c.UpdatedAt = now
	
	if reason != "" {
		c.ErrorMessage = &reason
	}
}

// Value implements the driver.Valuer interface for CommandStatus
func (cs CommandStatus) Value() (driver.Value, error) {
	return string(cs), nil
}

// Scan implements the sql.Scanner interface for CommandStatus
func (cs *CommandStatus) Scan(value interface{}) error {
	if value == nil {
		*cs = CommandStatusUnknown
		return nil
	}
	
	switch v := value.(type) {
	case string:
		*cs = CommandStatus(v)
	case []byte:
		*cs = CommandStatus(v)
	default:
		return fmt.Errorf("cannot scan %T into CommandStatus", value)
	}
	
	return nil
}

// MarshalJSON implements json.Marshaler for CommandStatus
func (cs CommandStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(cs))
}

// UnmarshalJSON implements json.Unmarshaler for CommandStatus
func (cs *CommandStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*cs = CommandStatus(s)
	return nil
}