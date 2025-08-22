package backendconfig

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
)

type BackendConfig struct {
	CertFilePath, KeyFilePath, CaCertFilePath, DbHost, DbPort, JwtSecret string
}

func FromEnv() (BackendConfig, error) {
	var missing []string

	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}

		return val
	}

	cfg := BackendConfig{
		CertFilePath:   get("CERT_FILE_PATH"),
		KeyFilePath:    get("KEY_FILE_PATH"),
		CaCertFilePath: get("CA_CERT_FILE_PATH"),
		DbHost:         get("SNITCH_DB_HOST"),
		DbPort:         get("SNITCH_DB_PORT"),
		JwtSecret:      get("SNITCH_JWT_SECRET"),
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

func (backendConfig BackendConfig) DbURL() (*url.URL, error) {
	return url.Parse("https://" + net.JoinHostPort(backendConfig.DbHost, backendConfig.DbPort))
}
