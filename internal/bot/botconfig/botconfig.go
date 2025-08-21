package botconfig

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
)

type BotConfig struct {
	DiscordToken, BackendHost, BackendPort, CaCertPath, JwtSecret string
}

func FromEnv() (BotConfig, error) {
	var missing []string

	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}

		return val
	}

	cfg := BotConfig{
		DiscordToken: get("SNITCH_DISCORD_TOKEN"),
		BackendHost:  get("SNITCH_BACKEND_HOST"),
		BackendPort:  get("SNITCH_BACKEND_PORT"),
		CaCertPath:   get("CA_CERT_FILE_PATH"),
		JwtSecret:    get("SNITCH_JWT_SECRET"),
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

func (botConfig BotConfig) BackendURL() (*url.URL, error) {
	return url.Parse("https://" + net.JoinHostPort(botConfig.BackendHost, botConfig.BackendPort))
}
