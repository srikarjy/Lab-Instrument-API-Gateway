package models

import (
	"fmt"
	"time"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeDeviceOffline    AlertType = "device_offline"
	AlertTypeDeviceError      AlertType = "device_error"
	AlertTypeCommandTimeout   AlertType = "command_timeout"
	AlertTypeDataQuality      AlertType = "data_quality"
	AlertTypeSystemHealth     AlertType = "system_health"
	AlertTypeSecurityBreach   AlertType = "security_breach"
	AlertTypePerformance      AlertType = "performance"
)

// Alert represents a system alert or notification
type Alert struct {
	ID              string                 `json:"id" db:"id"`
	DeviceID        *string                `json:"device_id" db:"device_id"`
	Type            AlertType              `json:"type" db:"type"`
	Severity        AlertSeverity          `json:"severity" db:"severity"`
	Message         string                 `json:"message" db:"message"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	Acknowledged    bool                   `json:"acknowledged" db:"acknowledged"`
	AcknowledgedAt  *time.Time             `json:"acknowledged_at" db:"acknowledged_at"`
	AcknowledgedBy  *string                `json:"acknowledged_by" db:"acknowledged_by"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	ResolvedAt      *time.Time             `json:"resolved_at" db:"resolved_at"`
}

// Validate validates the alert data
func (a *Alert) Validate() error {
	if a.Type == "" {
		return fmt.Errorf("alert type is required")
	}
	
	if a.Severity == "" {
		return fmt.Errorf("alert severity is required")
	}
	
	if a.Message == "" {
		return fmt.Errorf("alert message is required")
	}
	
	// Validate severity
	validSeverities := map[AlertSeverity]bool{
		AlertSeverityInfo:     true,
		AlertSeverityWarning:  true,
		AlertSeverityError:    true,
		AlertSeverityCritical: true,
	}
	
	if !validSeverities[a.Severity] {
		return fmt.Errorf("invalid alert severity: %s", a.Severity)
	}
	
	// Validate type
	validTypes := map[AlertType]bool{
		AlertTypeDeviceOffline:  true,
		AlertTypeDeviceError:    true,
		AlertTypeCommandTimeout: true,
		AlertTypeDataQuality:    true,
		AlertTypeSystemHealth:   true,
		AlertTypeSecurityBreach: true,
		AlertTypePerformance:    true,
	}
	
	if !validTypes[a.Type] {
		return fmt.Errorf("invalid alert type: %s", a.Type)
	}
	
	return nil
}

// IsResolved returns true if the alert has been resolved
func (a *Alert) IsResolved() bool {
	return a.ResolvedAt != nil
}

// IsCritical returns true if the alert is critical
func (a *Alert) IsCritical() bool {
	return a.Severity == AlertSeverityCritical
}

// Age returns the age of the alert
func (a *Alert) Age() time.Duration {
	return time.Since(a.CreatedAt)
}

// SetDefaults sets default values for the alert
func (a *Alert) SetDefaults() {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
}

// Acknowledge acknowledges the alert
func (a *Alert) Acknowledge(acknowledgedBy string) {
	a.Acknowledged = true
	now := time.Now()
	a.AcknowledgedAt = &now
	a.AcknowledgedBy = &acknowledgedBy
}

// Resolve resolves the alert
func (a *Alert) Resolve() {
	now := time.Now()
	a.ResolvedAt = &now
}