# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Snitch is a Discord bot system for cross-server user reporting and management. It consists of three main services:

- **Database Service** (port 5200): Dedicated database service using embedded libSQL with HTTPS/TLS
- **Backend API Server** (port 4200): Connect RPC service with real-time WebSocket events over HTTPS/TLS  
- **Discord Bot**: Slash command interface with real-time event handling over HTTPS/TLS

All services communicate over **HTTPS with HTTP/2** using **Ed25519 TLS certificates** for security.

## Development Commands

### Setup and Running

```bash
# Start complete development environment with auto-rebuild and TLS
./run.sh

# Individual service builds
go build ./cmd/db
go build ./cmd/backend
go build ./cmd/bot

# Generate protocol buffers
buf generate

# Generate type-safe database queries (run locally before committing)
sqlc generate
```

### TLS Certificate Management

```bash
# Generate TLS certificates (automatic on first run)
./scripts/generate-certs.sh

# Verify existing certificates
./scripts/generate-certs.sh --verify

# Force regenerate all certificates
./scripts/generate-certs.sh --force

# Show certificate help
./scripts/generate-certs.sh --help
```

### Database Migrations

```bash
# Create new metadata migration (for groups, servers tables)
goose -dir internal/db/migrations/metadata create -s migration_name sql

# Create new tenant migration (for users, reports, user_history tables)  
goose -dir internal/db/migrations/tenant create -s migration_name sql

# IMPORTANT: Always use -s flag for sequential numbering (required for sqlc compatibility)

# Migrations run automatically on service startup
# - Metadata migrations apply to the single metadata database
# - Tenant migrations apply to all existing tenant databases

# Manual migration testing (if needed)
goose -dir internal/db/migrations/metadata sqlite3 ./data/metadata.db up
goose -dir internal/db/migrations/tenant sqlite3 ./data/group_<GROUP_ID>.db up

# Generate sqlc after schema changes (required before committing)
sqlc generate
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/backend/service
```

## Architecture

### Multi-Database Design

- **Database Service**: Dedicated service managing all database operations via gRPC
- **Metadata Database**: Group metadata and server associations (embedded libSQL)
- **Group Databases**: Individual database files per Discord server group (embedded libSQL)
- Each group database contains: users, servers, reports, user_history tables

### Key Components

#### Database Service (Connect RPC)

- `DatabaseService`: All database operations via gRPC
  - Metadata operations (groups, servers)
  - Report CRUD operations
  - User history tracking

#### Backend Services (Connect RPC)

- `RegistrarService`: Server group registration (proxies to database service)
- `ReportService`: User reporting operations (proxies to database service)
- `UserHistoryService`: User history operations (proxies to database service)
- `EventService`: Real-time event streaming via Connect RPC

#### Event System

- Connect RPC streaming for real-time notifications
- Event types: ReportCreated, ReportDeleted, UserBanned
- Type-safe protobuf event messages
- Cross-service communication between backend and bot

#### Authentication & Security

- **TLS encryption**: All service-to-service communication encrypted with Ed25519 certificates
- **Certificate validation**: Services validate certificates against internal CA
- **Server ID headers**: Request authorization
- **Group-based multi-tenancy**: Separate database files per Discord server group

### Database Technology

- **libSQL embedded** with SQLite compatibility
- Dedicated database service with gRPC API
- Multi-tenancy through separate database files per group

#### Migration System

- **Goose-based migrations** with embedded migration files
- **Dual migration structure**:
  - `migrations/metadata/` - for metadata database (groups, servers)
  - `migrations/tenant/` - for tenant databases (users, reports, user_history)
- **Sequential migration numbering** (00001_, 00002_, etc.) for sqlc compatibility
- **Automatic migration execution** on service startup
- **Embedded migrations** compiled into binary using Go embed
- **Tenant discovery** - automatically finds and migrates all existing tenant databases

### TLS Security Architecture

#### Certificate Infrastructure

- **Certificate Authority**: Self-signed Ed25519 CA for internal services
- **Certificate Types**:
  - CA Certificate (`certs/ca/ca-cert.pem`): 10-year validity
  - Database Service (`certs/db/cert.pem`): 1-year validity
  - Backend Service (`certs/backend/cert.pem`): 1-year validity  
  - Bot Service (`certs/bot/cert.pem`): 1-year validity (client auth)

#### Subject Alternative Names (SANs)

Each service certificate includes:
- Docker service name (`snitch-db`, `snitch-backend`, `snitch-bot`)
- `localhost` (local development)
- `127.0.0.1` (loopback IP)

#### TLS Configuration

- **Algorithm**: Ed25519 (modern, fast, secure elliptic curve)
- **Protocol**: HTTPS with HTTP/2 support (`h2`, fallback to `http/1.1`)
- **Certificate Validation**: All services validate against internal CA
- **Performance**: HTTP/2 multiplexing, header compression, binary protocol

#### Service Communication Flow

```
Bot Service (HTTPS Client)
    ↓ TLS + Certificate Validation
Backend Service (Port 4200, HTTPS/HTTP2)  
    ↓ TLS + Certificate Validation
Database Service (Port 5200, HTTPS/HTTP2)
```

#### Automated Certificate Management

- **Generation**: Automatic on first `./run.sh` execution
- **Verification**: Automatic validation on each startup
- **Regeneration**: `./scripts/generate-certs.sh --force`
- **Git Exclusion**: Certificates not committed to repository
- **Security**: Private keys have 600 permissions, certificates 644

### Container Images

**Service-specific base image choices:**

- **Database Service**: `gcr.io/distroless/cc-debian12`
  - Requires CGO and GCC runtime libraries for libSQL/SQLite
  - Includes glibc + libgcc_s.so.1 + libstdc++.so.6
  
- **Backend/Bot Services**: `gcr.io/distroless/static-debian12`  
  - Pure Go with static linking (`CGO_ENABLED=0`)
  - No runtime dependencies needed
  - Size: ~2MB (maximum security)

**Security benefit**: Distroless images contain no shell, package manager, or unnecessary utilities, significantly reducing attack surface.

### Configuration

Required environment variables:

- `SNITCH_DISCORD_TOKEN`: Discord bot token
- `SNITCH_DB_HOST/PORT`: Database service connection (for backend service)

## File Structure Notes

### Entry Points

- `/cmd/db/main.go`: Database service
- `/cmd/backend/main.go`: Backend API server
- `/cmd/bot/main.go`: Discord bot service

### Code Organization

- `/internal/db/`: Database service code (embedded libSQL operations)
- `/internal/backend/`: Backend service code (gRPC client, service proxies)
- `/internal/bot/`: Discord bot code (commands, middleware, events)
- `/pkg/proto/`: Protocol buffer definitions and generated code

### Development Tools

- `compose.yml`: Complete Docker development environment (3 services) with TLS certificate mounts
- `buf.yaml`: Protocol buffer configuration
- `db.Containerfile`: Database service container
- `/bruno/`: API testing collection
- `/scripts/generate-certs.sh`: Automated TLS certificate generation and management

## Development Workflow

1. Use `./run.sh` for development (simplified setup with auto-rebuild and automatic TLS certificate management)
2. Protocol buffer changes require `buf generate`
3. Database schema changes require `sqlc generate` after migration files are updated
4. **IMPORTANT**: Run `sqlc generate` locally before committing - CI will verify generated code is up-to-date
5. All services communicate via **HTTPS with HTTP/2** and real-time Connect streaming events
6. **TLS certificates** are automatically generated and validated - no manual intervention required

## Key Patterns

- **Security-first**: All service communication encrypted with Ed25519 TLS certificates
- **Multi-tenancy**: Each Discord server group uses separate database files
- **Event-driven**: Real-time updates via Connect RPC streaming between services
- **Type Safety**: Generated code for gRPC services (Buf) and TLS certificate validation
- **Service-oriented**: Dedicated database service with HTTPS/HTTP2 gRPC API
- **Container-first**: Docker Compose development with watch mode for hot reloading
- **Automated operations**: Certificate generation, validation, and service startup
