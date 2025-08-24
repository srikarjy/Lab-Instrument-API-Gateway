package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yourorg/lab-gateway/internal/device"
	"github.com/yourorg/lab-gateway/pkg/logger"
	"github.com/yourorg/lab-gateway/pkg/models"
	"github.com/yourorg/lab-gateway/pkg/repository"
	pb "github.com/yourorg/lab-gateway/proto"
)

// DeviceStatusHandler handles device status-related gRPC operations
type DeviceStatusHandler struct {
	repos             repository.RepositoryManager
	connectionManager *device.ConnectionManager
	logger            *logger.Logger
}

// NewDeviceStatusHandler creates a new device status handler
func NewDeviceStatusHandler(repos repository.RepositoryManager, connMgr *device.ConnectionManager, logger *logger.Logger) *DeviceStatusHandler {
	return &DeviceStatusHandler{
		repos:             repos,
		connectionManager: connMgr,
		logger:            logger,
	}
}

// GetDeviceStatus handles device status requests
func (h *DeviceStatusHandler) GetDeviceStatus(ctx context.Context, req *pb.GetDeviceStatusRequest) (*pb.GetDeviceStatusResponse, error) {
	// Input validation
	if err := h.validateGetDeviceStatusRequest(req); err != nil {
		h.logger.WithError(err).WithField("device_id", req.DeviceId).Error("Invalid device status request")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	h.logger.WithField("device_id", req.DeviceId).Debug("Processing device status request")

	// Get device information
	device, err := h.repos.Device().GetByID(ctx, req.DeviceId)
	if err != nil {
		if err == repository.ErrNotFound {
			h.logger.WithField("device_id", req.DeviceId).Warn("Device not found")
			return nil, status.Error(codes.NotFound, "Device not found")
		}
		h.logger.WithError(err).WithField("device_id", req.DeviceId).Error("Failed to get device")
		return nil, status.Error(codes.Internal, "Failed to retrieve device information")
	}

	// Get connection status
	connectionStatus := h.connectionManager.GetConnectionStatus(req.DeviceId)
	
	// Update device status based on connection
	actualStatus := h.determineActualDeviceStatus(device, connectionStatus)
	
	// Get device alerts for health assessment
	alertFilter := repository.AlertFilter{
		Filter: repository.Filter{
			Limit: 10, // Get recent alerts
		},
		DeviceIDs:    []string{req.DeviceId},
		Severities:   []models.AlertSeverity{models.AlertSeverityCritical, models.AlertSeverityError},
		Resolved:     &[]bool{false}[0], // Get unresolved alerts
	}
	
	alerts, err := h.repos.Alert().List(ctx, alertFilter)
	if err != nil {
		h.logger.WithError(err).WithField("device_id", req.DeviceId).Warn("Failed to get device alerts")
		// Don't fail the request if alerts can't be retrieved
	}

	// Aggregate health information
	healthStatus := h.aggregateHealthStatus(device, connectionStatus, alerts)

	// Get active capabilities from connection
	activeCapabilities := h.getActiveCapabilities(device, connectionStatus)

	// Prepare metadata with performance metrics
	metadata := h.prepareStatusMetadata(device, connectionStatus)

	// Convert last seen timestamp
	var lastSeenProto *timestamppb.Timestamp
	if device.LastSeen != nil {
		lastSeenProto = timestamppb.New(*device.LastSeen)
	}

	return &pb.GetDeviceStatusResponse{
		DeviceId:           req.DeviceId,
		Status:             h.convertDeviceStatusToProto(actualStatus),
		LastSeen:           lastSeenProto,
		Metadata:           metadata,
		ActiveCapabilities: activeCapabilities,
		Health:             healthStatus,
	}, nil
}

// validateGetDeviceStatusRequest validates the device status request
func (h *DeviceStatusHandler) validateGetDeviceStatusRequest(req *pb.GetDeviceStatusRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if strings.TrimSpace(req.DeviceId) == "" {
		return fmt.Errorf("device_id is required")
	}

	if len(req.DeviceId) > 255 {
		return fmt.Errorf("device_id too long (max 255 characters)")
	}

	return nil
}

// determineActualDeviceStatus determines the actual device status based on database and connection info
func (h *DeviceStatusHandler) determineActualDeviceStatus(device *models.Device, connStatus *device.ConnectionStatus) models.DeviceStatus {
	// If no connection info, use database status
	if connStatus == nil {
		// Check if device should be considered offline based on last seen
		if device.LastSeen != nil {
			offlineThreshold := 5 * time.Minute
			if time.Since(*device.LastSeen) > offlineThreshold {
				return models.DeviceStatusOffline
			}
		}
		return device.Status
	}

	// Use connection status to determine actual status
	if connStatus.IsConnected {
		if connStatus.IsHealthy {
			return models.DeviceStatusOnline
		} else {
			return models.DeviceStatusError
		}
	}

	// Check if recently disconnected
	disconnectThreshold := 30 * time.Second
	if time.Since(connStatus.LastSeen) < disconnectThreshold {
		return models.DeviceStatusConnecting
	}

	return models.DeviceStatusOffline
}

// aggregateHealthStatus aggregates health information from multiple sources
func (h *DeviceStatusHandler) aggregateHealthStatus(device *models.Device, connStatus *device.ConnectionStatus, alerts []*models.Alert) pb.HealthStatus {
	// Start with serving status
	healthStatus := pb.HealthStatus_HEALTH_SERVING

	// Check device status
	if device.Status == models.DeviceStatusError || device.Status == models.DeviceStatusOffline {
		healthStatus = pb.HealthStatus_HEALTH_NOT_SERVING
	}

	// Check connection health
	if connStatus != nil && !connStatus.IsHealthy {
		healthStatus = pb.HealthStatus_HEALTH_NOT_SERVING
	}

	// Check for critical alerts
	for _, alert := range alerts {
		if alert.IsCritical() && !alert.IsResolved() {
			healthStatus = pb.HealthStatus_HEALTH_NOT_SERVING
			break
		}
	}

	// Check connection timeout
	if connStatus != nil {
		heartbeatTimeout := 2 * time.Minute
		if time.Since(connStatus.LastHeartbeat) > heartbeatTimeout {
			healthStatus = pb.HealthStatus_HEALTH_NOT_SERVING
		}
	}

	return healthStatus
}

// getActiveCapabilities returns the currently active capabilities
func (h *DeviceStatusHandler) getActiveCapabilities(device *models.Device, connStatus *device.ConnectionStatus) []string {
	// If device is not connected or healthy, return empty capabilities
	if connStatus == nil || !connStatus.IsConnected || !connStatus.IsHealthy {
		return []string{}
	}

	// Return all device capabilities if connected and healthy
	capabilities := make([]string, len(device.Capabilities))
	copy(capabilities, device.Capabilities)
	
	return capabilities
}

// prepareStatusMetadata prepares metadata with performance metrics and connection info
func (h *DeviceStatusHandler) prepareStatusMetadata(device *models.Device, connStatus *device.ConnectionStatus) map[string]string {
	metadata := make(map[string]string)

	// Add device metadata
	if device.Metadata != nil {
		for k, v := range device.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			} else {
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Add connection information
	if connStatus != nil {
		metadata["connection_id"] = connStatus.ConnectionID
		metadata["session_id"] = connStatus.SessionID
		metadata["connected_at"] = connStatus.ConnectedAt.Format(time.RFC3339)
		metadata["last_heartbeat"] = connStatus.LastHeartbeat.Format(time.RFC3339)
		metadata["is_connected"] = fmt.Sprintf("%t", connStatus.IsConnected)
		metadata["is_healthy"] = fmt.Sprintf("%t", connStatus.IsHealthy)

		// Add performance metrics if available
		if connStatus.Metrics != nil {
			for k, v := range connStatus.Metrics {
				metadata[fmt.Sprintf("metric_%s", k)] = fmt.Sprintf("%v", v)
			}
		}

		// Add connection statistics
		metadata["messages_sent"] = fmt.Sprintf("%d", connStatus.MessagesSent)
		metadata["messages_received"] = fmt.Sprintf("%d", connStatus.MessagesReceived)
		metadata["bytes_sent"] = fmt.Sprintf("%d", connStatus.BytesSent)
		metadata["bytes_received"] = fmt.Sprintf("%d", connStatus.BytesReceived)
		
		if connStatus.LastError != nil {
			metadata["last_error"] = *connStatus.LastError
			metadata["last_error_at"] = connStatus.LastErrorAt.Format(time.RFC3339)
		}
	}

	// Add device registration info
	metadata["device_type"] = device.Type
	metadata["firmware_version"] = device.Version
	metadata["registered_at"] = device.RegisteredAt.Format(time.RFC3339)
	metadata["updated_at"] = device.UpdatedAt.Format(time.RFC3339)

	return metadata
}

// convertDeviceStatusToProto converts internal device status to protobuf enum
func (h *DeviceStatusHandler) convertDeviceStatusToProto(status models.DeviceStatus) pb.DeviceStatus {
	switch status {
	case models.DeviceStatusOnline:
		return pb.DeviceStatus_DEVICE_STATUS_ONLINE
	case models.DeviceStatusOffline:
		return pb.DeviceStatus_DEVICE_STATUS_OFFLINE
	case models.DeviceStatusError:
		return pb.DeviceStatus_DEVICE_STATUS_ERROR
	case models.DeviceStatusMaintenance:
		return pb.DeviceStatus_DEVICE_STATUS_MAINTENANCE
	case models.DeviceStatusConnecting:
		return pb.DeviceStatus_DEVICE_STATUS_CONNECTING
	default:
		return pb.DeviceStatus_DEVICE_STATUS_UNKNOWN
	}
}