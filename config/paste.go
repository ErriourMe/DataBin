package config

import (
	"goravel/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("paste", map[string]any{
		// TTL in hours; 0 means paste never expires.
		"ttl_hours": config.Env("PASTE_TTL_HOURS", 0),
		// Maximum paste size in bytes.
		"max_bytes": config.Env("PASTE_MAX_BYTES", 2*1024*1024),
		"key_length": config.Env("PASTE_KEY_LENGTH", 10),
	})
}
