# Lab Instrument API Gateway

A high-performance gRPC-based API gateway for managing laboratory instrument connections, real-time data streaming, and device control.

## Features

- **Device Management**: Register and manage laboratory instruments
- **Real-time Streaming**: Bidirectional data streaming with 10,000+ messages/second throughput
- **Command Execution**: Remote device control and command tracking
- **Historical Data**: Query and analyze measurement history
- **High Availability**: Supports 1000+ concurrent connections with 99.9% uptime
- **Security**: mTLS authentication and comprehensive authorization
- **Monitoring**: Prometheus metrics and structured logging

## Architecture

- **Technology Stack**: Go, gRPC, PostgreSQL, Docker, Kubernetes
- **Deployment**: AWS EKS with horizontal pod autoscaling
- **Database**: AWS RDS PostgreSQL with Multi-AZ deployment
- **Monitoring**: Prometheus, Grafana, CloudWatch

## Security

### 🔒 Security Features

- **mTLS Authentication**: Mutual TLS for all gRPC connections
- **JWT Authorization**: Role-based access control with JWT tokens
- **Input Validation**: Comprehensive request validation and sanitization
- **Rate Limiting**: Configurable rate limiting per client/endpoint
- **SQL Injection Protection**: Prepared statements and parameterized queries
- **CORS Protection**: Configurable CORS policies
- **Encryption**: Data encryption at rest and in transit

### 🛡️ Security Best Practices

1. **Environment Variables**: Never commit sensitive data to version control
   ```bash
   cp .env.example .env
   # Edit .env with your secure values
   ```

2. **Database Security**:
   - Use strong passwords (min 12 characters with mixed case, numbers, symbols)
   - Enable SSL/TLS for database connections
   - Use database connection pooling with proper limits
   - Regular security updates and patches

3. **TLS Configuration**:
   ```bash
   # Generate TLS certificates for production
   openssl genrsa -out server.key 2048
   openssl req -new -x509 -key server.key -out server.crt -days 365
   ```

4. **JWT Security**:
   - Use strong JWT secrets (min 32 characters)
   - Implement token rotation and expiration
   - Store tokens securely (httpOnly cookies recommended)

5. **Network Security**:
   - Use firewalls and security groups
   - Implement network segmentation
   - Regular security audits and penetration testing

### 🚨 Security Checklist

- [ ] Change all default passwords
- [ ] Generate strong JWT secrets
- [ ] Enable TLS/SSL for all connections
- [ ] Configure proper CORS policies
- [ ] Set up rate limiting
- [ ] Enable audit logging
- [ ] Regular security updates
- [ ] Implement monitoring and alerting
- [ ] Use secrets management (AWS Secrets Manager, HashiCorp Vault)
- [ ] Enable database encryption at rest

### 🔍 Security Monitoring

- Monitor failed authentication attempts
- Track unusual API usage patterns
- Set up alerts for security events
- Regular log analysis and threat detection
- Implement intrusion detection systems

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Protocol Buffers compiler (protoc)

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/yourorg/lab-gateway.git
cd lab-gateway
```

2. Install dependencies:
```bash
go mod tidy
```

3. Start development environment:
```bash
docker-compose up -d
```

4. Run the server:
```bash
go run cmd/server/main.go
```

## API Documentation

The gateway provides the following gRPC services:

- `RegisterDevice`: Register a new laboratory instrument
- `StreamData`: Bidirectional streaming for real-time data
- `SendCommand`: Execute commands on devices
- `GetDeviceStatus`: Retrieve device status and health
- `ListDevices`: List registered devices with filtering
- `GetMeasurements`: Query historical measurement data

## Performance Requirements

- Support 1000+ concurrent device connections
- Handle 10,000+ messages per second
- Maintain 99.9% uptime
- Response time under 100ms for API calls

## Security

- Mutual TLS (mTLS) authentication for device connections
- JWT token validation for API clients
- Role-based access control (RBAC)
- Input validation and rate limiting
- Audit logging for security events

## Deployment

### Local Development
```bash
docker-compose up --build
```

### Kubernetes
```bash
kubectl apply -f k8s/
```

### AWS EKS with Helm
```bash
helm install lab-gateway ./helm/lab-gateway \
  --namespace lab-instruments \
  --create-namespace
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.