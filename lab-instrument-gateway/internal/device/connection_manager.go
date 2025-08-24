package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/lab-gateway/pkg/logger"
	"github.com/yourorg/lab-gateway/pkg/models"
)

// ConnectionStatus represents the status of a device connection
type ConnectionStatus struct {
	ConnectionID     string                 `json:"connection_id"`
	DeviceID         string                 `json:"device_id"`
	SessionID        string                 `json:"session_id"`
	StreamID         *string                `json:"stream_id,omitempty"`
	IsConnected      bool                   `json:"is_connected"`
	IsHealthy        bool                   `json:"is_healthy"`
	ConnectedAt      time.Time              `json:"connected_at"`
	LastSeen         time.Time              `json:"last_seen"`
	LastHeartbeat    time.Time              `json:"last_heartbeat"`
	MessagesSent     int64                  `json:"messages_sent"`
	MessagesReceived int64                  `json:"messages_received"`
	BytesSent        int64                  `json:"bytes_sent"`
	BytesReceived    int64                  `json:"bytes_received"`
	LastError        *string                `json:"last_error,omitempty"`
	LastErrorAt      time.Time              `json:"last_error_at"`
	Metrics          map[string]interface{} `json:"metrics,omitempty"`
}

// ConnectionManager manages active device connections
type ConnectionManager struct {
	connections map[string]*ConnectionStatus
	sessions    map[string]*models.DeviceSession
	mutex       sync.RWMutex
	logger      *logger.Logger
	
	// Configuration
	heartbeatTimeout time.Duration
	cleanupInterval  time.Duration
	
	// Channels for lifecycle management
	stopChan chan struct{}
	doneChan chan struct{}
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(logger *logger.Logger) *ConnectionManager {
	cm := &ConnectionManager{
		connections:      make(map[string]*ConnectionStatus),
		sessions:         make(map[string]*models.DeviceSession),
		logger:           logger,
		heartbeatTimeout: 2 * time.Minute,
		cleanupInterval:  30 * time.Second,
		stopChan:         make(chan struct{}),
		doneChan:         make(chan struct{}),
	}
	
	// Start background cleanup routine
	go cm.cleanupRoutine()
	
	return cm
}

// GenerateSessionID generates a new session ID for a device
func (cm *ConnectionManager) GenerateSessionID(deviceID string) string {
	return fmt.Sprintf("%s-%s", deviceID, uuid.New().String()[:8])
}

// RegisterConnection registers a new device connection
func (cm *ConnectionManager) RegisterConnection(ctx context.Context, session *models.DeviceSession) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	connectionID := uuid.New().String()
	now := time.Now()
	
	// Create connection status
	connStatus := &ConnectionStatus{
		ConnectionID:     connectionID,
		DeviceID:         session.DeviceID,
		SessionID:        session.SessionID,
		StreamID:         session.StreamID,
		IsConnected:      true,
		IsHealthy:        true,
		ConnectedAt:      now,
		LastSeen:         now,
		LastHeartbeat:    now,
		MessagesSent:     0,
		MessagesReceived: 0,
		BytesSent:        0,
		BytesReceived:    0,
		Metrics:          make(map[string]interface{}),
	}
	
	// Store connection and session
	cm.connections[session.DeviceID] = connStatus
	cm.sessions[session.SessionID] = session
	
	cm.logger.WithFields(map[string]interface{}{
		"device_id":     session.DeviceID,
		"session_id":    session.SessionID,
		"connection_id": connectionID,
	}).Info("Device connection registered")
	
	return nil
}

// UpdateHeartbeat updates the heartbeat timestamp for a device
func (cm *ConnectionManager) UpdateHeartbeat(deviceID string, metrics map[string]interface{}) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	connStatus, exists := cm.connections[deviceID]
	if !exists {
		return fmt.Errorf("connection not found for device: %s", deviceID)
	}
	
	now := time.Now()
	connStatus.LastHeartbeat = now
	connStatus.LastSeen = now
	connStatus.IsHealthy = true
	
	// Update metrics if provided
	if metrics != nil {
		if connStatus.Metrics == nil {
			connStatus.Metrics = make(map[string]interface{})
		}
		for k, v := range metrics {
			connStatus.Metrics[k] = v
		}
	}
	
	cm.logger.WithFields(map[string]interface{}{
		"device_id":      deviceID,
		"connection_id":  connStatus.ConnectionID,
		"last_heartbeat": now,
	}).Debug("Device heartbeat updated")
	
	return nil
}

// UpdateConnectionStats updates connection statistics
func (cm *ConnectionManager) UpdateConnectionStats(deviceID string, messagesSent, messagesReceived, bytesSent, bytesReceived int64) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	connStatus, exists := cm.connections[deviceID]
	if !exists {
		return fmt.Errorf("connection not found for device: %s", deviceID)
	}
	
	connStatus.MessagesSent += messagesSent
	connStatus.MessagesReceived += messagesReceived
	connStatus.BytesSent += bytesSent
	connStatus.BytesReceived += bytesReceived
	connStatus.LastSeen = time.Now()
	
	return nil
}

// RecordConnectionError records a connection error
func (cm *ConnectionManager) RecordConnectionError(deviceID string, errorMsg string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	connStatus, exists := cm.connections[deviceID]
	if !exists {
		return fmt.Errorf("connection not found for device: %s", deviceID)
	}
	
	now := time.Now()
	connStatus.LastError = &errorMsg
	connStatus.LastErrorAt = now
	connStatus.IsHealthy = false
	
	cm.logger.WithFields(map[string]interface{}{
		"device_id":     deviceID,
		"connection_id": connStatus.ConnectionID,
		"error":         errorMsg,
	}).Warn("Connection error recorded")
	
	return nil
}

// DisconnectDevice marks a device as disconnected
func (cm *ConnectionManager) DisconnectDevice(deviceID string, reason string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	connStatus, exists := cm.connections[deviceID]
	if !exists {
		return fmt.Errorf("connection not found for device: %s", deviceID)
	}
	
	connStatus.IsConnected = false
	connStatus.IsHealthy = false
	connStatus.LastSeen = time.Now()
	
	if reason != "" {
		connStatus.LastError = &reason
		connStatus.LastErrorAt = time.Now()
	}
	
	// Remove from sessions
	if session, exists := cm.sessions[connStatus.SessionID]; exists {
		session.IsActive = false
		delete(cm.sessions, connStatus.SessionID)
	}
	
	cm.logger.WithFields(map[string]interface{}{
		"device_id":     deviceID,
		"connection_id": connStatus.ConnectionID,
		"reason":        reason,
	}).Info("Device disconnected")
	
	return nil
}

// GetConnectionStatus returns the connection status for a device
func (cm *ConnectionManager) GetConnectionStatus(deviceID string) *ConnectionStatus {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	connStatus, exists := cm.connections[deviceID]
	if !exists {
		return nil
	}
	
	// Create a copy to avoid race conditions
	statusCopy := *connStatus
	if connStatus.Metrics != nil {
		statusCopy.Metrics = make(map[string]interface{})
		for k, v := range connStatus.Metrics {
			statusCopy.Metrics[k] = v
		}
	}
	
	return &statusCopy
}

// GetActiveConnections returns all active connections
func (cm *ConnectionManager) GetActiveConnections() map[string]*ConnectionStatus {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	activeConnections := make(map[string]*ConnectionStatus)
	
	for deviceID, connStatus := range cm.connections {
		if connStatus.IsConnected {
			// Create a copy
			statusCopy := *connStatus
			if connStatus.Metrics != nil {
				statusCopy.Metrics = make(map[string]interface{})
				for k, v := range connStatus.Metrics {
					statusCopy.Metrics[k] = v
				}
			}
			activeConnections[deviceID] = &statusCopy
		}
	}
	
	return activeConnections
}

// GetConnectionCount returns the number of active connections
func (cm *ConnectionManager) GetConnectionCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	count := 0
	for _, connStatus := range cm.connections {
		if connStatus.IsConnected {
			count++
		}
	}
	
	return count
}

// GetSessionByID returns a session by its ID
func (cm *ConnectionManager) GetSessionByID(sessionID string) *models.DeviceSession {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	session, exists := cm.sessions[sessionID]
	if !exists {
		return nil
	}
	
	// Create a copy
	sessionCopy := *session
	if session.Metadata != nil {
		sessionCopy.Metadata = make(map[string]interface{})
		for k, v := range session.Metadata {
			sessionCopy.Metadata[k] = v
		}
	}
	
	return &sessionCopy
}

// cleanupRoutine runs periodic cleanup of stale connections
func (cm *ConnectionManager) cleanupRoutine() {
	defer close(cm.doneChan)
	
	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cm.stopChan:
			cm.logger.Info("Connection manager cleanup routine stopping")
			return
		case <-ticker.C:
			cm.performCleanup()
		}
	}
}

// performCleanup removes stale connections and updates health status
func (cm *ConnectionManager) performCleanup() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	now := time.Now()
	staleConnections := make([]string, 0)
	unhealthyConnections := make([]string, 0)
	
	for deviceID, connStatus := range cm.connections {
		// Check for stale connections (no heartbeat for too long)
		if now.Sub(connStatus.LastHeartbeat) > cm.heartbeatTimeout {
			if connStatus.IsConnected {
				connStatus.IsConnected = false
				connStatus.IsHealthy = false
				unhealthyConnections = append(unhealthyConnections, deviceID)
			}
			
			// Remove very old connections (offline for more than 1 hour)
			if now.Sub(connStatus.LastSeen) > time.Hour {
				staleConnections = append(staleConnections, deviceID)
			}
		}
	}
	
	// Remove stale connections
	for _, deviceID := range staleConnections {
		if connStatus, exists := cm.connections[deviceID]; exists {
			// Remove associated session
			if _, exists := cm.sessions[connStatus.SessionID]; exists {
				delete(cm.sessions, connStatus.SessionID)
			}
			delete(cm.connections, deviceID)
		}
	}
	
	if len(staleConnections) > 0 || len(unhealthyConnections) > 0 {
		cm.logger.WithFields(map[string]interface{}{
			"stale_connections":     len(staleConnections),
			"unhealthy_connections": len(unhealthyConnections),
			"active_connections":    cm.getActiveConnectionCount(),
		}).Debug("Connection cleanup completed")
	}
}

// getActiveConnectionCount returns the count of active connections (must be called with lock held)
func (cm *ConnectionManager) getActiveConnectionCount() int {
	count := 0
	for _, connStatus := range cm.connections {
		if connStatus.IsConnected {
			count++
		}
	}
	return count
}

// GetStats returns connection manager statistics
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	totalConnections := len(cm.connections)
	activeConnections := cm.getActiveConnectionCount()
	totalSessions := len(cm.sessions)
	
	var totalMessagesSent, totalMessagesReceived int64
	var totalBytesSent, totalBytesReceived int64
	
	for _, connStatus := range cm.connections {
		totalMessagesSent += connStatus.MessagesSent
		totalMessagesReceived += connStatus.MessagesReceived
		totalBytesSent += connStatus.BytesSent
		totalBytesReceived += connStatus.BytesReceived
	}
	
	return map[string]interface{}{
		"total_connections":      totalConnections,
		"active_connections":     activeConnections,
		"total_sessions":         totalSessions,
		"total_messages_sent":    totalMessagesSent,
		"total_messages_received": totalMessagesReceived,
		"total_bytes_sent":       totalBytesSent,
		"total_bytes_received":   totalBytesReceived,
		"heartbeat_timeout":      cm.heartbeatTimeout.String(),
		"cleanup_interval":       cm.cleanupInterval.String(),
	}
}

// Close shuts down the connection manager
func (cm *ConnectionManager) Close() error {
	cm.logger.Info("Shutting down connection manager")
	
	close(cm.stopChan)
	
	// Wait for cleanup routine to finish
	select {
	case <-cm.doneChan:
		cm.logger.Info("Connection manager cleanup routine stopped")
	case <-time.After(5 * time.Second):
		cm.logger.Warn("Connection manager cleanup routine did not stop within timeout")
	}
	
	// Clear all connections and sessions
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	cm.connections = make(map[string]*ConnectionStatus)
	cm.sessions = make(map[string]*models.DeviceSession)
	
	cm.logger.Info("Connection manager shut down completed")
	return nil
}