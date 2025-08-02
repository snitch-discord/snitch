# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Snitch is a Discord bot system for cross-server user reporting and management. It consists of three main services:

- **Database Service** (port 5200): Dedicated database service using embedded libSQL
- **Backend API Server** (port 4200): Connect RPC service with real-time WebSocket events
- **Discord Bot**: Slash command interface with real-time event handling

## Development Commands

### Setup and Running

```bash
# Start complete development environment with auto-rebuild
./run.sh

# Individual service builds
go build ./cmd/db
go build ./cmd/backend
go build ./cmd/bot

# Generate protocol buffers
buf generate

# Generate type-safe database queries
sqlc generate
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

# Generate sqlc after schema changes
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

#### Authentication

- Server ID headers for request authorization
- Group-based multi-tenancy via separate database files

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

- `compose.yml`: Complete Docker development environment (3 services)
- `buf.yaml`: Protocol buffer configuration
- `db.Containerfile`: Database service container
- `/bruno/`: API testing collection

## Development Workflow

1. Use `./run.sh` for development (simplified setup with auto-rebuild)
2. Protocol buffer changes require `buf generate`
3. Database schema changes are handled in the database service
4. All services communicate via gRPC and real-time Connect streaming events

## Key Patterns

- **Multi-tenancy**: Each Discord server group uses separate database files
- **Event-driven**: Real-time updates via Connect RPC streaming between services
- **Type Safety**: Generated code for gRPC services (Buf)
- **Service-oriented**: Dedicated database service with gRPC API
- **Container-first**: Docker Compose development with watch mode for hot reloading
