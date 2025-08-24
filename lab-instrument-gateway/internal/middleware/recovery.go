package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yourorg/lab-gateway/pkg/logger"
)

// RecoveryInterceptor creates a unary server interceptor for panic recovery
func RecoveryInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get correlation ID if available
				correlationID := GetCorrelationID(ctx)
				
				// Log the panic with stack trace
				log.WithFields(map[string]interface{}{
					"correlation_id": correlationID,
					"method":         info.FullMethod,
					"panic":          r,
					"stack_trace":    string(debug.Stack()),
					"request":        req,
				}).Error("gRPC handler panicked")
				
				// Return internal server error
				err = status.Error(codes.Internal, "Internal server error occurred")
			}
		}()
		
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor creates a stream server interceptor for panic recovery
func StreamRecoveryInterceptor(log *logger.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get correlation ID if available
				correlationID := GetCorrelationID(stream.Context())
				
				// Log the panic with stack trace
				log.WithFields(map[string]interface{}{
					"correlation_id": correlationID,
					"method":         info.FullMethod,
					"panic":          r,
					"stack_trace":    string(debug.Stack()),
					"stream_type":    getStreamType(info),
				}).Error("gRPC stream handler panicked")
				
				// Return internal server error
				err = status.Error(codes.Internal, "Internal server error occurred")
			}
		}()
		
		return handler(srv, stream)
	}
}

// RecoveryHandler is a function type for custom panic recovery handling
type RecoveryHandler func(ctx context.Context, method string, panic interface{}) error

// RecoveryInterceptorWithHandler creates a unary server interceptor with custom recovery handling
func RecoveryInterceptorWithHandler(log *logger.Logger, recoveryHandler RecoveryHandler) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get correlation ID if available
				correlationID := GetCorrelationID(ctx)
				
				// Log the panic with stack trace
				log.WithFields(map[string]interface{}{
					"correlation_id": correlationID,
					"method":         info.FullMethod,
					"panic":          r,
					"stack_trace":    string(debug.Stack()),
					"request":        req,
				}).Error("gRPC handler panicked")
				
				// Use custom recovery handler if provided
				if recoveryHandler != nil {
					err = recoveryHandler(ctx, info.FullMethod, r)
				} else {
					err = status.Error(codes.Internal, "Internal server error occurred")
				}
			}
		}()
		
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptorWithHandler creates a stream server interceptor with custom recovery handling
func StreamRecoveryInterceptorWithHandler(log *logger.Logger, recoveryHandler RecoveryHandler) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get correlation ID if available
				correlationID := GetCorrelationID(stream.Context())
				
				// Log the panic with stack trace
				log.WithFields(map[string]interface{}{
					"correlation_id": correlationID,
					"method":         info.FullMethod,
					"panic":          r,
					"stack_trace":    string(debug.Stack()),
					"stream_type":    getStreamType(info),
				}).Error("gRPC stream handler panicked")
				
				// Use custom recovery handler if provided
				if recoveryHandler != nil {
					err = recoveryHandler(stream.Context(), info.FullMethod, r)
				} else {
					err = status.Error(codes.Internal, "Internal server error occurred")
				}
			}
		}()
		
		return handler(srv, stream)
	}
}

// DefaultRecoveryHandler provides a default implementation for panic recovery
func DefaultRecoveryHandler(ctx context.Context, method string, panic interface{}) error {
	// Categorize panics and return appropriate errors
	switch p := panic.(type) {
	case string:
		if p == "runtime error: invalid memory address or nil pointer dereference" {
			return status.Error(codes.Internal, "Null pointer error occurred")
		}
		return status.Error(codes.Internal, fmt.Sprintf("Server error: %s", p))
	case error:
		return status.Error(codes.Internal, fmt.Sprintf("Server error: %v", p))
	default:
		return status.Error(codes.Internal, "Unknown server error occurred")
	}
}

// ValidationRecoveryHandler handles panics that might occur during validation
func ValidationRecoveryHandler(ctx context.Context, method string, panic interface{}) error {
	// Check if panic is related to validation
	switch p := panic.(type) {
	case string:
		if contains(p, "validation") || contains(p, "invalid") {
			return status.Error(codes.InvalidArgument, "Request validation failed")
		}
	case error:
		if contains(p.Error(), "validation") || contains(p.Error(), "invalid") {
			return status.Error(codes.InvalidArgument, "Request validation failed")
		}
	}
	
	// Fall back to default handling
	return DefaultRecoveryHandler(ctx, method, panic)
}

// DatabaseRecoveryHandler handles panics that might occur during database operations
func DatabaseRecoveryHandler(ctx context.Context, method string, panic interface{}) error {
	// Check if panic is related to database operations
	switch p := panic.(type) {
	case string:
		if contains(p, "database") || contains(p, "sql") || contains(p, "connection") {
			return status.Error(codes.Unavailable, "Database service temporarily unavailable")
		}
	case error:
		errStr := p.Error()
		if contains(errStr, "database") || contains(errStr, "sql") || contains(errStr, "connection") {
			return status.Error(codes.Unavailable, "Database service temporarily unavailable")
		}
	}
	
	// Fall back to default handling
	return DefaultRecoveryHandler(ctx, method, panic)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     containsSubstring(s, substr)))
}

// containsSubstring performs a simple substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// PanicInfo represents information about a panic
type PanicInfo struct {
	Method        string
	CorrelationID string
	Panic         interface{}
	StackTrace    string
	Context       context.Context
}

// RecoveryCallback is called when a panic is recovered
type RecoveryCallback func(info PanicInfo)

// RecoveryInterceptorWithCallback creates a recovery interceptor that calls a callback on panic
func RecoveryInterceptorWithCallback(log *logger.Logger, callback RecoveryCallback) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get correlation ID if available
				correlationID := GetCorrelationID(ctx)
				stackTrace := string(debug.Stack())
				
				// Log the panic
				log.WithFields(map[string]interface{}{
					"correlation_id": correlationID,
					"method":         info.FullMethod,
					"panic":          r,
					"stack_trace":    stackTrace,
					"request":        req,
				}).Error("gRPC handler panicked")
				
				// Call callback if provided
				if callback != nil {
					panicInfo := PanicInfo{
						Method:        info.FullMethod,
						CorrelationID: correlationID,
						Panic:         r,
						StackTrace:    stackTrace,
						Context:       ctx,
					}
					callback(panicInfo)
				}
				
				// Return internal server error
				err = status.Error(codes.Internal, "Internal server error occurred")
			}
		}()
		
		return handler(ctx, req)
	}
}