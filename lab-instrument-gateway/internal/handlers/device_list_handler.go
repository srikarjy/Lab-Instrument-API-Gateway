package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yourorg/lab-gateway/pkg/logger"
	"github.com/yourorg/lab-gateway/pkg/models"
	"github.com/yourorg/lab-gateway/pkg/repository"
	pb "github.com/yourorg/lab-gateway/proto"
)

// DeviceListHandler handles device listing gRPC operations
type DeviceListHandler struct {
	repos  repository.RepositoryManager
	logger *logger.Logger
}

// NewDeviceListHandler creates a new device list handler
func NewDeviceListHandler(repos repository.RepositoryManager, logger *logger.Logger) *DeviceListHandler {
	return &DeviceListHandler{
		repos:  repos,
		logger: logger,
	}
}

// PageToken represents the pagination token structure
type PageToken struct {
	Offset int    `json:"offset"`
	SortBy string `json:"sort_by"`
	Order  string `json:"order"`
}

// ListDevices handles device listing requests with pagination, filtering, and sorting
func (h *DeviceListHandler) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	// Input validation
	if err := h.validateListDevicesRequest(req); err != nil {
		h.logger.WithError(err).Error("Invalid device list request")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	h.logger.WithFields(map[string]interface{}{
		"page_size":  req.PageSize,
		"page_token": req.PageToken,
		"sort_by":    req.SortBy,
		"ascending":  req.Ascending,
	}).Debug("Processing device list request")

	// Parse pagination token
	offset := 0
	if req.PageToken != "" {
		parsedOffset, err := h.parsePageToken(req.PageToken)
		if err != nil {
			h.logger.WithError(err).Warn("Invalid page token")
			return nil, status.Error(codes.InvalidArgument, "Invalid page token")
		}
		offset = parsedOffset
	}

	// Build device filter
	deviceFilter := h.buildDeviceFilter(req, offset)

	// Get devices
	devices, err := h.repos.Device().List(ctx, deviceFilter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list devices")
		return nil, status.Error(codes.Internal, "Failed to retrieve devices")
	}

	// Get total count
	totalCount, err := h.repos.Device().Count(ctx, deviceFilter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to count devices")
		return nil, status.Error(codes.Internal, "Failed to count devices")
	}

	// Convert devices to protobuf format
	deviceInfos := make([]*pb.DeviceInfo, len(devices))
	for i, device := range devices {
		deviceInfos[i] = h.convertDeviceToProto(device)
	}

	// Generate next page token
	var nextPageToken string
	if offset+len(devices) < int(totalCount) {
		nextPageToken = h.generatePageToken(offset+len(devices), deviceFilter.SortBy, deviceFilter.Order)
	}

	h.logger.WithFields(map[string]interface{}{
		"returned_count": len(devices),
		"total_count":    totalCount,
		"has_next_page":  nextPageToken != "",
	}).Debug("Device list request completed")

	return &pb.ListDevicesResponse{
		Devices:       deviceInfos,
		NextPageToken: nextPageToken,
		TotalCount:    int32(totalCount),
	}, nil
}

// validateListDevicesRequest validates the device list request
func (h *DeviceListHandler) validateListDevicesRequest(req *pb.ListDevicesRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Validate page size
	if req.PageSize <= 0 {
		req.PageSize = 50 // Default page size
	}

	if req.PageSize > 1000 {
		return fmt.Errorf("page_size too large (max 1000)")
	}

	// Validate sort field
	if req.SortBy != "" {
		validSortFields := map[string]bool{
			"id":            true,
			"name":          true,
			"type":          true,
			"status":        true,
			"last_seen":     true,
			"registered_at": true,
			"updated_at":    true,
		}

		if !validSortFields[req.SortBy] {
			return fmt.Errorf("invalid sort field: %s", req.SortBy)
		}
	}

	// Validate filter
	if req.Filter != nil {
		if err := h.validateDeviceFilter(req.Filter); err != nil {
			return fmt.Errorf("invalid filter: %w", err)
		}
	}

	return nil
}

// validateDeviceFilter validates the device filter
func (h *DeviceListHandler) validateDeviceFilter(filter *pb.DeviceFilter) error {
	// Validate status filters
	for _, status := range filter.Status {
		if !h.isValidDeviceStatus(status) {
			return fmt.Errorf("invalid device status: %v", status)
		}
	}

	// Validate types
	for _, deviceType := range filter.Types {
		if strings.TrimSpace(deviceType) == "" {
			return fmt.Errorf("device type cannot be empty")
		}
	}

	// Validate time range
	if filter.LastSeenAfter != nil && filter.LastSeenBefore != nil {
		if filter.LastSeenAfter.AsTime().After(filter.LastSeenBefore.AsTime()) {
			return fmt.Errorf("last_seen_after cannot be after last_seen_before")
		}
	}

	// Validate metadata filters
	if filter.MetadataFilters != nil {
		for key, value := range filter.MetadataFilters {
			if len(key) > 100 {
				return fmt.Errorf("metadata filter key too long: %s", key)
			}
			if len(value) > 1000 {
				return fmt.Errorf("metadata filter value too long for key %s", key)
			}
		}
	}

	return nil
}

// buildDeviceFilter builds the repository filter from the protobuf request
func (h *DeviceListHandler) buildDeviceFilter(req *pb.ListDevicesRequest, offset int) repository.DeviceFilter {
	filter := repository.DeviceFilter{
		Filter: repository.Filter{
			Limit:  int(req.PageSize),
			Offset: offset,
			SortBy: req.SortBy,
			Order:  "ASC",
		},
	}

	// Set sort order
	if !req.Ascending {
		filter.Order = "DESC"
	}

	// Set default sort field
	if filter.SortBy == "" {
		filter.SortBy = "updated_at"
		filter.Order = "DESC"
	}

	// Apply filters if provided
	if req.Filter != nil {
		// Status filters
		if len(req.Filter.Status) > 0 {
			filter.Statuses = make([]models.DeviceStatus, len(req.Filter.Status))
			for i, status := range req.Filter.Status {
				filter.Statuses[i] = h.convertProtoToDeviceStatus(status)
			}
		}

		// Type filters
		if len(req.Filter.Types) > 0 {
			filter.Types = make([]string, len(req.Filter.Types))
			for i, deviceType := range req.Filter.Types {
				filter.Types[i] = strings.ToLower(strings.TrimSpace(deviceType))
			}
		}

		// Time range filters
		if req.Filter.LastSeenAfter != nil {
			lastSeenAfter := req.Filter.LastSeenAfter.AsTime()
			filter.LastSeenAfter = &lastSeenAfter
		}

		if req.Filter.LastSeenBefore != nil {
			lastSeenBefore := req.Filter.LastSeenBefore.AsTime()
			filter.LastSeenBefore = &lastSeenBefore
		}

		// Metadata filters
		if req.Filter.MetadataFilters != nil {
			filter.MetadataFilters = make(map[string]interface{})
			for k, v := range req.Filter.MetadataFilters {
				filter.MetadataFilters[k] = v
			}
		}
	}

	return filter
}

// parsePageToken parses the pagination token
func (h *DeviceListHandler) parsePageToken(token string) (int, error) {
	// Decode base64 token
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid token encoding: %w", err)
	}

	// Parse JSON token
	var pageToken PageToken
	if err := json.Unmarshal(decoded, &pageToken); err != nil {
		return 0, fmt.Errorf("invalid token format: %w", err)
	}

	return pageToken.Offset, nil
}

// generatePageToken generates a pagination token
func (h *DeviceListHandler) generatePageToken(offset int, sortBy, order string) string {
	token := PageToken{
		Offset: offset,
		SortBy: sortBy,
		Order:  order,
	}

	tokenBytes, _ := json.Marshal(token)
	return base64.URLEncoding.EncodeToString(tokenBytes)
}

// convertDeviceToProto converts a device model to protobuf format
func (h *DeviceListHandler) convertDeviceToProto(device *models.Device) *pb.DeviceInfo {
	// Convert metadata
	metadata := make(map[string]string)
	if device.Metadata != nil {
		for k, v := range device.Metadata {
			if str, ok := v.(string); ok {
				metadata[k] = str
			} else {
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Convert capabilities
	capabilities := make([]string, len(device.Capabilities))
	copy(capabilities, device.Capabilities)

	// Convert last seen timestamp
	var lastSeenProto *timestamppb.Timestamp
	if device.LastSeen != nil {
		lastSeenProto = timestamppb.New(*device.LastSeen)
	}

	return &pb.DeviceInfo{
		DeviceId:     device.ID,
		Name:         device.Name,
		Type:         device.Type,
		Version:      device.Version,
		Status:       h.convertDeviceStatusToProto(device.Status),
		LastSeen:     lastSeenProto,
		RegisteredAt: timestamppb.New(device.RegisteredAt),
		Metadata:     metadata,
		Capabilities: capabilities,
	}
}

// convertDeviceStatusToProto converts internal device status to protobuf enum
func (h *DeviceListHandler) convertDeviceStatusToProto(status models.DeviceStatus) pb.DeviceStatus {
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

// convertProtoToDeviceStatus converts protobuf device status to internal enum
func (h *DeviceListHandler) convertProtoToDeviceStatus(status pb.DeviceStatus) models.DeviceStatus {
	switch status {
	case pb.DeviceStatus_DEVICE_STATUS_ONLINE:
		return models.DeviceStatusOnline
	case pb.DeviceStatus_DEVICE_STATUS_OFFLINE:
		return models.DeviceStatusOffline
	case pb.DeviceStatus_DEVICE_STATUS_ERROR:
		return models.DeviceStatusError
	case pb.DeviceStatus_DEVICE_STATUS_MAINTENANCE:
		return models.DeviceStatusMaintenance
	case pb.DeviceStatus_DEVICE_STATUS_CONNECTING:
		return models.DeviceStatusConnecting
	default:
		return models.DeviceStatusUnknown
	}
}

// isValidDeviceStatus checks if the protobuf device status is valid
func (h *DeviceListHandler) isValidDeviceStatus(status pb.DeviceStatus) bool {
	switch status {
	case pb.DeviceStatus_DEVICE_STATUS_UNKNOWN,
		pb.DeviceStatus_DEVICE_STATUS_ONLINE,
		pb.DeviceStatus_DEVICE_STATUS_OFFLINE,
		pb.DeviceStatus_DEVICE_STATUS_ERROR,
		pb.DeviceStatus_DEVICE_STATUS_MAINTENANCE,
		pb.DeviceStatus_DEVICE_STATUS_CONNECTING:
		return true
	default:
		return false
	}
}