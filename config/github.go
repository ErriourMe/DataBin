package config

import (
	"goravel/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("github", map[string]any{
		"repo":               "ErriourMe/DataBin",
		"release_cache_mins": 60,
	})
}
