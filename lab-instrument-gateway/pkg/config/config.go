package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
	Security SecurityConfig
	Performance PerformanceConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host     string
	Port     int
	GRPCPort int
	TLSCert  string
	TLSKey   string
	TLSCA    string
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Port    int
	Path    string
	Enabled bool
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	JWTSecret           string
	RateLimitRequests   int
	RateLimitWindow     time.Duration
	TLSEnabled          bool
}

// PerformanceConfig holds performance-related configuration
type PerformanceConfig struct {
	MaxConcurrentStreams int
	MaxMessageSize       int
	ConnectionTimeout    time.Duration
	KeepaliveTime        time.Duration
	KeepaliveTimeout     time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:     getEnv("SERVER_HOST", "0.0.0.0"),
			Port:     getEnvAsInt("SERVER_PORT", 8080),
			GRPCPort: getEnvAsInt("GRPC_PORT", 9090),
			TLSCert:  getEnv("TLS_CERT_FILE", ""),
			TLSKey:   getEnv("TLS_KEY_FILE", ""),
			TLSCA:    getEnv("TLS_CA_FILE", ""),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			Name:     getEnv("DB_NAME", "lab_instruments"),
			User:     getEnv("DB_USER", "user"),
			Password: getEnv("DB_PASSWORD", "password"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Metrics: MetricsConfig{
			Port:    getEnvAsInt("METRICS_PORT", 8081),
			Path:    getEnv("METRICS_PATH", "/metrics"),
			Enabled: getEnvAsBool("PROMETHEUS_ENABLED", true),
		},
		Security: SecurityConfig{
			JWTSecret:           getEnv("JWT_SECRET", "your-jwt-secret-key"),
			RateLimitRequests:   getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
			RateLimitWindow:     getEnvAsDuration("RATE_LIMIT_WINDOW", time.Minute),
			TLSEnabled:          getEnvAsBool("TLS_ENABLED", false),
		},
		Performance: PerformanceConfig{
			MaxConcurrentStreams: getEnvAsInt("MAX_CONCURRENT_STREAMS", 1000),
			MaxMessageSize:       getEnvAsInt("MAX_MESSAGE_SIZE", 4194304), // 4MB
			ConnectionTimeout:    getEnvAsDuration("CONNECTION_TIMEOUT", 30*time.Second),
			KeepaliveTime:        getEnvAsDuration("KEEPALIVE_TIME", 30*time.Second),
			KeepaliveTimeout:     getEnvAsDuration("KEEPALIVE_TIMEOUT", 5*time.Second),
		},
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}