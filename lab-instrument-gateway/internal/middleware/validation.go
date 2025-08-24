package middleware

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator interface for request validation
type Validator interface {
	Validate() error
}

// ValidationInterceptor creates a unary server interceptor for input validation
func ValidationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Validate request if it implements Validator interface
		if validator, ok := req.(Validator); ok {
			if err := validator.Validate(); err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("validation failed: %v", err))
			}
		}
		
		// Perform additional validation based on method
		if err := validateByMethod(info.FullMethod, req); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		
		// Call handler
		return handler(ctx, req)
	}
}

// validateByMethod performs method-specific validation
func validateByMethod(method string, req interface{}) error {
	switch {
	case strings.Contains(method, "RegisterDevice"):
		return validateRegisterDeviceRequest(req)
	case strings.Contains(method, "GetDeviceStatus"):
		return validateGetDeviceStatusRequest(req)
	case strings.Contains(method, "ListDevices"):
		return validateListDevicesRequest(req)
	case strings.Contains(method, "SendCommand"):
		return validateSendCommandRequest(req)
	case strings.Contains(method, "GetMeasurements"):
		return validateGetMeasurementsRequest(req)
	default:
		// Generic validation for unknown methods
		return validateGenericRequest(req)
	}
}

// validateRegisterDeviceRequest validates device registration requests
func validateRegisterDeviceRequest(req interface{}) error {
	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// Check if request is nil
	if !v.IsValid() {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Validate device_id field
	if deviceIDField := v.FieldByName("DeviceId"); deviceIDField.IsValid() {
		deviceID := deviceIDField.String()
		if err := validateDeviceID(deviceID); err != nil {
			return err
		}
	}
	
	// Validate name field
	if nameField := v.FieldByName("Name"); nameField.IsValid() {
		name := nameField.String()
		if err := validateDeviceName(name); err != nil {
			return err
		}
	}
	
	// Validate type field
	if typeField := v.FieldByName("Type"); typeField.IsValid() {
		deviceType := typeField.String()
		if err := validateDeviceType(deviceType); err != nil {
			return err
		}
	}
	
	// Validate version field
	if versionField := v.FieldByName("Version"); versionField.IsValid() {
		version := versionField.String()
		if err := validateVersion(version); err != nil {
			return err
		}
	}
	
	// Validate capabilities field
	if capabilitiesField := v.FieldByName("Capabilities"); capabilitiesField.IsValid() {
		if capabilitiesField.Kind() == reflect.Slice {
			capabilities := make([]string, capabilitiesField.Len())
			for i := 0; i < capabilitiesField.Len(); i++ {
				capabilities[i] = capabilitiesField.Index(i).String()
			}
			if err := validateCapabilities(capabilities); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// validateGetDeviceStatusRequest validates device status requests
func validateGetDeviceStatusRequest(req interface{}) error {
	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if !v.IsValid() {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Validate device_id field
	if deviceIDField := v.FieldByName("DeviceId"); deviceIDField.IsValid() {
		deviceID := deviceIDField.String()
		if err := validateDeviceID(deviceID); err != nil {
			return err
		}
	}
	
	return nil
}

// validateListDevicesRequest validates device listing requests
func validateListDevicesRequest(req interface{}) error {
	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if !v.IsValid() {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Validate page_size field
	if pageSizeField := v.FieldByName("PageSize"); pageSizeField.IsValid() {
		pageSize := int(pageSizeField.Int())
		if pageSize < 0 {
			return fmt.Errorf("page_size cannot be negative")
		}
		if pageSize > 1000 {
			return fmt.Errorf("page_size too large (max 1000)")
		}
	}
	
	// Validate sort_by field
	if sortByField := v.FieldByName("SortBy"); sortByField.IsValid() {
		sortBy := sortByField.String()
		if sortBy != "" {
			validSortFields := map[string]bool{
				"id":            true,
				"name":          true,
				"type":          true,
				"status":        true,
				"last_seen":     true,
				"registered_at": true,
				"updated_at":    true,
			}
			if !validSortFields[sortBy] {
				return fmt.Errorf("invalid sort field: %s", sortBy)
			}
		}
	}
	
	return nil
}

// validateSendCommandRequest validates command sending requests
func validateSendCommandRequest(req interface{}) error {
	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if !v.IsValid() {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Validate device_id field
	if deviceIDField := v.FieldByName("DeviceId"); deviceIDField.IsValid() {
		deviceID := deviceIDField.String()
		if err := validateDeviceID(deviceID); err != nil {
			return err
		}
	}
	
	// Validate timeout_seconds field
	if timeoutField := v.FieldByName("TimeoutSeconds"); timeoutField.IsValid() {
		timeout := int(timeoutField.Int())
		if timeout <= 0 {
			return fmt.Errorf("timeout_seconds must be positive")
		}
		if timeout > 3600 { // Max 1 hour
			return fmt.Errorf("timeout_seconds too large (max 3600)")
		}
	}
	
	return nil
}

// validateGetMeasurementsRequest validates measurement retrieval requests
func validateGetMeasurementsRequest(req interface{}) error {
	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if !v.IsValid() {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Validate device_id field
	if deviceIDField := v.FieldByName("DeviceId"); deviceIDField.IsValid() {
		deviceID := deviceIDField.String()
		if err := validateDeviceID(deviceID); err != nil {
			return err
		}
	}
	
	// Validate page_size field
	if pageSizeField := v.FieldByName("PageSize"); pageSizeField.IsValid() {
		pageSize := int(pageSizeField.Int())
		if pageSize < 0 {
			return fmt.Errorf("page_size cannot be negative")
		}
		if pageSize > 10000 {
			return fmt.Errorf("page_size too large (max 10000)")
		}
	}
	
	return nil
}

// validateGenericRequest performs generic validation for unknown request types
func validateGenericRequest(req interface{}) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Check if request is a pointer to nil
	v := reflect.ValueOf(req)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return fmt.Errorf("request cannot be nil")
	}
	
	return nil
}

// validateDeviceID validates device ID format and constraints
func validateDeviceID(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	
	if deviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	
	if len(deviceID) > 255 {
		return fmt.Errorf("device_id too long (max 255 characters)")
	}
	
	// Check for invalid characters
	if strings.ContainsAny(deviceID, " \t\n\r") {
		return fmt.Errorf("device_id cannot contain whitespace characters")
	}
	
	return nil
}

// validateDeviceName validates device name format and constraints
func validateDeviceName(name string) error {
	name = strings.TrimSpace(name)
	
	if name == "" {
		return fmt.Errorf("device name is required")
	}
	
	if len(name) > 255 {
		return fmt.Errorf("device name too long (max 255 characters)")
	}
	
	return nil
}

// validateDeviceType validates device type
func validateDeviceType(deviceType string) error {
	deviceType = strings.TrimSpace(deviceType)
	
	if deviceType == "" {
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
	
	if !validTypes[strings.ToLower(deviceType)] {
		return fmt.Errorf("invalid device type: %s", deviceType)
	}
	
	return nil
}

// validateVersion validates firmware version format
func validateVersion(version string) error {
	version = strings.TrimSpace(version)
	
	if version == "" {
		return fmt.Errorf("firmware version is required")
	}
	
	if len(version) > 50 {
		return fmt.Errorf("firmware version too long (max 50 characters)")
	}
	
	return nil
}

// validateCapabilities validates device capabilities
func validateCapabilities(capabilities []string) error {
	if len(capabilities) == 0 {
		return fmt.Errorf("at least one capability is required")
	}
	
	validCapabilities := map[string]bool{
		"temperature":      true,
		"humidity":         true,
		"pressure":         true,
		"ph":              true,
		"conductivity":     true,
		"turbidity":        true,
		"dissolved_oxygen": true,
		"flow_rate":        true,
		"level":            true,
		"weight":           true,
		"vibration":        true,
		"acceleration":     true,
		"voltage":          true,
		"current":          true,
		"power":            true,
		"frequency":        true,
		"spectrum":         true,
		"image":            true,
		"control":          true,
		"calibration":      true,
	}
	
	for _, capability := range capabilities {
		capability = strings.ToLower(strings.TrimSpace(capability))
		if !validCapabilities[capability] {
			return fmt.Errorf("invalid capability: %s", capability)
		}
	}
	
	return nil
}