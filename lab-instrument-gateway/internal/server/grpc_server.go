package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/yourorg/lab-gateway/internal/device"
	"github.com/yourorg/lab-gateway/internal/handlers"
	"github.com/yourorg/lab-gateway/internal/middleware"
	"github.com/yourorg/lab-gateway/pkg/logger"
	"github.com/yourorg/lab-gateway/pkg/repository"
	pb "github.com/yourorg/lab-gateway/proto"
)

// GRPCServer represents the gRPC server
type GRPCServer struct {
	server            *grpc.Server
	listener          net.Listener
	repos             repository.RepositoryManager
	connectionManager *device.ConnectionManager
	logger            *logger.Logger
	
	// Handlers
	deviceHandler       *handlers.DeviceHandler
	deviceStatusHandler *handlers.DeviceStatusHandler
	deviceListHandler   *handlers.DeviceListHandler
	
	// Configuration
	port           int
	maxMessageSize int
	maxConcurrent  int
}

// Config represents the gRPC server configuration
type Config struct {
	Port           int
	MaxMessageSize int // in bytes
	MaxConcurrent  int // max concurrent streams
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(config Config, repos repository.RepositoryManager, logger *logger.Logger) (*GRPCServer, error) {
	// Create connection manager
	connectionManager := device.NewConnectionManager(logger)
	
	// Create handlers
	deviceHandler := handlers.NewDeviceHandler(repos, connectionManager, logger)
	deviceStatusHandler := handlers.NewDeviceStatusHandler(repos, connectionManager, logger)
	deviceListHandler := handlers.NewDeviceListHandler(repos, logger)
	
	// Set default configuration values
	if config.Port == 0 {
		config.Port = 8080
	}
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 4 * 1024 * 1024 // 4MB
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 1000
	}
	
	return &GRPCServer{
		repos:               repos,
		connectionManager:   connectionManager,
		logger:              logger,
		deviceHandler:       deviceHandler,
		deviceStatusHandler: deviceStatusHandler,
		deviceListHandler:   deviceListHandler,
		port:                config.Port,
		maxMessageSize:      config.MaxMessageSize,
		maxConcurrent:       config.MaxConcurrent,
	}, nil
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}
	s.listener = listener
	
	// Create gRPC server with options
	serverOptions := []grpc.ServerOption{
		// Message size limits
		grpc.MaxRecvMsgSize(s.maxMessageSize),
		grpc.MaxSendMsgSize(s.maxMessageSize),
		
		// Concurrent streams limit
		grpc.MaxConcurrentStreams(uint32(s.maxConcurrent)),
		
		// Keepalive settings for high-concurrency scenarios
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		
		// Middleware chain
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor(s.logger),
			middleware.ValidationInterceptor(),
			middleware.MetricsInterceptor(),
			middleware.RecoveryInterceptor(s.logger),
		),
		grpc.ChainStreamInterceptor(
			middleware.StreamLoggingInterceptor(s.logger),
			middleware.StreamRecoveryInterceptor(s.logger),
		),
	}
	
	s.server = grpc.NewServer(serverOptions...)
	
	// Register service implementation
	labInstrumentService := &LabInstrumentService{
		deviceHandler:       s.deviceHandler,
		deviceStatusHandler: s.deviceStatusHandler,
		deviceListHandler:   s.deviceListHandler,
		connectionManager:   s.connectionManager,
		repos:               s.repos,
		logger:              s.logger,
	}
	
	pb.RegisterLabInstrumentGatewayServer(s.server, labInstrumentService)
	
	// Enable reflection for development
	reflection.Register(s.server)
	
	s.logger.WithFields(map[string]interface{}{
		"port":             s.port,
		"max_message_size": s.maxMessageSize,
		"max_concurrent":   s.maxConcurrent,
	}).Info("Starting gRPC server")
	
	// Start serving
	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.logger.WithError(err).Error("gRPC server failed")
		}
	}()
	
	s.logger.WithField("address", listener.Addr().String()).Info("gRPC server started")
	return nil
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping gRPC server")
	
	// Create a channel to signal when graceful stop is complete
	stopped := make(chan struct{})
	
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()
	
	// Wait for graceful stop or context timeout
	select {
	case <-stopped:
		s.logger.Info("gRPC server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("gRPC server graceful stop timed out, forcing stop")
		s.server.Stop()
	}
	
	// Close connection manager
	if err := s.connectionManager.Close(); err != nil {
		s.logger.WithError(err).Warn("Failed to close connection manager")
	}
	
	return nil
}

// GetStats returns server statistics
func (s *GRPCServer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"port":             s.port,
		"max_message_size": s.maxMessageSize,
		"max_concurrent":   s.maxConcurrent,
		"is_serving":       s.server != nil,
	}
	
	// Add connection manager stats
	if s.connectionManager != nil {
		connStats := s.connectionManager.GetStats()
		for k, v := range connStats {
			stats[fmt.Sprintf("connection_%s", k)] = v
		}
	}
	
	return stats
}

// LabInstrumentService implements the gRPC service interface
type LabInstrumentService struct {
	pb.UnimplementedLabInstrumentGatewayServer
	
	deviceHandler       *handlers.DeviceHandler
	deviceStatusHandler *handlers.DeviceStatusHandler
	deviceListHandler   *handlers.DeviceListHandler
	connectionManager   *device.ConnectionManager
	repos               repository.RepositoryManager
	logger              *logger.Logger
}

// RegisterDevice handles device registration
func (s *LabInstrumentService) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.RegisterDeviceResponse, error) {
	return s.deviceHandler.RegisterDevice(ctx, req)
}

// GetDeviceStatus handles device status requests
func (s *LabInstrumentService) GetDeviceStatus(ctx context.Context, req *pb.GetDeviceStatusRequest) (*pb.GetDeviceStatusResponse, error) {
	return s.deviceStatusHandler.GetDeviceStatus(ctx, req)
}

// ListDevices handles device listing requests
func (s *LabInstrumentService) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	return s.deviceListHandler.ListDevices(ctx, req)
}

// StreamData handles real-time data streaming (placeholder implementation)
func (s *LabInstrumentService) StreamData(stream pb.LabInstrumentGateway_StreamDataServer) error {
	// TODO: Implement streaming functionality
	s.logger.Info("StreamData called - not yet implemented")
	return fmt.Errorf("streaming not yet implemented")
}

// SendCommand handles command sending (placeholder implementation)
func (s *LabInstrumentService) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	// TODO: Implement command sending functionality
	s.logger.WithField("device_id", req.DeviceId).Info("SendCommand called - not yet implemented")
	return nil, fmt.Errorf("command sending not yet implemented")
}

// GetMeasurements handles historical data requests (placeholder implementation)
func (s *LabInstrumentService) GetMeasurements(ctx context.Context, req *pb.GetMeasurementsRequest) (*pb.GetMeasurementsResponse, error) {
	// TODO: Implement measurements retrieval functionality
	s.logger.WithField("device_id", req.DeviceId).Info("GetMeasurements called - not yet implemented")
	return nil, fmt.Errorf("measurements retrieval not yet implemented")
}

// HealthCheck handles health check requests
func (s *LabInstrumentService) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// Perform repository health check
	if err := s.repos.HealthCheck(ctx); err != nil {
		s.logger.WithError(err).Error("Repository health check failed")
		return &pb.HealthCheckResponse{
			Status:    pb.HealthStatus_HEALTH_NOT_SERVING,
			Message:   "Repository health check failed",
			Timestamp: nil,
		}, nil
	}
	
	// Get connection manager stats
	connStats := s.connectionManager.GetStats()
	details := make(map[string]string)
	for k, v := range connStats {
		details[k] = fmt.Sprintf("%v", v)
	}
	
	return &pb.HealthCheckResponse{
		Status:    pb.HealthStatus_HEALTH_SERVING,
		Message:   "Service is healthy",
		Details:   details,
		Timestamp: nil,
	}, nil
}