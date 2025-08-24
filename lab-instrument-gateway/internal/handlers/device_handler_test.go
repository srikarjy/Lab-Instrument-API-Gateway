package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yourorg/lab-gateway/internal/device"
	"github.com/yourorg/lab-gateway/pkg/logger"
	"github.com/yourorg/lab-gateway/pkg/models"
	"github.com/yourorg/lab-gateway/pkg/repository"
	pb "github.com/yourorg/lab-gateway/proto"
)

// MockRepositoryManager is a mock implementation of RepositoryManager
type MockRepositoryManager struct {
	mock.Mock
}

func (m *MockRepositoryManager) Device() repository.DeviceRepository {
	args := m.Called()
	return args.Get(0).(repository.DeviceRepository)
}

func (m *MockRepositoryManager) Measurement() repository.MeasurementRepository {
	args := m.Called()
	return args.Get(0).(repository.MeasurementRepository)
}

func (m *MockRepositoryManager) Command() repository.CommandRepository {
	args := m.Called()
	return args.Get(0).(repository.CommandRepository)
}

func (m *MockRepositoryManager) Alert() repository.AlertRepository {
	args := m.Called()
	return args.Get(0).(repository.AlertRepository)
}

func (m *MockRepositoryManager) WithTransaction(ctx context.Context, fn func(ctx context.Context, repos repository.RepositoryManager) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockRepositoryManager) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepositoryManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockDeviceRepository is a mock implementation of DeviceRepository
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) Create(ctx context.Context, device *models.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetByID(ctx context.Context, id string) (*models.Device, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) Update(ctx context.Context, device *models.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDeviceRepository) List(ctx context.Context, filter repository.DeviceFilter) ([]*models.Device, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Device), args.Error(1)
}



func (m *MockDeviceRepository) UpdateStatus(ctx context.Context, id string, status models.DeviceStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDeviceRepository) Count(ctx context.Context, filter repository.DeviceFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDeviceRepository) CreateBulk(ctx context.Context, devices []*models.Device) (*repository.BulkResult, error) {
	args := m.Called(ctx, devices)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.BulkResult), args.Error(1)
}

func (m *MockDeviceRepository) UpdateBulk(ctx context.Context, devices []*models.Device) (*repository.BulkResult, error) {
	args := m.Called(ctx, devices)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.BulkResult), args.Error(1)
}

func (m *MockDeviceRepository) UpdateLastSeen(ctx context.Context, deviceID string, timestamp time.Time) error {
	args := m.Called(ctx, deviceID, timestamp)
	return args.Error(0)
}

func (m *MockDeviceRepository) SearchByMetadata(ctx context.Context, metadata map[string]interface{}) ([]*models.Device, error) {
	args := m.Called(ctx, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByCapability(ctx context.Context, capability string) ([]*models.Device, error) {
	args := m.Called(ctx, capability)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetOnlineDevices(ctx context.Context) ([]*models.Device, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetOfflineDevices(ctx context.Context, threshold time.Duration) ([]*models.Device, error) {
	args := m.Called(ctx, threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Device), args.Error(1)
}



// MockAlertRepository is a mock implementation of AlertRepository
type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertRepository) GetByID(ctx context.Context, id string) (*models.Alert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Alert), args.Error(1)
}

func (m *MockAlertRepository) Update(ctx context.Context, alert *models.Alert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockAlertRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAlertRepository) List(ctx context.Context, filter repository.AlertFilter) ([]*models.Alert, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Alert), args.Error(1)
}

func (m *MockAlertRepository) Count(ctx context.Context, filter repository.AlertFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAlertRepository) Acknowledge(ctx context.Context, alertID string, acknowledgedBy string) error {
	args := m.Called(ctx, alertID, acknowledgedBy)
	return args.Error(0)
}

func (m *MockAlertRepository) Resolve(ctx context.Context, alertID string) error {
	args := m.Called(ctx, alertID)
	return args.Error(0)
}

func (m *MockAlertRepository) GetUnacknowledged(ctx context.Context) ([]*models.Alert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Alert), args.Error(1)
}

func (m *MockAlertRepository) GetUnresolved(ctx context.Context) ([]*models.Alert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Alert), args.Error(1)
}

func (m *MockAlertRepository) GetCriticalAlerts(ctx context.Context) ([]*models.Alert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Alert), args.Error(1)
}

func (m *MockAlertRepository) GetAlertStats(ctx context.Context, timeRange repository.TimeRangeFilter) (map[models.AlertSeverity]int64, error) {
	args := m.Called(ctx, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[models.AlertSeverity]int64), args.Error(1)
}

func (m *MockAlertRepository) GetAlertsByDevice(ctx context.Context, deviceID string, limit int) ([]*models.Alert, error) {
	args := m.Called(ctx, deviceID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Alert), args.Error(1)
}

func (m *MockAlertRepository) DeleteResolvedOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	args := m.Called(ctx, threshold)
	return args.Get(0).(int64), args.Error(1)
}

// MockMeasurementRepository is a mock implementation of MeasurementRepository
type MockMeasurementRepository struct {
	mock.Mock
}

func (m *MockMeasurementRepository) Create(ctx context.Context, measurement *models.Measurement) error {
	args := m.Called(ctx, measurement)
	return args.Error(0)
}

func (m *MockMeasurementRepository) GetByID(ctx context.Context, id string) (*models.Measurement, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Measurement), args.Error(1)
}

func (m *MockMeasurementRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMeasurementRepository) CreateBulk(ctx context.Context, measurements []*models.Measurement) (*repository.BulkResult, error) {
	args := m.Called(ctx, measurements)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.BulkResult), args.Error(1)
}

func (m *MockMeasurementRepository) CreateBatch(ctx context.Context, batch *models.MeasurementBatch) error {
	args := m.Called(ctx, batch)
	return args.Error(0)
}

func (m *MockMeasurementRepository) List(ctx context.Context, filter repository.MeasurementFilter) ([]*models.Measurement, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Measurement), args.Error(1)
}

func (m *MockMeasurementRepository) Count(ctx context.Context, filter repository.MeasurementFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMeasurementRepository) GetByTimeRange(ctx context.Context, deviceID string, startTime, endTime time.Time) ([]*models.Measurement, error) {
	args := m.Called(ctx, deviceID, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Measurement), args.Error(1)
}

func (m *MockMeasurementRepository) GetLatest(ctx context.Context, deviceID string, measurementType string) (*models.Measurement, error) {
	args := m.Called(ctx, deviceID, measurementType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Measurement), args.Error(1)
}

func (m *MockMeasurementRepository) GetLatestByDevice(ctx context.Context, deviceID string, limit int) ([]*models.Measurement, error) {
	args := m.Called(ctx, deviceID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Measurement), args.Error(1)
}

func (m *MockMeasurementRepository) Aggregate(ctx context.Context, req repository.AggregationRequest) ([]*repository.AggregationResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.AggregationResult), args.Error(1)
}

func (m *MockMeasurementRepository) GetStatistics(ctx context.Context, filter repository.MeasurementFilter) (*models.MeasurementStats, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MeasurementStats), args.Error(1)
}

func (m *MockMeasurementRepository) DeleteOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	args := m.Called(ctx, threshold)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMeasurementRepository) DeleteByDevice(ctx context.Context, deviceID string) (int64, error) {
	args := m.Called(ctx, deviceID)
	return args.Get(0).(int64), args.Error(1)
}

// MockCommandRepository is a mock implementation of CommandRepository
type MockCommandRepository struct {
	mock.Mock
}

func (m *MockCommandRepository) Create(ctx context.Context, command *models.Command) error {
	args := m.Called(ctx, command)
	return args.Error(0)
}

func (m *MockCommandRepository) GetByID(ctx context.Context, id string) (*models.Command, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Command), args.Error(1)
}

func (m *MockCommandRepository) GetByCommandID(ctx context.Context, commandID string) (*models.Command, error) {
	args := m.Called(ctx, commandID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Command), args.Error(1)
}

func (m *MockCommandRepository) Update(ctx context.Context, command *models.Command) error {
	args := m.Called(ctx, command)
	return args.Error(0)
}

func (m *MockCommandRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCommandRepository) List(ctx context.Context, filter repository.CommandFilter) ([]*models.Command, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Command), args.Error(1)
}

func (m *MockCommandRepository) Count(ctx context.Context, filter repository.CommandFilter) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommandRepository) GetPendingCommands(ctx context.Context, deviceID string) ([]*models.Command, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Command), args.Error(1)
}

func (m *MockCommandRepository) GetExecutingCommands(ctx context.Context, deviceID string) ([]*models.Command, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Command), args.Error(1)
}

func (m *MockCommandRepository) UpdateStatus(ctx context.Context, commandID string, status models.CommandStatus) error {
	args := m.Called(ctx, commandID, status)
	return args.Error(0)
}

func (m *MockCommandRepository) GetExpiredCommands(ctx context.Context) ([]*models.Command, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Command), args.Error(1)
}

func (m *MockCommandRepository) MarkExpiredAsTimeout(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommandRepository) DeleteCompletedOlderThan(ctx context.Context, threshold time.Time) (int64, error) {
	args := m.Called(ctx, threshold)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommandRepository) GetCommandStats(ctx context.Context, deviceID string, timeRange repository.TimeRangeFilter) (map[models.CommandStatus]int64, error) {
	args := m.Called(ctx, deviceID, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[models.CommandStatus]int64), args.Error(1)
}

func TestDeviceHandler_RegisterDevice(t *testing.T) {
	// Setup
	mockRepos := &MockRepositoryManager{}
	mockDeviceRepo := &MockDeviceRepository{}
	logger := logger.NewDefaultLogger()
	connMgr := device.NewConnectionManager(logger)
	
	handler := NewDeviceHandler(mockRepos, connMgr, logger)
	
	tests := []struct {
		name           string
		request        *pb.RegisterDeviceRequest
		setupMocks     func()
		expectedError  bool
		expectedStatus bool
	}{
		{
			name: "successful new device registration",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device-1",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{"temperature", "humidity"},
				Metadata: map[string]string{
					"location": "lab-1",
				},
			},
			setupMocks: func() {
				mockRepos.On("Device").Return(mockDeviceRepo)
				mockDeviceRepo.On("GetByID", mock.Anything, "test-device-1").Return(nil, repository.ErrNotFound)
				mockDeviceRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Device")).Return(nil)
				mockDeviceRepo.On("UpdateStatus", mock.Anything, "test-device-1", models.DeviceStatusOnline).Return(nil)
			},
			expectedError:  false,
			expectedStatus: true,
		},
		{
			name: "successful device update",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "existing-device",
				Name:         "Updated Device",
				Type:         "sensor",
				Version:      "2.0.0",
				Capabilities: []string{"temperature"},
			},
			setupMocks: func() {
				existingDevice := &models.Device{
					ID:           "existing-device",
					Name:         "Old Device",
					Type:         "sensor",
					Version:      "1.0.0",
					Status:       models.DeviceStatusOffline,
					RegisteredAt: time.Now().Add(-time.Hour),
					CreatedAt:    time.Now().Add(-time.Hour),
					UpdatedAt:    time.Now().Add(-time.Hour),
				}
				
				mockRepos.On("Device").Return(mockDeviceRepo)
				mockDeviceRepo.On("GetByID", mock.Anything, "existing-device").Return(existingDevice, nil)
				mockDeviceRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Device")).Return(nil)
				mockDeviceRepo.On("UpdateStatus", mock.Anything, "existing-device", models.DeviceStatusOnline).Return(nil)
			},
			expectedError:  false,
			expectedStatus: true,
		},
		{
			name: "invalid device ID",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{"temperature"},
			},
			setupMocks:     func() {},
			expectedError:  true,
			expectedStatus: false,
		},
		{
			name: "invalid device type",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device",
				Name:         "Test Device",
				Type:         "invalid-type",
				Version:      "1.0.0",
				Capabilities: []string{"temperature"},
			},
			setupMocks:     func() {},
			expectedError:  true,
			expectedStatus: false,
		},
		{
			name: "missing capabilities",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{},
			},
			setupMocks:     func() {},
			expectedError:  true,
			expectedStatus: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockRepos.ExpectedCalls = nil
			mockDeviceRepo.ExpectedCalls = nil
			
			// Setup mocks
			tt.setupMocks()
			
			// Execute
			ctx := context.Background()
			resp, err := handler.RegisterDevice(ctx, tt.request)
			
			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.expectedStatus, resp.Success)
				assert.NotEmpty(t, resp.SessionId)
				assert.NotNil(t, resp.RegisteredAt)
			}
			
			// Verify mocks
			mockRepos.AssertExpectations(t)
			mockDeviceRepo.AssertExpectations(t)
		})
	}
}

func TestDeviceHandler_validateRegisterDeviceRequest(t *testing.T) {
	handler := &DeviceHandler{}
	
	tests := []struct {
		name        string
		request     *pb.RegisterDeviceRequest
		expectError bool
	}{
		{
			name: "valid request",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device-1",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{"temperature"},
			},
			expectError: false,
		},
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
		},
		{
			name: "empty device ID",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{"temperature"},
			},
			expectError: true,
		},
		{
			name: "empty device name",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device",
				Name:         "",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{"temperature"},
			},
			expectError: true,
		},
		{
			name: "invalid device type",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device",
				Name:         "Test Device",
				Type:         "invalid",
				Version:      "1.0.0",
				Capabilities: []string{"temperature"},
			},
			expectError: true,
		},
		{
			name: "empty version",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "",
				Capabilities: []string{"temperature"},
			},
			expectError: true,
		},
		{
			name: "no capabilities",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{},
			},
			expectError: true,
		},
		{
			name: "invalid capability",
			request: &pb.RegisterDeviceRequest{
				DeviceId:     "test-device",
				Name:         "Test Device",
				Type:         "sensor",
				Version:      "1.0.0",
				Capabilities: []string{"invalid-capability"},
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateRegisterDeviceRequest(tt.request)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeviceHandler_createDeviceFromRequest(t *testing.T) {
	handler := &DeviceHandler{}
	
	request := &pb.RegisterDeviceRequest{
		DeviceId:     "test-device-1",
		Name:         "Test Device",
		Type:         "Sensor",
		Version:      "1.0.0",
		Capabilities: []string{"Temperature", "Humidity"},
		Metadata: map[string]string{
			"location": "lab-1",
			"owner":    "test-user",
		},
	}
	
	device := handler.createDeviceFromRequest(request)
	
	assert.Equal(t, "test-device-1", device.ID)
	assert.Equal(t, "Test Device", device.Name)
	assert.Equal(t, "sensor", device.Type) // Should be normalized to lowercase
	assert.Equal(t, "1.0.0", device.Version)
	assert.Equal(t, models.DeviceStatusConnecting, device.Status)
	assert.Equal(t, []string{"temperature", "humidity"}, []string(device.Capabilities)) // Should be normalized
	assert.Equal(t, "lab-1", device.Metadata["location"])
	assert.Equal(t, "test-user", device.Metadata["owner"])
	assert.False(t, device.RegisteredAt.IsZero())
	assert.False(t, device.CreatedAt.IsZero())
	assert.False(t, device.UpdatedAt.IsZero())
}