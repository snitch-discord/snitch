package dbconfig

import (
	"fmt"
	"net/url"
	"os"
	"sort"
)

type LibSQLConfig struct {
	Host, Port, AdminPort, AuthKey string
}

func LibSQLConfigFromEnv() (LibSQLConfig, error) {
	var missing []string

	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		return val
	}

	cfg := LibSQLConfig{
		Host:      get("LIBSQL_HOST"),
		Port:      get("LIBSQL_PORT"),
		AdminPort: get("LIBSQL_ADMIN_PORT"),
		AuthKey:   get("LIBSQL_AUTH_KEY"),
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

// NamespaceURL returns the URL for a namespace without auth token
func (libSQLConfig LibSQLConfig) NamespaceURL(namespace string) (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s.%s:%s", namespace, libSQLConfig.Host, libSQLConfig.Port))
}

// MetadataURL returns the URL for the metadata database without auth token
func (libSQLConfig LibSQLConfig) MetadataURL() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://metadata.%s:%s", libSQLConfig.Host, libSQLConfig.Port))
}

func (libSQLConfig LibSQLConfig) AdminURL() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:%s", libSQLConfig.Host, libSQLConfig.AdminPort))
}

// DatabaseURL returns the URL for the main database without auth token  
func (libSQLConfig LibSQLConfig) DatabaseURL() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:%s", libSQLConfig.Host, libSQLConfig.Port))
}
