package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

const (
	versionFile    = "version.txt"
	githubRepo     = "ErriourMe/DataBin"
	defaultVersion = "dev"
	releaseCache   = 60 * time.Minute
)

type ReleaseInfo struct {
	Version       string
	URL           string
	UpdateVersion string
	UpdateURL     string
}

type Service struct {
	once      sync.Once
	installed string

	mu            sync.Mutex
	latestTag     string
	latestHTMLURL string
	latestExpires time.Time
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Current() ReleaseInfo {
	s.once.Do(func() {
		s.installed = readVersionFile()
	})

	info := ReleaseInfo{
		Version: stripVPrefix(s.installed),
		URL:     releaseURL(s.installed),
	}

	if updateTag, updateURL, ok := s.newerReleaseThan(s.installed); ok {
		info.UpdateVersion = stripVPrefix(updateTag)
		info.UpdateURL = updateURL
	}

	return info
}

func (s *Service) newerReleaseThan(installed string) (tag string, htmlURL string, ok bool) {
	latestTag, latestURL, found := s.fetchLatestRelease()
	if !found {
		return "", "", false
	}

	installedSemver := semverTag(installed)
	latestSemver := semverTag(latestTag)
	if !semver.IsValid(installedSemver) || !semver.IsValid(latestSemver) {
		return "", "", false
	}
	if semver.Compare(installedSemver, latestSemver) >= 0 {
		return "", "", false
	}

	if latestURL == "" {
		latestURL = releaseURL(latestTag)
	}
	return latestTag, latestURL, true
}

func (s *Service) fetchLatestRelease() (tagName string, htmlURL string, ok bool) {
	s.mu.Lock()
	if s.latestTag != "" && time.Now().Before(s.latestExpires) {
		tagName, htmlURL = s.latestTag, s.latestHTMLURL
		s.mu.Unlock()
		return tagName, htmlURL, true
	}
	s.mu.Unlock()

	tagName, htmlURL, fetched := s.fetchLatestReleaseFromGitHub()
	if !fetched {
		return "", "", false
	}

	s.mu.Lock()
	s.latestTag = tagName
	s.latestHTMLURL = htmlURL
	s.latestExpires = time.Now().Add(releaseCache)
	s.mu.Unlock()

	return tagName, htmlURL, true
}

func (s *Service) fetchLatestReleaseFromGitHub() (tagName string, htmlURL string, ok bool) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DataBin")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", false
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", false
	}

	tagName = strings.TrimSpace(payload.TagName)
	if tagName == "" {
		return "", "", false
	}

	return tagName, strings.TrimSpace(payload.HTMLURL), true
}

func readVersionFile() string {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return defaultVersion
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return defaultVersion
	}
	return version
}

func stripVPrefix(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func semverTag(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if semver.IsValid(version) {
		return version
	}
	return ""
}

func releaseURL(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || stripVPrefix(version) == defaultVersion {
		return fmt.Sprintf("https://github.com/%s/releases", githubRepo)
	}
	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return fmt.Sprintf("https://github.com/%s/releases/tag/%s", githubRepo, tag)
}
