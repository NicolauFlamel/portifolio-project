# Docker Deployment Guide

This guide explains how to deploy the backend API using Docker with three separate instances, each configured with its own admin user and read-only access to other organizations' channels.

## Architecture Overview

The deployment consists of three backend instances:

1. **backend-union** (port 3000)
   - Admin access: `Admin@union.gov.br` on union-channel
   - Read access: `User1@state.gov.br` and `User1@region.gov.br` on other channels

2. **backend-state** (port 3001)
   - Admin access: `Admin@state.gov.br` on state-channel
   - Read access: `User1@union.gov.br` and `User1@region.gov.br` on other channels

3. **backend-region** (port 3002)
   - Admin access: `Admin@region.gov.br` on region-channel
   - Read access: `User1@union.gov.br` and `User1@state.gov.br` on other channels

Each instance uses a different configuration file that specifies which user credentials to use for each channel.

## Prerequisites

1. **Blockchain Network Running**: The Hyperledger Fabric network must be running first
   ```bash
   cd ../gov-ledger/network
   ./start-network.sh
   ```

2. **Docker Network**: Ensure the `gov-spending-network` Docker network exists
   ```bash
   docker network inspect gov-spending-network
   ```
   If it doesn't exist, it will be created automatically when you start the blockchain network.

3. **Crypto-config**: All user certificates must be generated
   - Admin users: `Admin@{org}.gov.br`
   - Read users: `User1@{org}.gov.br`

   These are located in: `../gov-ledger/network/crypto-config/peerOrganizations/{org}.gov.br/users/`

## Building the Docker Image

Build the backend Docker image:

```bash
cd backend
docker-compose build
```

This will create a multi-stage Docker image:
- **Build stage**: Compiles the Go application
- **Runtime stage**: Creates a minimal Alpine Linux image with only the binary

## Starting the Backend Instances

Start all three backend instances:

```bash
docker-compose up -d
```

This will start:
- `backend-union` on port 3000
- `backend-state` on port 3001
- `backend-region` on port 3002

Check the status:

```bash
docker-compose ps
```

View logs:

```bash
# All instances
docker-compose logs -f

# Specific instance
docker-compose logs -f backend-union
docker-compose logs -f backend-state
docker-compose logs -f backend-region
```

## Health Checks

Each instance has a health check endpoint:

```bash
# Union instance
curl http://localhost:3000/health

# State instance
curl http://localhost:3001/health

# Region instance
curl http://localhost:3002/health
```

Expected response:
```json
{"status":"healthy"}
```

## Configuration Details

### Network Path

The blockchain network files are mounted as read-only volumes:
```yaml
volumes:
  - ../gov-ledger/network:/network:ro
```

The backend expects the network path at `/network` inside the container.

### Peer Endpoints

In Docker, the backend connects to peers using service names instead of localhost:

- Union: `peer0.union.gov.br:7051`
- State: `peer0.state.gov.br:9051`
- Region: `peer0.region.gov.br:11051`

### User Credentials

Each instance configuration file specifies which user to use per channel:

**config-union.yaml**:
```yaml
channels:
  union:
    user_name: "Admin"  # Full admin access to own channel
  state:
    user_name: "User1"  # Read-only access to state channel
  region:
    user_name: "User1"  # Read-only access to region channel
```

## API Usage Examples

### Union Instance (Admin on union-channel)

Create a document on union-channel:
```bash
curl -X POST http://localhost:3000/api/union/documents \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc-001",
    "title": "Federal Budget 2024",
    "amount": 1000000,
    "category": "education"
  }'
```

Query documents from state-channel (read-only):
```bash
curl http://localhost:3000/api/state/documents
```

### State Instance (Admin on state-channel)

Create a document on state-channel:
```bash
curl -X POST http://localhost:3001/api/state/documents \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc-002",
    "title": "State Infrastructure",
    "amount": 500000,
    "category": "infrastructure"
  }'
```

### Region Instance (Admin on region-channel)

Create a document on region-channel:
```bash
curl -X POST http://localhost:3002/api/region/documents \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc-003",
    "title": "Municipal Services",
    "amount": 100000,
    "category": "public_services"
  }'
```

## Stopping the Backend

Stop all instances:

```bash
docker-compose down
```

Stop and remove volumes (if any):

```bash
docker-compose down -v
```

## Troubleshooting

### Connection Issues

If the backend can't connect to peers:

1. **Check network connectivity**:
   ```bash
   docker network inspect gov-spending-network
   ```
   Ensure all containers are on the same network.

2. **Verify peer containers are running**:
   ```bash
   docker ps | grep peer
   ```

3. **Check peer endpoints in config files**:
   ```bash
   cat config-union.yaml | grep peer_endpoint
   ```

### Certificate Issues

If you get certificate errors:

1. **Verify crypto-config exists**:
   ```bash
   ls -la ../gov-ledger/network/crypto-config/peerOrganizations/
   ```

2. **Check user directories**:
   ```bash
   # Union Admin
   ls ../gov-ledger/network/crypto-config/peerOrganizations/union.gov.br/users/Admin@union.gov.br/msp/

   # State User1
   ls ../gov-ledger/network/crypto-config/peerOrganizations/state.gov.br/users/User1@state.gov.br/msp/
   ```

3. **Rebuild the network** if certificates are missing:
   ```bash
   cd ../gov-ledger/network
   ./teardown.sh
   ./start-network.sh
   ```

### Permission Errors

If you get permission denied errors:

1. **Check file permissions** on crypto-config:
   ```bash
   chmod -R 755 ../gov-ledger/network/crypto-config
   ```

2. **Run with proper user**:
   The Docker containers run as root by default, which should have read access to mounted volumes.

### Logs Not Showing

If logs are not appearing:

1. **Increase log level**:
   Edit the config files and set:
   ```yaml
   logging:
     level: "debug"
   ```

2. **Rebuild and restart**:
   ```bash
   docker-compose down
   docker-compose up -d
   docker-compose logs -f
   ```

## Environment Variables

You can override configuration using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | 3000 |
| `GIN_MODE` | Gin mode (debug/release) | release |
| `FABRIC_NETWORK_PATH` | Path to network directory | /network |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | info |

Example with custom environment:
```bash
docker-compose up -d -e LOG_LEVEL=debug
```

## Production Considerations

For production deployment:

1. **Use TLS for API**: Add HTTPS support with proper certificates
2. **Add authentication**: Implement JWT or OAuth for API endpoints
3. **Resource limits**: Set memory and CPU limits in docker-compose.yml
4. **Monitoring**: Add Prometheus metrics and health monitoring
5. **Backup**: Regular backups of the blockchain state
6. **Log aggregation**: Use ELK stack or similar for centralized logging
7. **Secrets management**: Use Docker secrets or external secret manager

Example with resource limits:
```yaml
services:
  backend-union:
    # ... other config ...
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
```

## Advanced Configuration

### Custom User Configuration

To use a different user (not Admin or User1):

1. Generate new user certificates using Fabric CA or cryptogen
2. Update the config file:
   ```yaml
   channels:
     union:
       user_name: "CustomUser"
   ```

### Single Channel Access

To limit an instance to only one channel, remove other channels from the config file:

```yaml
fabric:
  channels:
    union:
      # only union channel configuration
```

### External Network Path

To use a network path outside the project:

```yaml
volumes:
  - /path/to/network:/network:ro
```

## Next Steps

- Set up reverse proxy (nginx/traefik) for load balancing
- Implement API rate limiting
- Add integration tests for Docker deployment
- Configure continuous deployment pipeline
