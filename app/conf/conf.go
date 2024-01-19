package conf

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	YtApiKeys     []string
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

	ytApiKeys := os.Getenv("YT_API_KEYS")
	if ytApiKeys == "" {
		log.Fatal("You forgot to provide YouTube API keys!")
	}
	config.YtApiKeys = strings.Split(ytApiKeys, ",")

	config.DbPoolSize, _ = strconv.Atoi(os.Getenv("DB_POOL_SIZE"))
	if config.DbPoolSize == 0 {
		config.DbPoolSize = 100
	}

	return config
}
