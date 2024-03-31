package conf

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	YtApiKey     string
	DSN           string
	DbPoolSize int
	RandomSeed int64
}

func ParseConfig() *Config {
	config := &Config{}

	config.DSN = os.Getenv("DSN")
	if config.DSN == "" {
		config.DSN = "server.db"
	}

	config.YtApiKey = os.Getenv("YT_API_KEY")
	if config.YtApiKey == "" {
		log.Fatal("You forgot to provide YouTube API key!")
	}

	config.DbPoolSize, _ = strconv.Atoi(os.Getenv("DB_POOL_SIZE"))
	if config.DbPoolSize == 0 {
		config.DbPoolSize = 100
	}

	return config
}
