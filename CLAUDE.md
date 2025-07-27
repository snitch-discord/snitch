# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Snitch is a Discord bot system for cross-server user reporting and management. It consists of two main services:

- **Backend API Server** (port 4200): Connect RPC service with real-time WebSocket events
- **Discord Bot**: Slash command interface with real-time event handling

## Development Commands

### Setup and Running

```bash
# Start complete development environment with auto-rebuild
./run.sh

# Individual service builds
go build ./cmd/backend
go build ./cmd/bot

# Generate protocol buffers
buf generate

# Generate type-safe database queries
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

- **Metadata Database**: Group metadata and server associations
- **Group Databases**: Individual namespaced databases per Discord server group
- Each group database contains: users, servers, reports, user_history tables

### Key Components

#### Backend Services (Connect RPC)

- `RegistrarService`: Server group registration
- `ReportService`: User reporting CRUD operations
- `UserHistoryService`: User history tracking
- `EventService`: Real-time event streaming via Connect RPC

#### Event System

- Connect RPC streaming for real-time notifications
- Event types: ReportCreated, ReportDeleted, UserBanned
- Type-safe protobuf event messages
- Cross-service communication between backend and bot

#### Authentication

- Ed25519 JWT tokens for service authentication
- Server ID headers for request authorization
- Group-based multi-tenancy via LibSQL namespaces

### Database Technology

- **LibSQL (Turso)** with SQLite compatibility
- **SQLC** for type-safe query generation
- Multi-tenancy through database namespaces

### Configuration

Required environment variables:

- `SNITCH_DISCORD_TOKEN`: Discord bot token
- `LIBSQL_HOST/PORT`: Database connection
- `LIBSQL_AUTH_KEY`: Ed25519 private key (base64)
- `PUBLIC_KEY`: Ed25519 public key (base64)

## File Structure Notes

### Entry Points

- `/cmd/backend/main.go`: Backend API server
- `/cmd/bot/main.go`: Discord bot service

### Code Organization

- `/internal/backend/`: Backend service code (service, jwt, metadata, etc.)
- `/internal/bot/`: Discord bot code (commands, middleware, events)
- `/pkg/proto/`: Protocol buffer definitions and generated code

### Development Tools

- `compose.yml`: Complete Docker development environment
- `buf.yaml`: Protocol buffer configuration
- `sqlc.yml`: Database code generation
- `/bruno/`: API testing collection

## Development Workflow

1. Use `./run.sh` for development (includes key generation and auto-rebuild)
2. Protocol buffer changes require `buf generate`
3. Database schema changes require `sqlc generate`
4. Both services communicate via gRPC and real-time Connect streaming events

## Key Patterns

- **Multi-tenancy**: Each Discord server group uses isolated database namespace
- **Event-driven**: Real-time updates via Connect RPC streaming between services
- **Type Safety**: Generated code for both database queries (SQLC) and gRPC services (Buf)
- **Container-first**: Docker Compose development with watch mode for hot reloading
