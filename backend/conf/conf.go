package conf

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	Addr            string
	YouTubeApiKeys  []string
	DSN             string
	MaxCloseDbTries int
}

func ParseConfig(path string) *Config {
	// parse config

	config := &Config{}

	config.Addr = os.Getenv("ADDR")
	if config.Addr == "" {
		config.Addr = ":80"
	}

	config.DSN = os.Getenv("DSN")
	if config.DSN == "" {
		config.DSN = "server.db"
	}

	config.YouTubeApiKeys = strings.Split(os.Getenv("YT_API_KEYS"), ",")
	if len(config.YouTubeApiKeys) == 0 {
		log.Fatal("You forgot to provide YouTube API keys!")
	}

	return config
}
