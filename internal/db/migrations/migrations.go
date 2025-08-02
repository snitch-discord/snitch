package migrations

import (
	"embed"
)

// Embedded migration files for metadata database
//go:embed metadata
var MetadataMigrations embed.FS

// Embedded migration files for tenant databases
//go:embed tenant
var TenantMigrations embed.FS