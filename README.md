# Snitch Discord Bot

A Discord bot system for cross-server user reporting and management. Enables Discord communities to share user reports and maintain synchronized moderation across multiple servers within a group.

## Architecture

- **Backend API Server**: Connect RPC service with real-time event streaming
- **Discord Bot**: Slash command interface with live event notifications
- **Multi-tenant Database**: LibSQL with isolated namespaces per server group

## Features

### 🔗 **Server Group Management**

- Create and join server groups for shared moderation
- Isolated data per group with secure multi-tenancy

### 📝 **User Reporting System**

- Report users with detailed information
- List and manage reports across all servers in your group
- Delete reports when resolved

### 👤 **User History Tracking**

- Track username/display name changes
- View user history across the server group

### ⚡ **Real-time Events**

- Live notifications for new reports
- Real-time updates when reports are deleted
- Event streaming between backend and bot

## Development

### Quick Start

```bash
# Start development environment with auto-rebuild
./run.sh

# Individual builds
go build ./cmd/backend
go build ./cmd/bot

# Generate code
buf generate    # Protocol buffers
sqlc generate   # Database queries
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific service tests
go test ./internal/backend/service
go test ./internal/bot/events
```

## Commands

### `/register`

- **`/register group create <name>`** - Create a new server group
- **`/register group join <code>`** - Join an existing server group

### `/report`

- **`/report new <user> <reason>`** - Report a user
- **`/report list [user] [reporter]`** - List reports with optional filters
- **`/report delete <report-id>`** - Delete a report

### `/user`

- **`/user history <user>`** - View user's name change history

## Configuration

Required environment variables:

```bash
SNITCH_DISCORD_TOKEN=your_discord_bot_token
LIBSQL_HOST=your_libsql_host
LIBSQL_PORT=443
LIBSQL_AUTH_KEY=base64_encoded_ed25519_private_key
PUBLIC_KEY=base64_encoded_ed25519_public_key
```

## Tech Stack

- **Language**: Go 1.24+
- **RPC**: Connect (gRPC-compatible)
- **Database**: LibSQL (Turso) with SQLite compatibility
- **Discord**: DiscordGo library
- **Auth**: Ed25519 JWT tokens
- **Code Gen**: Protocol Buffers (Buf) + SQLC

## Key Patterns

- **Multi-tenancy**: Isolated database namespaces per Discord server group
- **Event-driven**: Real-time updates via Connect RPC streaming
- **Type Safety**: Generated code for database queries and gRPC services
- **Middleware**: Comprehensive middleware chain for logging, auth, and error handling
