package repository

import (
	"context"
	"errors"
	"time"

	"github.com/yourorg/lab-gateway/pkg/models"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")
)

// Common interfaces for filtering, sorting, and pagination

// Filter represents common filtering options
type Filter struct {
	Limit  int
	Offset int
	SortBy string
	Order  string // "ASC" or "DESC"
}

// TimeRangeFilter represents time-based filtering
type TimeRangeFilter struct {
	StartTime *time.Time
	EndTime   *time.Time
}

// DeviceFilter represents device-specific filtering options
type DeviceFilter struct {
	Filter
	DeviceIDs    []string
	Types        []string
	Statuses     []models.DeviceStatus
	Capabilities []string
	LastSeenAfter *time.Time
	LastSeenBefore *time.Time
	MetadataFilters map[string]interface{}
}

// MeasurementFilter represents measurement-specific filtering options
type MeasurementFilter struct {
	Filter
	TimeRangeFilter
	DeviceIDs []string
	Types     []string
	Qualities []models.QualityCode
	BatchID   *string
}

// CommandFilter represents command-specific filtering options
type CommandFilter struct {
	Filter
	TimeRangeFilter
	DeviceIDs []string
	Types     []string
	Statuses  []models.CommandStatus
	Priorities []int
}

// AlertFilter represents alert-specific filtering options
type AlertFilter struct {
	Filter
	TimeRangeFilter
	DeviceIDs    []string
	Types        []models.AlertType
	Severities   []models.AlertSeverity
	Acknowledged *bool
	Resolved     *bool
}

// AggregationRequest represents aggregation parameters
type AggregationRequest struct {
	DeviceIDs        []string
	Types            []string
	TimeRange        TimeRangeFilter
	GroupByInterval  time.Duration
	AggregationType  string // "avg", "min", "max", "sum", "count"
}

// AggregationResult represents aggregated measurement data
type AggregationResult struct {
	DeviceID    string                 `json:"device_id"`
	Type        string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Value       float64                `json:"value"`
	Count       int64                  `json:"count"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// BulkResult represents the result of bulk operations
type BulkResult struct {
	SuccessCount int
	FailureCount int
	Errors       []error
}

// DeviceRepository defines the interface for device data operations
type DeviceRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, device *models.Device) error
	GetByID(ctx context.Context, id string) (*models.Device, error)
	Update(ctx context.Context, device *models.Device) error
	Delete(ctx context.Context, id string) error
	
	// Bulk operations
	CreateBulk(ctx context.Context, devices []*models.Device) (*BulkResult, error)
	UpdateBulk(ctx context.Context, devices []*models.Device) (*BulkResult, error)
	
	// Query operations
	List(ctx context.Context, filter DeviceFilter) ([]*models.Device, error)
	Count(ctx context.Context, filter DeviceFilter) (int64, error)
	
	// Status operations
	UpdateStatus(ctx context.Context, deviceID string, status models.DeviceStatus) error
	UpdateLastSeen(ctx context.Context, deviceID string, timestamp time.Time) error
	
	// Search operations
	SearchByMetadata(ctx context.Context, metadata map[string]interface{}) ([]*models.Device, error)
	GetByCapability(ctx context.Context, capability string) ([]*models.Device, error)
	
	// Health operations
	GetOnlineDevices(ctx context.Context) ([]*models.Device, error)
	GetOfflineDevices(ctx context.Context, threshold time.Duration) ([]*models.Device, error)
}

// MeasurementRepository defines the interface for measurement data operations
type MeasurementRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, measurement *models.Measurement) error
	GetByID(ctx context.Context, id string) (*models.Measurement, error)
	Delete(ctx context.Context, id string) error
	
	// Bulk operations for high-throughput scenarios
	CreateBulk(ctx context.Context, measurements []*models.Measurement) (*BulkResult, error)
	CreateBatch(ctx context.Context, batch *models.MeasurementBatch) error
	
	// Query operations
	List(ctx context.Context, filter MeasurementFilter) ([]*models.Measurement, error)
	Count(ctx context.Context, filter MeasurementFilter) (int64, error)
	
	// Time-series specific operations
	GetByTimeRange(ctx context.Context, deviceID string, startTime, endTime time.Time) ([]*models.Measurement, error)
	GetLatest(ctx context.Context, deviceID string, measurementType string) (*models.Measurement, error)
	GetLatestByDevice(ctx context.Context, deviceID string, limit int) ([]*models.Measurement, error)
	
	// Aggregation operations
	Aggregate(ctx context.Context, req AggregationRequest) ([]*AggregationResult, error)
	GetStatistics(ctx context.Context, filter MeasurementFilter) (*models.MeasurementStats, error)
	
	// Cleanup operations
	DeleteOlderThan(ctx context.Context, threshold time.Time) (int64, error)
	DeleteByDevice(ctx context.Context, deviceID string) (int64, error)
}

// CommandRepository defines the interface for command data operations
type CommandRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, command *models.Command) error
	GetByID(ctx context.Context, id string) (*models.Command, error)
	GetByCommandID(ctx context.Context, commandID string) (*models.Command, error)
	Update(ctx context.Context, command *models.Command) error
	Delete(ctx context.Context, id string) error
	
	// Query operations
	List(ctx context.Context, filter CommandFilter) ([]*models.Command, error)
	Count(ctx context.Context, filter CommandFilter) (int64, error)
	
	// Command lifecycle operations
	GetPendingCommands(ctx context.Context, deviceID string) ([]*models.Command, error)
	GetExecutingCommands(ctx context.Context, deviceID string) ([]*models.Command, error)
	UpdateStatus(ctx context.Context, commandID string, status models.CommandStatus) error
	
	// Timeout and cleanup operations
	GetExpiredCommands(ctx context.Context) ([]*models.Command, error)
	MarkExpiredAsTimeout(ctx context.Context) (int64, error)
	DeleteCompletedOlderThan(ctx context.Context, threshold time.Time) (int64, error)
	
	// Statistics operations
	GetCommandStats(ctx context.Context, deviceID string, timeRange TimeRangeFilter) (map[models.CommandStatus]int64, error)
}

// AlertRepository defines the interface for alert data operations
type AlertRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, alert *models.Alert) error
	GetByID(ctx context.Context, id string) (*models.Alert, error)
	Update(ctx context.Context, alert *models.Alert) error
	Delete(ctx context.Context, id string) error
	
	// Query operations
	List(ctx context.Context, filter AlertFilter) ([]*models.Alert, error)
	Count(ctx context.Context, filter AlertFilter) (int64, error)
	
	// Alert lifecycle operations
	Acknowledge(ctx context.Context, alertID string, acknowledgedBy string) error
	Resolve(ctx context.Context, alertID string) error
	
	// Alert management operations
	GetUnacknowledged(ctx context.Context) ([]*models.Alert, error)
	GetUnresolved(ctx context.Context) ([]*models.Alert, error)
	GetCriticalAlerts(ctx context.Context) ([]*models.Alert, error)
	
	// Statistics operations
	GetAlertStats(ctx context.Context, timeRange TimeRangeFilter) (map[models.AlertSeverity]int64, error)
	GetAlertsByDevice(ctx context.Context, deviceID string, limit int) ([]*models.Alert, error)
	
	// Cleanup operations
	DeleteResolvedOlderThan(ctx context.Context, threshold time.Time) (int64, error)
}

// RepositoryManager defines the interface for managing all repositories
type RepositoryManager interface {
	Device() DeviceRepository
	Measurement() MeasurementRepository
	Command() CommandRepository
	Alert() AlertRepository
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(ctx context.Context, repos RepositoryManager) error) error
	
	// Health check
	HealthCheck(ctx context.Context) error
	
	// Close resources
	Close() error
}