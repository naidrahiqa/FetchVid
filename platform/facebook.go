package platform

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Facebook struct{}

func (f *Facebook) Name() string { return "facebook" }

func (f *Facebook) Match(rawurl string) bool {
	u := strings.ToLower(rawurl)
	return contains(u, "facebook.com") || contains(u, "fb.com") ||
		contains(u, "fb.watch") || contains(u, "web.facebook.com")
}

func (f *Facebook) ExtractURLs(rawurl string, cookies string) ([]VideoInfo, error) {
	// Check if this is a direct reel/video URL - return it directly
	if isDirectReelOrVideo(rawurl) {
		title := extractTitleFromURL(rawurl)
		return []VideoInfo{{URL: rawurl, Title: title, Source: "facebook"}}, nil
	}

	// Step 1: Resolve share URLs
	resolved := resolveShareURL(rawurl, cookies)
	if resolved != rawurl {
		rawurl = resolved
	}

	// Re-check after resolve
	if isDirectReelOrVideo(rawurl) {
		title := extractTitleFromURL(rawurl)
		return []VideoInfo{{URL: rawurl, Title: title, Source: "facebook"}}, nil
	}

	// Step 2: Extract user ID
	uid := extractFacebookID(rawurl)
	username := extractFacebookUsername(rawurl)

	if uid == "" && username == "" {
		return nil, fmt.Errorf("tidak bisa extract user ID dari URL: %s", rawurl)
	}

	// Step 3: Scrape profile page for reel/video links
	entries, err := scrapeFacebookProfile(uid, username, cookies)
	if err != nil {
		return nil, fmt.Errorf("gagal scrape: %w", err)
	}

	return entries, nil
}

func (f *Facebook) ConsoleScripts() []ScriptInfo {
	return []ScriptInfo{
		{
			Platform: "facebook",
			Label:    "Profile - Reels",
			Script:   "copy([...document.querySelectorAll('a[href*=\"/reel/\"]')].map(a=>a.href.split(\"?\")[0]).filter((v,i,a)=>a.indexOf(v)===i).join('\\n'))",
			Desc:     "Untuk akun personal Facebook",
		},
		{
			Platform: "facebook",
			Label:    "Fanpage - Reels + Video",
			Script:   "copy([...document.querySelectorAll('a[href*=\"/reel/\"], a[href*=\"/video/\"]')].map(a=>a.href.split(\"?\")[0]).filter((v,i,a)=>a.indexOf(v)===i).join('\\n'))",
			Desc:     "Untuk fanpage Facebook",
		},
		{
			Platform: "facebook",
			Label:    "Semua Video",
			Script:   "copy([...document.querySelectorAll('a[href*=\"/reel/\"], a[href*=\"/video/\"], a[href*=\"/watch\"]')].map(a=>a.href.split(\"?\")[0]).filter((v,i,a)=>a.indexOf(v)===i).join('\\n'))",
			Desc:     "Ambil semua link video (reel + video + watch)",
		},
	}
}

// isDirectReelOrVideo checks if URL is a direct reel/video link
func isDirectReelOrVideo(rawurl string) bool {
	lower := strings.ToLower(rawurl)
	return strings.Contains(lower, "/reel/") || strings.Contains(lower, "/video/") ||
		strings.Contains(lower, "/watch/") || strings.Contains(lower, "/watch?v=")
}

// extractTitleFromURL gets a title from a direct reel/video URL
func extractTitleFromURL(rawurl string) string {
	parts := strings.Split(rawurl, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			id := parts[i]
			if strings.Contains(rawurl, "/reel/") {
				return "Reel " + id
			}
			return "Video " + id
		}
	}
	return "Video"
}

// extractFacebookUsername extracts username from URL like /username or /username/reels
func extractFacebookUsername(rawurl string) string {
	parsed, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	path := strings.Trim(parsed.Path, "/")
	if path == "" || strings.Contains(path, "profile.php") || strings.Contains(path, "/pages/") {
		return ""
	}
	parts := strings.Split(path, "/")
	username := parts[0]
	if username == "" || strings.Contains(username, ".") {
		return ""
	}
	return username
}

// resolveShareURL follows redirects from share URLs
func resolveShareURL(rawurl string, cookies string) string {
	if !contains(rawurl, "/share/") {
		return rawurl
	}

	// Try yt-dlp-based resolution first (most reliable)
	// This is done in the engine package; here we just try HTTP redirect
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, _ := http.NewRequest("GET", rawurl, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/125.0.0.0 Safari/537.36")
	if cookies != "" {
		if data, err := os.ReadFile(cookies); err == nil {
			req.Header.Set("Cookie", parseNetscapeCookie(string(data)))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return rawurl
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if finalURL != rawurl {
		// Extract user ID from /people/ redirect
		re := regexp.MustCompile(`/people/[^/]+/(\d+)`)
		if m := re.FindStringSubmatch(finalURL); len(m) > 1 {
			return fmt.Sprintf("https://www.facebook.com/profile.php?id=%s", m[1])
		}
		// Extract from profile.php
		if parsed, err := url.Parse(finalURL); err == nil {
			if id := parsed.Query().Get("id"); id != "" {
				return fmt.Sprintf("https://www.facebook.com/profile.php?id=%s", id)
			}
		}
		return finalURL
	}
	return rawurl
}

// extractFacebookID extracts user ID from a Facebook URL
func extractFacebookID(rawurl string) string {
	parsed, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}

	// profile.php?id=XXX
	if id := parsed.Query().Get("id"); id != "" {
		return id
	}

	// /people/Name/ID
	re := regexp.MustCompile(`/people/[^/]+/(\d+)`)
	if m := re.FindStringSubmatch(parsed.Path); len(m) > 1 {
		return m[1]
	}

	// /pages/Name/ID
	re2 := regexp.MustCompile(`/pages/[^/]+/(\d+)`)
	if m := re2.FindStringSubmatch(parsed.Path); len(m) > 1 {
		return m[1]
	}

	return ""
}

// scrapeFacebookProfile fetches profile page and extracts reel/video URLs
func scrapeFacebookProfile(uid string, username string, cookies string) ([]VideoInfo, error) {
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/125.0.0.0 Safari/537.36",
		"Accept-Language": "en-US,en;q=0.9",
	}

	var urlsToTry []string
	if uid != "" {
		urlsToTry = append(urlsToTry,
			fmt.Sprintf("https://www.facebook.com/profile.php?id=%s&sk=reels_tab", uid),
			fmt.Sprintf("https://www.facebook.com/profile.php?id=%s&sk=videos_tab", uid),
			fmt.Sprintf("https://www.facebook.com/profile.php?id=%s&sk=videos", uid),
			fmt.Sprintf("https://www.facebook.com/profile.php?id=%s", uid),
		)
	}
	if username != "" {
		urlsToTry = append(urlsToTry,
			fmt.Sprintf("https://www.facebook.com/%s/reels", username),
			fmt.Sprintf("https://www.facebook.com/%s/videos", username),
			fmt.Sprintf("https://www.facebook.com/%s", username),
		)
	}

	var allEntries []VideoInfo
	seen := make(map[string]bool)

	for _, scrapeURL := range urlsToTry {
		body, err := fetchWithCookies(scrapeURL, headers, cookies)
		if err != nil {
			continue
		}

		entries := extractFromHTML(body, scrapeURL, seen)
		allEntries = append(allEntries, entries...)

		if len(allEntries) > 0 {
			break
		}
	}

	return allEntries, nil
}

func fetchWithCookies(rawurl string, headers map[string]string, cookiesFile string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", rawurl, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if cookiesFile != "" {
		if data, err := os.ReadFile(cookiesFile); err == nil {
			req.Header.Set("Cookie", parseNetscapeCookie(string(data)))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

// parseNetscapeCookie parses Netscape cookie file format for simple forwarding
func parseNetscapeCookie(data string) string {
	var parts []string
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 7 {
			name := strings.TrimSpace(fields[5])
			value := strings.TrimSpace(fields[6])
			if !strings.HasPrefix(name, "#") {
				parts = append(parts, name+"="+value)
			}
		}
	}
	return strings.Join(parts, "; ")
}

func extractFromHTML(html, baseURL string, seen map[string]bool) []VideoInfo {
	var entries []VideoInfo
	re := regexp.MustCompile(`/reel/(\d+)`)
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		u := fmt.Sprintf("https://www.facebook.com/reel/%s", m[1])
		if !seen[u] {
			seen[u] = true
			entries = append(entries, VideoInfo{URL: u, Title: fmt.Sprintf("Reel %s", m[1]), Source: "facebook"})
		}
	}

	re2 := regexp.MustCompile(`/watch/\?v=(\d+)`)
	for _, m := range re2.FindAllStringSubmatch(html, -1) {
		u := fmt.Sprintf("https://www.facebook.com/watch/?v=%s", m[1])
		if !seen[u] {
			seen[u] = true
			entries = append(entries, VideoInfo{URL: u, Title: fmt.Sprintf("Video %s", m[1]), Source: "facebook"})
		}
	}

	re3 := regexp.MustCompile(`/video/(\d+)`)
	for _, m := range re3.FindAllStringSubmatch(html, -1) {
		u := fmt.Sprintf("https://www.facebook.com/video/%s", m[1])
		if !seen[u] {
			seen[u] = true
			entries = append(entries, VideoInfo{URL: u, Title: fmt.Sprintf("Video %s", m[1]), Source: "facebook"})
		}
	}

	return entries
}

// Ensure package-level unused import suppression
var _ = filepath.Join
