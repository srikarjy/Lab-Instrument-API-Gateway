package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"github.com/yourorg/lab-gateway/pkg/logger"
)

// ContextKey represents a context key type
type ContextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey ContextKey = "correlation_id"
	// RequestStartTimeKey is the context key for request start time
	RequestStartTimeKey ContextKey = "request_start_time"
)

// LoggingInterceptor creates a unary server interceptor for request/response logging
func LoggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Generate correlation ID
		correlationID := uuid.New().String()
		ctx = context.WithValue(ctx, CorrelationIDKey, correlationID)
		
		// Record start time
		startTime := time.Now()
		ctx = context.WithValue(ctx, RequestStartTimeKey, startTime)
		
		// Create logger with correlation ID
		reqLogger := log.WithFields(map[string]interface{}{
			"correlation_id": correlationID,
			"method":         info.FullMethod,
			"request_time":   startTime.Format(time.RFC3339),
		})
		
		// Log request
		reqLogger.WithField("request", req).Info("gRPC request started")
		
		// Call handler
		resp, err := handler(ctx, req)
		
		// Calculate duration
		duration := time.Since(startTime)
		
		// Prepare log fields
		logFields := map[string]interface{}{
			"correlation_id": correlationID,
			"method":         info.FullMethod,
			"duration_ms":    duration.Milliseconds(),
			"duration":       duration.String(),
		}
		
		// Log response
		if err != nil {
			// Extract gRPC status
			grpcStatus := status.Convert(err)
			logFields["error"] = err.Error()
			logFields["grpc_code"] = grpcStatus.Code().String()
			logFields["grpc_message"] = grpcStatus.Message()
			
			// Log level based on error type
			if grpcStatus.Code() == codes.Internal || grpcStatus.Code() == codes.Unknown {
				reqLogger.WithFields(logFields).Error("gRPC request failed")
			} else {
				reqLogger.WithFields(logFields).Warn("gRPC request completed with error")
			}
		} else {
			logFields["response"] = resp
			reqLogger.WithFields(logFields).Info("gRPC request completed successfully")
		}
		
		return resp, err
	}
}

// StreamLoggingInterceptor creates a stream server interceptor for stream logging
func StreamLoggingInterceptor(log *logger.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Generate correlation ID
		correlationID := uuid.New().String()
		
		// Create wrapped stream with correlation ID in context
		wrappedStream := &loggingServerStream{
			ServerStream:  stream,
			correlationID: correlationID,
			logger:        log,
			method:        info.FullMethod,
			startTime:     time.Now(),
		}
		
		// Create logger with correlation ID
		streamLogger := log.WithFields(map[string]interface{}{
			"correlation_id": correlationID,
			"method":         info.FullMethod,
			"stream_type":    getStreamType(info),
		})
		
		// Log stream start
		streamLogger.Info("gRPC stream started")
		
		// Call handler
		err := handler(srv, wrappedStream)
		
		// Calculate duration
		duration := time.Since(wrappedStream.startTime)
		
		// Prepare log fields
		logFields := map[string]interface{}{
			"correlation_id":    correlationID,
			"method":            info.FullMethod,
			"duration_ms":       duration.Milliseconds(),
			"duration":          duration.String(),
			"messages_sent":     wrappedStream.messagesSent,
			"messages_received": wrappedStream.messagesReceived,
		}
		
		// Log stream completion
		if err != nil {
			grpcStatus := status.Convert(err)
			logFields["error"] = err.Error()
			logFields["grpc_code"] = grpcStatus.Code().String()
			logFields["grpc_message"] = grpcStatus.Message()
			
			if grpcStatus.Code() == codes.Internal || grpcStatus.Code() == codes.Unknown {
				streamLogger.WithFields(logFields).Error("gRPC stream failed")
			} else {
				streamLogger.WithFields(logFields).Warn("gRPC stream completed with error")
			}
		} else {
			streamLogger.WithFields(logFields).Info("gRPC stream completed successfully")
		}
		
		return err
	}
}

// loggingServerStream wraps grpc.ServerStream to add logging functionality
type loggingServerStream struct {
	grpc.ServerStream
	correlationID      string
	logger             *logger.Logger
	method             string
	startTime          time.Time
	messagesSent       int64
	messagesReceived   int64
}

// Context returns the context with correlation ID
func (s *loggingServerStream) Context() context.Context {
	ctx := s.ServerStream.Context()
	return context.WithValue(ctx, CorrelationIDKey, s.correlationID)
}

// SendMsg logs outgoing messages and delegates to the underlying stream
func (s *loggingServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.messagesSent++
		s.logger.WithFields(map[string]interface{}{
			"correlation_id": s.correlationID,
			"method":         s.method,
			"direction":      "outbound",
			"message_count":  s.messagesSent,
		}).Debug("gRPC stream message sent")
	}
	return err
}

// RecvMsg logs incoming messages and delegates to the underlying stream
func (s *loggingServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.messagesReceived++
		s.logger.WithFields(map[string]interface{}{
			"correlation_id": s.correlationID,
			"method":         s.method,
			"direction":      "inbound",
			"message_count":  s.messagesReceived,
		}).Debug("gRPC stream message received")
	}
	return err
}

// getStreamType determines the type of stream
func getStreamType(info *grpc.StreamServerInfo) string {
	if info.IsClientStream && info.IsServerStream {
		return "bidirectional"
	} else if info.IsClientStream {
		return "client_stream"
	} else if info.IsServerStream {
		return "server_stream"
	}
	return "unknown"
}

// GetCorrelationID extracts correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return correlationID
	}
	return ""
}

// GetRequestStartTime extracts request start time from context
func GetRequestStartTime(ctx context.Context) time.Time {
	if startTime, ok := ctx.Value(RequestStartTimeKey).(time.Time); ok {
		return startTime
	}
	return time.Time{}
}