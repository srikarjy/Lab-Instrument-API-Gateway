package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// DeviceStatus represents the status of a device
type DeviceStatus string

const (
	DeviceStatusUnknown     DeviceStatus = "unknown"
	DeviceStatusOnline      DeviceStatus = "online"
	DeviceStatusOffline     DeviceStatus = "offline"
	DeviceStatusError       DeviceStatus = "error"
	DeviceStatusMaintenance DeviceStatus = "maintenance"
	DeviceStatusConnecting  DeviceStatus = "connecting"
)

// Device represents a laboratory instrument device
type Device struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Type         string                 `json:"type" db:"type"`
	Version      string                 `json:"version" db:"version"`
	Status       DeviceStatus           `json:"status" db:"status"`
	Metadata     map[string]interface{} `json:"metadata" db:"metadata"`
	Capabilities pq.StringArray         `json:"capabilities" db:"capabilities"`
	LastSeen     *time.Time             `json:"last_seen" db:"last_seen"`
	RegisteredAt time.Time              `json:"registered_at" db:"registered_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// DeviceSession represents an active device connection session
type DeviceSession struct {
	ID            string                 `json:"id" db:"id"`
	DeviceID      string                 `json:"device_id" db:"device_id"`
	SessionID     string                 `json:"session_id" db:"session_id"`
	StreamID      *string                `json:"stream_id" db:"stream_id"`
	ConnectedAt   time.Time              `json:"connected_at" db:"connected_at"`
	LastHeartbeat time.Time              `json:"last_heartbeat" db:"last_heartbeat"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	IsActive      bool                   `json:"is_active" db:"is_active"`
}

// Validate validates the device data
func (d *Device) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("device ID is required")
	}
	
	if d.Name == "" {
		return fmt.Errorf("device name is required")
	}
	
	if d.Type == "" {
		return fmt.Errorf("device type is required")
	}
	
	// Validate status
	validStatuses := map[DeviceStatus]bool{
		DeviceStatusUnknown:     true,
		DeviceStatusOnline:      true,
		DeviceStatusOffline:     true,
		DeviceStatusError:       true,
		DeviceStatusMaintenance: true,
		DeviceStatusConnecting:  true,
	}
	
	if !validStatuses[d.Status] {
		return fmt.Errorf("invalid device status: %s", d.Status)
	}
	
	return nil
}

// IsOnline returns true if the device is currently online
func (d *Device) IsOnline() bool {
	return d.Status == DeviceStatusOnline
}

// IsHealthy returns true if the device is in a healthy state
func (d *Device) IsHealthy() bool {
	return d.Status == DeviceStatusOnline || d.Status == DeviceStatusConnecting
}

// HasCapability checks if the device has a specific capability
func (d *Device) HasCapability(capability string) bool {
	for _, cap := range d.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// UpdateLastSeen updates the last seen timestamp
func (d *Device) UpdateLastSeen() {
	now := time.Now()
	d.LastSeen = &now
	d.UpdatedAt = now
}

// SetStatus updates the device status and timestamp
func (d *Device) SetStatus(status DeviceStatus) {
	d.Status = status
	d.UpdatedAt = time.Now()
}

// Value implements the driver.Valuer interface for DeviceStatus
func (ds DeviceStatus) Value() (driver.Value, error) {
	return string(ds), nil
}

// Scan implements the sql.Scanner interface for DeviceStatus
func (ds *DeviceStatus) Scan(value interface{}) error {
	if value == nil {
		*ds = DeviceStatusUnknown
		return nil
	}
	
	switch v := value.(type) {
	case string:
		*ds = DeviceStatus(v)
	case []byte:
		*ds = DeviceStatus(v)
	default:
		return fmt.Errorf("cannot scan %T into DeviceStatus", value)
	}
	
	return nil
}

// MarshalJSON implements json.Marshaler for DeviceStatus
func (ds DeviceStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ds))
}

// UnmarshalJSON implements json.Unmarshaler for DeviceStatus
func (ds *DeviceStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*ds = DeviceStatus(s)
	return nil
}