package conf

import (
	"os"
	"strconv"
)

type Config struct {
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

	config.DbPoolSize, _ = strconv.Atoi(os.Getenv("DB_POOL_SIZE"))
	if config.DbPoolSize == 0 {
		config.DbPoolSize = 100
	}

	return config
}
