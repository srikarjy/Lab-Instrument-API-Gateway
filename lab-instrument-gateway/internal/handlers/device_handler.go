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

// DeviceHandler handles device-related gRPC operations
type DeviceHandler struct {
	repos             repository.RepositoryManager
	connectionManager *device.ConnectionManager
	logger            *logger.Logger
}

// NewDeviceHandler creates a new device handler
func NewDeviceHandler(repos repository.RepositoryManager, connMgr *device.ConnectionManager, logger *logger.Logger) *DeviceHandler {
	return &DeviceHandler{
		repos:             repos,
		connectionManager: connMgr,
		logger:            logger,
	}
}

// RegisterDevice handles device registration requests
func (h *DeviceHandler) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.RegisterDeviceResponse, error) {
	// Input validation
	if err := h.validateRegisterDeviceRequest(req); err != nil {
		h.logger.WithError(err).WithField("device_id", req.DeviceId).Error("Invalid device registration request")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	h.logger.WithFields(map[string]interface{}{
		"device_id": req.DeviceId,
		"name":      req.Name,
		"type":      req.Type,
		"version":   req.Version,
	}).Info("Processing device registration")

	// Check if device already exists
	existingDevice, err := h.repos.Device().GetByID(ctx, req.DeviceId)
	if err != nil && err != repository.ErrNotFound {
		h.logger.WithError(err).WithField("device_id", req.DeviceId).Error("Failed to check existing device")
		return nil, status.Error(codes.Internal, "Failed to check device registration status")
	}

	var device *models.Device
	var isUpdate bool

	if existingDevice != nil {
		// Device exists - update it
		isUpdate = true
		device = existingDevice
		h.updateDeviceFromRequest(device, req)
		
		if err := h.repos.Device().Update(ctx, device); err != nil {
			h.logger.WithError(err).WithField("device_id", req.DeviceId).Error("Failed to update existing device")
			return nil, status.Error(codes.Internal, "Failed to update device registration")
		}
		
		h.logger.WithField("device_id", req.DeviceId).Info("Device registration updated")
	} else {
		// New device - create it
		device = h.createDeviceFromRequest(req)
		
		if err := h.repos.Device().Create(ctx, device); err != nil {
			h.logger.WithError(err).WithField("device_id", req.DeviceId).Error("Failed to create new device")
			return nil, status.Error(codes.Internal, "Failed to register new device")
		}
		
		h.logger.WithField("device_id", req.DeviceId).Info("New device registered")
	}

	// Generate session ID for the device connection
	sessionID := h.connectionManager.GenerateSessionID(req.DeviceId)

	// Create device session
	session := &models.DeviceSession{
		DeviceID:      req.DeviceId,
		SessionID:     sessionID,
		ConnectedAt:   time.Now(),
		LastHeartbeat: time.Now(),
		IsActive:      true,
		Metadata:      make(map[string]interface{}),
	}

	// Store connection metadata
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			session.Metadata[k] = v
		}
	}

	// Register the connection
	if err := h.connectionManager.RegisterConnection(ctx, session); err != nil {
		h.logger.WithError(err).WithField("device_id", req.DeviceId).Warn("Failed to register device connection")
		// Don't fail the registration if connection tracking fails
	}

	// Update device status to online
	device.SetStatus(models.DeviceStatusOnline)
	device.UpdateLastSeen()
	
	if err := h.repos.Device().UpdateStatus(ctx, device.ID, device.Status); err != nil {
		h.logger.WithError(err).WithField("device_id", req.DeviceId).Warn("Failed to update device status to online")
	}

	// Prepare response
	message := "Device registered successfully"
	if isUpdate {
		message = "Device registration updated successfully"
	}

	return &pb.RegisterDeviceResponse{
		Success:      true,
		Message:      message,
		SessionId:    sessionID,
		RegisteredAt: timestamppb.New(device.RegisteredAt),
	}, nil
}

// validateRegisterDeviceRequest validates the device registration request
func (h *DeviceHandler) validateRegisterDeviceRequest(req *pb.RegisterDeviceRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Validate device ID
	if strings.TrimSpace(req.DeviceId) == "" {
		return fmt.Errorf("device_id is required")
	}

	if len(req.DeviceId) > 255 {
		return fmt.Errorf("device_id too long (max 255 characters)")
	}

	// Validate device name
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("device name is required")
	}

	if len(req.Name) > 255 {
		return fmt.Errorf("device name too long (max 255 characters)")
	}

	// Validate device type
	if strings.TrimSpace(req.Type) == "" {
		return fmt.Errorf("device type is required")
	}

	validTypes := map[string]bool{
		"sensor":        true,
		"actuator":      true,
		"analyzer":      true,
		"controller":    true,
		"spectrometer":  true,
		"chromatograph": true,
		"microscope":    true,
		"balance":       true,
		"ph_meter":      true,
		"thermometer":   true,
		"other":         true,
	}

	if !validTypes[strings.ToLower(req.Type)] {
		return fmt.Errorf("invalid device type: %s", req.Type)
	}

	// Validate firmware version
	if strings.TrimSpace(req.Version) == "" {
		return fmt.Errorf("firmware version is required")
	}

	if len(req.Version) > 50 {
		return fmt.Errorf("firmware version too long (max 50 characters)")
	}

	// Validate capabilities
	if len(req.Capabilities) == 0 {
		return fmt.Errorf("at least one capability is required")
	}

	validCapabilities := map[string]bool{
		"temperature":    true,
		"humidity":       true,
		"pressure":       true,
		"ph":            true,
		"conductivity":   true,
		"turbidity":      true,
		"dissolved_oxygen": true,
		"flow_rate":      true,
		"level":          true,
		"weight":         true,
		"vibration":      true,
		"acceleration":   true,
		"voltage":        true,
		"current":        true,
		"power":          true,
		"frequency":      true,
		"spectrum":       true,
		"image":          true,
		"control":        true,
		"calibration":    true,
	}

	for _, capability := range req.Capabilities {
		if !validCapabilities[strings.ToLower(capability)] {
			return fmt.Errorf("invalid capability: %s", capability)
		}
	}

	// Validate metadata
	if req.Metadata != nil {
		for key, value := range req.Metadata {
			if len(key) > 100 {
				return fmt.Errorf("metadata key too long: %s (max 100 characters)", key)
			}
			if len(value) > 1000 {
				return fmt.Errorf("metadata value too long for key %s (max 1000 characters)", key)
			}
		}
	}

	return nil
}

// createDeviceFromRequest creates a new device model from the registration request
func (h *DeviceHandler) createDeviceFromRequest(req *pb.RegisterDeviceRequest) *models.Device {
	now := time.Now()
	
	// Convert metadata
	metadata := make(map[string]interface{})
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = v
		}
	}

	// Normalize capabilities
	capabilities := make([]string, len(req.Capabilities))
	for i, cap := range req.Capabilities {
		capabilities[i] = strings.ToLower(strings.TrimSpace(cap))
	}

	return &models.Device{
		ID:           req.DeviceId,
		Name:         strings.TrimSpace(req.Name),
		Type:         strings.ToLower(strings.TrimSpace(req.Type)),
		Version:      strings.TrimSpace(req.Version),
		Status:       models.DeviceStatusConnecting,
		Metadata:     metadata,
		Capabilities: capabilities,
		RegisteredAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// updateDeviceFromRequest updates an existing device model from the registration request
func (h *DeviceHandler) updateDeviceFromRequest(device *models.Device, req *pb.RegisterDeviceRequest) {
	now := time.Now()
	
	// Update basic fields
	device.Name = strings.TrimSpace(req.Name)
	device.Type = strings.ToLower(strings.TrimSpace(req.Type))
	device.Version = strings.TrimSpace(req.Version)
	device.UpdatedAt = now

	// Update metadata
	if req.Metadata != nil {
		if device.Metadata == nil {
			device.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			device.Metadata[k] = v
		}
	}

	// Update capabilities
	if len(req.Capabilities) > 0 {
		capabilities := make([]string, len(req.Capabilities))
		for i, cap := range req.Capabilities {
			capabilities[i] = strings.ToLower(strings.TrimSpace(cap))
		}
		device.Capabilities = capabilities
	}

	// Update status if device was offline
	if device.Status == models.DeviceStatusOffline || device.Status == models.DeviceStatusError {
		device.Status = models.DeviceStatusConnecting
	}
}