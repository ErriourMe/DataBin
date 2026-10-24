package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppName  string
	AppHost  string
	AppPort  int
	Redis    Redis
	Paste    Paste
	ViewsDir string
	PublicDir string
}

type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Paste struct {
	TTLHours  int
	MaxBytes  int
	KeyLength int
}

func Load() Config {
	return Config{
		AppName:   env("APP_NAME", "DataBin"),
		AppHost:   env("APP_HOST", "0.0.0.0"),
		AppPort:   envInt("APP_PORT", 3000),
		ViewsDir:  env("VIEWS_DIR", "resources/views"),
		PublicDir: env("PUBLIC_DIR", "public"),
		Redis: Redis{
			Host:     env("REDIS_HOST", "127.0.0.1"),
			Port:     envInt("REDIS_PORT", 6379),
			Password: env("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		Paste: Paste{
			TTLHours:  envInt("PASTE_TTL_HOURS", 0),
			MaxBytes:  envInt("PASTE_MAX_BYTES", 2*1024*1024),
			KeyLength: envInt("PASTE_KEY_LENGTH", 10),
		},
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
