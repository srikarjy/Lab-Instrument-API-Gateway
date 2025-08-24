package middleware

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// Request duration histogram
	grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)
	
	// Request counter
	grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status_code"},
	)
	
	// Active connections gauge
	grpcActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "grpc_active_connections",
			Help: "Number of active gRPC connections",
		},
	)
	
	// Device registration metrics
	deviceRegistrationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "device_registrations_total",
			Help: "Total number of device registrations",
		},
		[]string{"device_type", "status"},
	)
	
	// Device status metrics
	deviceStatusTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "device_status_total",
			Help: "Number of devices by status",
		},
		[]string{"status"},
	)
	
	// Stream metrics
	grpcStreamMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_stream_messages_total",
			Help: "Total number of stream messages",
		},
		[]string{"method", "direction"},
	)
	
	// Stream duration
	grpcStreamDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_stream_duration_seconds",
			Help:    "Duration of gRPC streams in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)
	
	// Repository operation metrics
	repositoryOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "repository_operation_duration_seconds",
			Help:    "Duration of repository operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "repository", "status"},
	)
	
	// Database connection pool metrics
	databaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)
	
	databaseConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_idle",
			Help: "Number of idle database connections",
		},
	)
	
	// Error metrics
	grpcErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_errors_total",
			Help: "Total number of gRPC errors",
		},
		[]string{"method", "error_code", "error_type"},
	)
)

// MetricsInterceptor creates a unary server interceptor for metrics collection
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()
		
		// Increment active connections
		grpcActiveConnections.Inc()
		defer grpcActiveConnections.Dec()
		
		// Call handler
		resp, err := handler(ctx, req)
		
		// Calculate duration
		duration := time.Since(startTime)
		
		// Extract method name
		method := info.FullMethod
		
		// Determine status code
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}
		
		// Record metrics
		grpcRequestDuration.WithLabelValues(method, statusCode.String()).Observe(duration.Seconds())
		grpcRequestsTotal.WithLabelValues(method, statusCode.String()).Inc()
		
		// Record error metrics if applicable
		if err != nil {
			errorType := getErrorType(statusCode)
			grpcErrorsTotal.WithLabelValues(method, statusCode.String(), errorType).Inc()
		}
		
		// Record device-specific metrics
		recordDeviceMetrics(method, req, resp, err)
		
		return resp, err
	}
}

// StreamMetricsInterceptor creates a stream server interceptor for metrics collection
func StreamMetricsInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()
		
		// Increment active connections
		grpcActiveConnections.Inc()
		defer grpcActiveConnections.Dec()
		
		// Create wrapped stream for message counting
		wrappedStream := &metricsServerStream{
			ServerStream: stream,
			method:       info.FullMethod,
		}
		
		// Call handler
		err := handler(srv, wrappedStream)
		
		// Calculate duration
		duration := time.Since(startTime)
		
		// Determine status code
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}
		
		// Record metrics
		grpcStreamDuration.WithLabelValues(info.FullMethod, statusCode.String()).Observe(duration.Seconds())
		
		// Record message metrics
		grpcStreamMessagesTotal.WithLabelValues(info.FullMethod, "sent").Add(float64(wrappedStream.messagesSent))
		grpcStreamMessagesTotal.WithLabelValues(info.FullMethod, "received").Add(float64(wrappedStream.messagesReceived))
		
		// Record error metrics if applicable
		if err != nil {
			errorType := getErrorType(statusCode)
			grpcErrorsTotal.WithLabelValues(info.FullMethod, statusCode.String(), errorType).Inc()
		}
		
		return err
	}
}

// metricsServerStream wraps grpc.ServerStream to count messages
type metricsServerStream struct {
	grpc.ServerStream
	method           string
	messagesSent     int64
	messagesReceived int64
}

// SendMsg counts outgoing messages
func (s *metricsServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.messagesSent++
	}
	return err
}

// RecvMsg counts incoming messages
func (s *metricsServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.messagesReceived++
	}
	return err
}

// recordDeviceMetrics records device-specific metrics based on the method and request
func recordDeviceMetrics(method string, req, resp interface{}, err error) {
	switch {
	case method == "/lab_instrument.LabInstrumentGateway/RegisterDevice":
		recordDeviceRegistrationMetrics(req, resp, err)
	case method == "/lab_instrument.LabInstrumentGateway/GetDeviceStatus":
		recordDeviceStatusMetrics(req, resp, err)
	case method == "/lab_instrument.LabInstrumentGateway/ListDevices":
		recordDeviceListMetrics(req, resp, err)
	}
}

// recordDeviceRegistrationMetrics records metrics for device registration
func recordDeviceRegistrationMetrics(req, resp interface{}, err error) {
	// Extract device type from request using reflection
	deviceType := "unknown"
	status := "success"
	
	if err != nil {
		status = "error"
	}
	
	// Try to extract device type from request
	if reqMap, ok := req.(map[string]interface{}); ok {
		if dt, exists := reqMap["type"]; exists {
			if dtStr, ok := dt.(string); ok {
				deviceType = dtStr
			}
		}
	}
	
	deviceRegistrationsTotal.WithLabelValues(deviceType, status).Inc()
}

// recordDeviceStatusMetrics records metrics for device status requests
func recordDeviceStatusMetrics(req, resp interface{}, err error) {
	// Could extract and record device status distribution
	// This would require parsing the response to get device status
}

// recordDeviceListMetrics records metrics for device list requests
func recordDeviceListMetrics(req, resp interface{}, err error) {
	// Could record metrics about list operations
	// Such as number of devices returned, filter usage, etc.
}

// getErrorType categorizes errors for metrics
func getErrorType(code codes.Code) string {
	switch code {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return "client_error"
	case codes.DeadlineExceeded, codes.Canceled:
		return "timeout"
	case codes.NotFound:
		return "not_found"
	case codes.AlreadyExists:
		return "conflict"
	case codes.PermissionDenied, codes.Unauthenticated:
		return "auth_error"
	case codes.ResourceExhausted:
		return "rate_limit"
	case codes.Internal, codes.Unknown, codes.DataLoss:
		return "server_error"
	case codes.Unimplemented:
		return "not_implemented"
	case codes.Unavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

// UpdateDatabaseConnectionMetrics updates database connection pool metrics
func UpdateDatabaseConnectionMetrics(active, idle int) {
	databaseConnectionsActive.Set(float64(active))
	databaseConnectionsIdle.Set(float64(idle))
}

// RecordRepositoryOperation records metrics for repository operations
func RecordRepositoryOperation(operation, repository string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	
	repositoryOperationDuration.WithLabelValues(operation, repository, status).Observe(duration.Seconds())
}

// UpdateDeviceStatusMetrics updates device status distribution metrics
func UpdateDeviceStatusMetrics(statusCounts map[string]int) {
	// Reset all status gauges
	deviceStatusTotal.Reset()
	
	// Set current counts
	for status, count := range statusCounts {
		deviceStatusTotal.WithLabelValues(status).Set(float64(count))
	}
}

// GetMetricsRegistry returns the Prometheus registry for metrics exposure
func GetMetricsRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}