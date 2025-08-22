package backupconfig

import (
	"fmt"
	"os"
	"sort"
)

type BackupConfig struct {
	CaCertFilePath, DatabaseServiceURL, CronSchedule, BackupBucketEndpoint, BackupBucketName, BackupBucketAccessKey, BackupBucketSecretKey, BackupBucketRegion string
}

func FromEnv() (BackupConfig, error) {
	var missing []string

	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}

		return val
	}

	cfg := BackupConfig{
		CaCertFilePath:        get("CA_CERT_FILE_PATH"),
		DatabaseServiceURL:    get("DATABASE_SERVICE_URL"),
		CronSchedule:          get("BACKUP_CRON_SCHEDULE"),
		BackupBucketEndpoint:  get("BACKUP_BUCKET_ENDPOINT"),
		BackupBucketName:      get("BACKUP_BUCKET_NAME"),
		BackupBucketAccessKey: get("BACKUP_BUCKET_ACCESS_KEY"),
		BackupBucketSecretKey: get("BACKUP_BUCKET_SECRET_KEY"),
		BackupBucketRegion:    getOrDefault("BACKUP_BUCKET_REGION", "auto"),
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

func getOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
