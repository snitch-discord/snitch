package dbconfig

import (
	"fmt"
	"os"
	"sort"
)

type DatabaseConfig struct {
	Host string
	Port string
}

func DatabaseConfigFromEnv() (DatabaseConfig, error) {
	var missing []string

	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		return val
	}

	cfg := DatabaseConfig{
		Host: get("SNITCH_DB_HOST"),
		Port: get("SNITCH_DB_PORT"),
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}