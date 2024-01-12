package conf

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	YtApiKeys     []string
	YtApiMaxTries int
	DSN           string
	DbPoolSize int
}

func ParseConfig() *Config {
	// parse config

	config := &Config{}

	config.DSN = os.Getenv("DSN")
	if config.DSN == "" {
		config.DSN = "server.db"
	}

	config.YtApiKeys = strings.Split(os.Getenv("YT_API_KEYS"), ",")
	if len(config.YtApiKeys) == 0 {
		log.Fatal("You forgot to provide YouTube API keys!")
	}

	config.YtApiMaxTries, _ = strconv.Atoi(os.Getenv("YT_API_MAX_TRIES"))
	if config.YtApiMaxTries == 0 {
		config.YtApiMaxTries = 100
	}

	config.DbPoolSize, _ = strconv.Atoi(os.Getenv("DB_POOL_SIZE"))
	if config.DbPoolSize == 0 {
		config.DbPoolSize = 100
	}

	return config
}
