# SendPulse - Messaging Automation System

SendPulse is a robust messaging automation system that sends **N messages every Y minutes** from a database queue through webhook endpoints.

## 🔧 Requirements

- **Docker** (v20.10+ recommended)
- **Docker Compose** (v2.0+ recommended)
- **Go** (v1.21+ for local development)

## 🔧 Installation Methods

### Method 1: Docker Compose (Easiest)
```bash
# Build and start everything and also seed some example messages to the DB for better dev testing.
make run-dev

# Clean up when done
make clean-dev
```

### Method 2: Manual Setup
```bash
# 1. Start PostgreSQL
make run-dev-db

# 3. Build and run SendPulse
make build
./build/sendpulse server --config ./configs/sendpulse.yaml
```

## 🎮 CLI Usage

### Database Management
```bash
# Initialize database
./build/sendpulse database init

# Run migrations
./build/sendpulse database migrate

# Check migration status
./build/sendpulse database status

# Rollback last migration
./build/sendpulse database rollback

# Generate test data
./build/sendpulse database seed --count 50
```

### Server Management
```bash
# Start server with default config
./build/sendpulse server

# Start with custom config file
./build/sendpulse server --config /path/to/config.yaml

# Get help for any command
./build/sendpulse --help
./build/sendpulse database --help
```

## 📡 API Endpoints

### Message Control
```bash
# Start automatic message processing
curl -X POST http://localhost:8080/api/v1/messaging/start

# Stop automatic message processing
curl -X POST http://localhost:8080/api/v1/messaging/stop

# Check system status
curl http://localhost:8080/api/v1/messaging/status
```

### Message History
```bash
# Get sent messages (paginated)
curl "http://localhost:8080/api/v1/messages?page=1&page_size=10"
```

## ⚙️ Configuration

### Config File (`configs/sendpulse.yaml`)
```yaml
app_name: sendpulse
server:
  address: ":8080"
  mode: dev
database:
  dsn: "postgres://postgres:postgres@localhost:5432/sendpulse?sslmode=disable"
messaging:
  interval: 2m          # Send every 2 minutes
  batch_size: 2         # Send 2 messages per cycle
  max_retries: 3
  retry_delay: 5s
  enabled: true
webhook:
  url: "https://webhook.site/your-endpoint-here"
```

### Environment Variables
```bash
export SENDPULSE_DATABASE_DSN="postgres://user:pass@host:5432/dbname"
export SENDPULSE_WEBHOOK_URL="https://webhook.site/your-endpoint"
export SENDPULSE_MESSAGING_INTERVAL="2m"
export SENDPULSE_MESSAGING_BATCH_SIZE="2"
export SENDPULSE_MESSAGING_ENABLED="true"
```

## 🔨 Available Make Commands

```bash
make build           # Build the binary
make test            # Run all tests
make lint            # Format and vet code
make docker          # Build Docker image
make run-dev         # Start full development stack
make run-dev-db      # Start only PostgreSQL
make run-dev-srv     # Start only SendPulse service
make clean-dev       # Stop and cleanup all containers
make clean           # Remove build artifacts
```

## 🧪 Testing & Development

### Generate Test Data
```bash
# Add 100 random messages for testing
./build/sendpulse database seed --count 100
```

### Monitor Logs
```bash
# Watch all container logs
docker-compose -f containers/composes/dc.dev.yml logs -f

# Watch only SendPulse logs  
docker-compose -f containers/composes/dc.dev.yml logs -f sendpulse
```

### Run Tests
```bash
make test
make lint
```

## 🔍 Troubleshooting

### Common Issues
```bash
# Database connection issues
./build/sendpulse database status

# Check if services are running
docker-compose -f containers/composes/dc.dev.yml ps

# Reset everything
make clean-dev
make run-dev
```

## 📚 Architecture

- **No External Cron**: Custom Go ticker implementation
- **Message Safety**: Database transactions prevent message loss
- **Retry Logic**: Failed messages are retried with exponential backoff
- **Swagger Docs**: Complete API documentation at [`/swagger/`](http://localhost:8080/swagger/index.html)
