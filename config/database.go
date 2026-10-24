package config

import (
	"goravel/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("database", map[string]any{
		// Redis connections
		"redis": map[string]any{
			"default": map[string]any{
				"host":     config.Env("REDIS_HOST", ""),
				"password": config.Env("REDIS_PASSWORD", ""),
				"port":     config.Env("REDIS_PORT", 6379),
				"database": config.Env("REDIS_DB", 0),
			},
		},
	})
}
