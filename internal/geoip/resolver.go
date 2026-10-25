package geoip

import (
	"net"
	"strings"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

type Resolver struct {
	db *geoip2.Reader
}

var (
	global     *Resolver
	globalOnce sync.Once
)

func Open(path string) (*Resolver, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	db, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	return &Resolver{db: db}, nil
}

func SetDefault(r *Resolver) {
	globalOnce.Do(func() {
		global = r
	})
}

func Default() *Resolver {
	return global
}

func (r *Resolver) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// CountryCode returns ISO 3166-1 alpha-2 or empty string.
func (r *Resolver) CountryCode(ip string) string {
	if r == nil || r.db == nil {
		return ""
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil || parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsUnspecified() {
		return ""
	}
	rec, err := r.db.Country(parsed)
	if err != nil || rec == nil {
		return ""
	}
	return strings.ToUpper(rec.Country.IsoCode)
}

func CountryCode(ip string) string {
	if global == nil {
		return ""
	}
	return global.CountryCode(ip)
}
