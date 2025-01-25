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

func (libSQLConfig LibSQLConfig) NamespaceURL(namespace string, token string) (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s.%s:%s?authToken=%s", namespace, libSQLConfig.Host, libSQLConfig.Port, token))
}

func (libSQLConfig LibSQLConfig) MetadataDB() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://metadata.%s:%s", libSQLConfig.Host, libSQLConfig.Port))
}

func (libSQLConfig LibSQLConfig) AdminURL() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:%s", libSQLConfig.Host, libSQLConfig.AdminPort))
}

func (libSQLConfig LibSQLConfig) DatabaseURL(token string) (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:%s?authToken=%s", libSQLConfig.Host, libSQLConfig.Port, token))
}
