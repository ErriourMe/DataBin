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

type ReleaseInfo struct {
	Version string
	URL     string
}

type VersionService struct {
	mu      sync.Mutex
	cached  ReleaseInfo
	expires time.Time
}

func NewVersionService() *VersionService {
	return &VersionService{}
}

func (s *VersionService) Current() ReleaseInfo {
	s.mu.Lock()
	if s.cached.Version != "" && time.Now().Before(s.expires) {
		info := s.cached
		s.mu.Unlock()
		return info
	}
	s.mu.Unlock()

	if info, ok := s.fetchLatestRelease(); ok {
		s.mu.Lock()
		s.cached = info
		s.expires = time.Now().Add(s.cacheTTL())
		s.mu.Unlock()
		return info
	}

	return s.fallbackRelease()
}

func (s *VersionService) fallbackRelease() ReleaseInfo {
	return ReleaseInfo{
		Version: facades.Config().GetString("app.version", "dev"),
		URL:     s.releasesPageURL(),
	}
}

func (s *VersionService) releasesPageURL() string {
	repo := facades.Config().GetString("github.repo", "ErriourMe/DataBin")
	return fmt.Sprintf("https://github.com/%s/releases", repo)
}

func (s *VersionService) cacheTTL() time.Duration {
	mins := facades.Config().GetInt("github.release_cache_mins", 60)
	if mins < 1 {
		mins = 1
	}
	return time.Duration(mins) * time.Minute
}

func (s *VersionService) fetchLatestRelease() (ReleaseInfo, bool) {
	repo := facades.Config().GetString("github.repo", "ErriourMe/DataBin")
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ReleaseInfo{}, false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", facades.Config().GetString("app.name", "DataBin"))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ReleaseInfo{}, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ReleaseInfo{}, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ReleaseInfo{}, false
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ReleaseInfo{}, false
	}

	version := normalizeReleaseTag(payload.TagName)
	if version == "" {
		return ReleaseInfo{}, false
	}

	releaseURL := strings.TrimSpace(payload.HTMLURL)
	if releaseURL == "" {
		releaseURL = s.releasesPageURL()
	}

	return ReleaseInfo{
		Version: version,
		URL:     releaseURL,
	}, true
}

func normalizeReleaseTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	return strings.TrimPrefix(tag, "v")
}
