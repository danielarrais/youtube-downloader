package main

import (
	"net/url"
	"strings"
)

type Platform string

const (
	PlatformYouTube   Platform = "youtube"
	PlatformX         Platform = "x"
	PlatformInstagram Platform = "instagram"
	PlatformTikTok    Platform = "tiktok"
)

var platformDomains = map[Platform][]string{
	PlatformYouTube:   {"youtube.com", "youtu.be", "youtube-nocookie.com"},
	PlatformX:         {"x.com", "twitter.com"},
	PlatformInstagram: {"instagram.com"},
	PlatformTikTok:    {"tiktok.com"},
}

// detectPlatform identifies the source of a link by its domain. Unknown links
// are treated as YouTube to preserve the original behavior.
func detectPlatform(rawURL string) Platform {
	value := strings.TrimSpace(rawURL)
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return PlatformYouTube
	}
	host := strings.ToLower(parsed.Hostname())
	for platform, domains := range platformDomains {
		for _, domain := range domains {
			if host == domain || strings.HasSuffix(host, "."+domain) {
				return platform
			}
		}
	}
	return PlatformYouTube
}

func isYouTubeURL(rawURL string) bool {
	return detectPlatform(rawURL) == PlatformYouTube
}
