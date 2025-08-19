package dbconfig

import (
	"fmt"
	"os"
	"sort"
)

type DbConfig struct {
	DbDirPath, CertFilePath, KeyFilePath string
}

func FromEnv() (DbConfig, error) {
	var missing []string

	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}

		return val
	}

	cfg := DbConfig{
		DbDirPath:    get("DB_DIR_PATH"),
		CertFilePath: get("CERT_FILE_PATH"),
		KeyFilePath:  get("KEY_FILE_PATH"),
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}
