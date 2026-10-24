package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"goravel/app/facades"
)

type VersionService struct {
	mu      sync.Mutex
	cached  string
	expires time.Time
}

func NewVersionService() *VersionService {
	return &VersionService{}
}

func (s *VersionService) Current() string {
	s.mu.Lock()
	if s.cached != "" && time.Now().Before(s.expires) {
		v := s.cached
		s.mu.Unlock()
		return v
	}
	s.mu.Unlock()

	if v := s.fetchLatestRelease(); v != "" {
		s.mu.Lock()
		s.cached = v
		s.expires = time.Now().Add(s.cacheTTL())
		s.mu.Unlock()
		return v
	}

	return facades.Config().GetString("app.version", "dev")
}

func (s *VersionService) cacheTTL() time.Duration {
	mins := facades.Config().GetInt("github.release_cache_mins", 60)
	if mins < 1 {
		mins = 1
	}
	return time.Duration(mins) * time.Minute
}

func (s *VersionService) fetchLatestRelease() string {
	repo := facades.Config().GetString("github.repo", "ErriourMe/DataBin")
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", facades.Config().GetString("app.name", "DataBin"))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	return normalizeReleaseTag(payload.TagName)
}

func normalizeReleaseTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	return strings.TrimPrefix(tag, "v")
}
