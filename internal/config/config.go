package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppName   string
	AppURL    string
	AppSecret string
	AppHost   string
	AppPort   int
	Redis     Redis
	Paste     Paste
	RateLimit RateLimit
	ViewsDir    string
	PublicDir   string
	GeoIPDBPath string
}

type RateLimit struct {
	PostPerMinute int
}

type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Paste struct {
	TTLHours       int
	MaxBytes       int
	KeyLength      int
	EditEnabled    bool
	EditMaxAgeSec   int
	DeleteEnabled   bool
	DeleteMaxAgeSec      int
	MaxTotalMemoryMB     int
}

func Load() Config {
	return Config{
		AppName:   env("APP_NAME", "DataBin"),
		AppURL:    env("APP_URL", ""),
		AppSecret: appSecret(),
		AppHost:   env("APP_HOST", "0.0.0.0"),
		AppPort:   envInt("APP_PORT", 3000),
		ViewsDir:  env("VIEWS_DIR", "resources/views"),
		PublicDir:   env("PUBLIC_DIR", "public"),
		GeoIPDBPath: env("GEOIP_DB_PATH", ""),
		Redis: Redis{
			Host:     env("REDIS_HOST", "127.0.0.1"),
			Port:     envInt("REDIS_PORT", 6379),
			Password: env("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		Paste: Paste{
			TTLHours:      envInt("PASTE_TTL_HOURS", 0),
			MaxBytes:      envInt("PASTE_MAX_BYTES", 2*1024*1024),
			KeyLength:     envInt("PASTE_KEY_LENGTH", 10),
			EditEnabled:   envBool("PASTE_EDIT_ENABLED", true),
			EditMaxAgeSec: envInt("PASTE_EDIT_MAX_AGE_SECONDS", 0),
			DeleteEnabled:   envBool("PASTE_DELETE_ENABLED", true),
			DeleteMaxAgeSec:  envInt("PASTE_DELETE_MAX_AGE_SECONDS", 0),
			MaxTotalMemoryMB: envInt("PASTE_MAX_TOTAL_MEMORY_MB", 0),
		},
		RateLimit: RateLimit{
			PostPerMinute: envInt("RATE_LIMIT_POST_PER_MINUTE", 5),
		},
	}
}

func appSecret() string {
	if v := os.Getenv("APP_SECRET"); v != "" {
		return v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		return v
	}
	return "databin-dev-secret"
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
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
