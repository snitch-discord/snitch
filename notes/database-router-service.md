# Database Router Service Design

## Overview

A drop-in replacement for sqld that provides database-level isolation using LibSQL file-based databases with path-based routing. This service runs as a separate binary in its own container, implementing the LibSQL HTTP protocol that Go clients expect.

## Implementation Approach

Based on research of sqld (Rust) and go-libsql (Go) codebases:

### Key Insights
1. **LibSQL HTTP Protocol**: sqld uses JSON-based HTTP protocol for SQL operations
2. **Admin API**: `/v1/namespaces/{namespace}/create` for database lifecycle management  
3. **Query API**: `POST /` with JSON payloads containing SQL statements
4. **File-based Storage**: Each namespace maps to SQLite files on disk
5. **Go Client Expectations**: go-libsql uses CGo but expects standard HTTP protocol

### Router Architecture
The router service will:
1. **Parse tenant from path**: `/tenant/{groupID}/...` → extract `groupID`
2. **Route to SQLite files**: Map `groupID` → `./data/{groupID}.db`
3. **Implement LibSQL HTTP protocol**: Compatible JSON request/response format
4. **Maintain existing DNS routing**: Keep current subdomain logic for backward compatibility
5. **Run as separate service**: Independent binary with own container

## Current Architecture Analysis

### Connection Patterns

Based on `internal/backend/group/db.go:32` and `internal/backend/dbconfig/libsqlconfig.go`:

**Current URL pattern:**

```text
http://{namespace}.{host}:{port}?authToken={token}
```

**Examples:**

- Namespace: `http://groupid123.snitch:8080?authToken=xyz`
- Metadata: `http://metadata.snitch:8080?authToken=xyz`

### Admin API Patterns

From `internal/backend/libsqladmin/libsqladmin.go`:

**Current admin endpoints:**

- Create: `POST {adminURL}/v1/namespaces/{name}/create`
- Check: `GET {adminURL}/v1/namespaces/{name}/config`

## Proposed Router Service

### Connection Management Strategy

The key insight: **LibSQL clients use standard SQL driver interface** (`sql.Open("libsql", connectionString)`).

**Router service will:**

1. Accept connections on sqld's port (8080)
2. Extract tenant ID from URL path: `/tenant/{groupid}/...`
3. Route to appropriate SQLite file: `./data/{groupid}.db`
4. Maintain connection pool per tenant

### Path-Based Routing Design

**Path-based routing (chosen approach):**

```text
Client: http://snitch-libsql-server:8080/tenant/group-uuid?authToken=xyz
Router: Extracts 'group-uuid' and routes to ./data/group-uuid.db
```

**URL Pattern:**

- Metadata: `http://snitch-libsql-server:8080/tenant/metadata?authToken=xyz`
- Group: `http://snitch-libsql-server:8080/tenant/{groupID}?authToken=xyz`

### Service Architecture

```go
type DatabaseRouter struct {
    connections map[string]*sql.DB  // groupID -> LibSQL connection
    dataDir     string              // "./data/"
    mu          sync.RWMutex
}

func (dr *DatabaseRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Extract groupID from URL path: "/tenant/group-uuid/v1/execute"
    groupID := extractGroupIDFromPath(r.URL.Path)
    if groupID == "" {
        http.Error(w, "Invalid tenant path", http.StatusBadRequest)
        return
    }

    // Get or create connection for this tenant
    db := dr.getConnection(groupID)

    // Strip tenant prefix and proxy to LibSQL file
    // "/tenant/group-uuid/v1/execute" -> "/v1/execute"
    proxyPath := stripTenantPrefix(r.URL.Path, groupID)
    proxyToLibSQL(w, r, db, proxyPath)
}

func extractGroupIDFromPath(path string) string {
    // Parse "/tenant/{groupID}/..." -> groupID
    parts := strings.Split(strings.Trim(path, "/"), "/")
    if len(parts) >= 2 && parts[0] == "tenant" {
        return parts[1]
    }
    return ""
}

func stripTenantPrefix(path, groupID string) string {
    // "/tenant/group-uuid/v1/execute" -> "/v1/execute"
    prefix := fmt.Sprintf("/tenant/%s", groupID)
    return strings.TrimPrefix(path, prefix)
}

// proxyToLibSQL forwards the HTTP request to the LibSQL file-based database
func proxyToLibSQL(w http.ResponseWriter, r *http.Request, db *sql.DB, proxyPath string) {
    switch {
    case strings.HasPrefix(proxyPath, "/v1/namespaces"):
        // Handle admin endpoints (create/delete databases)
        handleAdminRequest(w, r, proxyPath)
    case proxyPath == "/" || proxyPath == "":
        // Handle SQL query requests (JSON protocol)
        handleQueryRequest(w, r, db)
    default:
        http.Error(w, "Unknown endpoint", http.StatusNotFound)
    }
}

func handleQueryRequest(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    // Parse LibSQL JSON request format
    // Expected format: {"statements": [{"q": "SELECT * FROM users"}]}
    var req struct {
        Statements []struct {
            Q string `json:"q"`
        } `json:"statements"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Execute SQL statements
    var results []map[string]interface{}
    for _, stmt := range req.Statements {
        rows, err := db.Query(stmt.Q)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        // Convert rows to JSON format expected by LibSQL clients
        result := convertRowsToLibSQLFormat(rows)
        results = append(results, result)
        rows.Close()
    }
    
    // Return LibSQL JSON response format
    response := map[string]interface{}{
        "results": results,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func handleAdminRequest(w http.ResponseWriter, r *http.Request, proxyPath string) {
    // Handle namespace creation/deletion
    // Implementation depends on admin endpoint requirements
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

### Connection Pool Management

```go
func (dr *DatabaseRouter) getConnection(groupID string) *sql.DB {
    dr.mu.RLock()
    if conn, exists := dr.connections[groupID]; exists {
        dr.mu.RUnlock()
        return conn
    }
    dr.mu.RUnlock()

    dr.mu.Lock()
    defer dr.mu.Unlock()

    // Double-check pattern
    if conn, exists := dr.connections[groupID]; exists {
        return conn
    }

    // Create new file-based LibSQL connection
    dbPath := filepath.Join(dr.dataDir, groupID+".db")
    conn, err := sql.Open("libsql", "file:"+dbPath)
    if err != nil {
        // Handle error
        return nil
    }

    dr.connections[groupID] = conn
    return conn
}
```

### Admin API Implementation

**Admin endpoints (preserving existing patterns):**

- `POST /v1/namespaces/{groupID}/create` → Create `./data/{groupID}.db`
- `GET /v1/namespaces/{groupID}/config` → Check if `./data/{groupID}.db` exists

### Deployment Strategy

**Swap-in process:**

1. Deploy router service on same port as sqld (8080)
2. Update client connection strings to use path-based routing
3. Admin port can be same service or separate port

**Client Changes Required:**
Update `DatabaseURL()` and `NamespaceURL()` methods in `dbconfig/libsqlconfig.go`:

```go
// Before: http://namespace.host:port
// After:  http://host:port/tenant/namespace

func (libSQLConfig LibSQLConfig) NamespaceURL(namespace string) (*url.URL, error) {
    return url.Parse(fmt.Sprintf("http://%s:%s/tenant/%s",
        libSQLConfig.Host, libSQLConfig.Port, namespace))
}

func (libSQLConfig LibSQLConfig) MetadataURL() (*url.URL, error) {
    return url.Parse(fmt.Sprintf("http://%s:%s/tenant/metadata",
        libSQLConfig.Host, libSQLConfig.Port))
}
```

## Implementation Priority

1. **HTTP server with path-based routing**
2. **LibSQL file connection management**
3. **Admin API for database lifecycle**
4. **Connection pooling and cleanup**
5. **Error handling and logging**

## File Structure

```text
./cmd/database-router/
    main.go
./internal/database-router/
    server.go          # HTTP server and routing
    connections.go     # Connection pool management
    admin.go          # Admin API handlers
    libsql_proxy.go   # LibSQL protocol proxying
```
