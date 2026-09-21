package platform

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type TikTok struct{}

func (t *TikTok) Name() string { return "tiktok" }

func (t *TikTok) Match(rawurl string) bool {
	u := strings.ToLower(rawurl)
	return contains(u, "tiktok.com") || contains(u, "vm.tiktok.com")
}

func (t *TikTok) ExtractURLs(rawurl string, cookies string) ([]VideoInfo, error) {
	ytdlpPath := findYtdlpLocate()

	// Try 1: --flat-playlist (profile/playlist extraction)
	entries, err1 := t.tryExtract(ytdlpPath, rawurl, cookies, true)
	if err1 == nil && len(entries) > 0 {
		return entries, nil
	}

	// Try 2: without --flat-playlist (direct URL or single video)
	entries, err2 := t.tryExtract(ytdlpPath, rawurl, cookies, false)
	if err2 == nil && len(entries) > 0 {
		return entries, nil
	}

	// Try 3: If profile extraction failed (embedding disabled), extract channel_id from page
	if err1 != nil && strings.Contains(err1.Error(), "embedding disabled") && isTikTokProfile(rawurl) {
		channelID, err3 := t.extractChannelID(ytdlpPath, rawurl, cookies)
		if err3 == nil && channelID != "" {
			profileURL := fmt.Sprintf("tiktokuser:%s", channelID)
			entries, err4 := t.tryExtract(ytdlpPath, profileURL, cookies, true)
			if err4 == nil && len(entries) > 0 {
				return entries, nil
			}
		}
	}

	// All failed - return detailed error
	errMsg := ""
	if err1 != nil && strings.Contains(err1.Error(), "private") {
		errMsg = "Akun ini private atau embedding disabled.\n"
		errMsg += "Solusi:\n"
		errMsg += "1. Pastikan cookies dari akun yang sudah FOLLOW akun ini\n"
		errMsg += "2. Atau pakai Script dari FetchVid (login di browser dulu)\n"
		errMsg += "3. Atau paste URL video manual"
	} else if err1 != nil {
		errMsg = err1.Error()
	}
	if errMsg == "" && err2 != nil {
		errMsg = err2.Error()
	}
	if errMsg == "" {
		errMsg = "Tidak ada video ditemukan"
	}

	return nil, fmt.Errorf(errMsg)
}

func isTikTokProfile(rawurl string) bool {
	return strings.Contains(rawurl, "/@") && !strings.Contains(rawurl, "/video/")
}

func (t *TikTok) extractChannelID(ytdlpPath, rawurl, cookies string) (string, error) {
	// Extract username from URL
	parts := strings.Split(rawurl, "/@")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid tiktok profile URL")
	}
	username := strings.Split(parts[1], "/")[0]
	username = strings.Split(username, "?")[0]

	// Try downloading single video from user to get channel_id
	// Use yt-dlp with cookies to get user info
	args := []string{
		"--dump-json",
		"--no-warnings", "--ignore-errors",
		"--no-check-certificates", "--geo-bypass",
		"--playlist-items", "1",
	}
	if cookies != "" {
		args = append(args, "--cookies", cookies)
	}
	args = append(args, rawurl)

	cmd := exec.Command(ytdlpPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get channel_id: %s", err.Error())
	}

	// Parse JSON output to find channel_id
	output := string(out)
	re := regexp.MustCompile(`"channel_id"\s*:\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1], nil
	}

	// Try sec_uid pattern
	re2 := regexp.MustCompile(`"sec_uid"\s*:\s*"([^"]+)"`)
	matches2 := re2.FindStringSubmatch(output)
	if len(matches2) >= 2 {
		return matches2[1], nil
	}

	return "", fmt.Errorf("channel_id not found in output")
}

func (t *TikTok) tryExtract(ytdlpPath, rawurl, cookies string, flatPlaylist bool) ([]VideoInfo, error) {
	args := []string{
		"--dump-json",
		"--no-warnings", "--ignore-errors",
		"--no-check-certificates", "--geo-bypass",
	}
	if flatPlaylist {
		args = append(args, "--flat-playlist")
	}
	if cookies != "" {
		args = append(args, "--cookies", cookies)
	}
	args = append(args, rawurl)

	cmd := exec.Command(ytdlpPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp error: %s\n%s", err.Error(), truncateStr(string(out), 300))
	}

	var entries []VideoInfo
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry, err := parseVideoEntry(line, "tiktok")
		if err != nil {
			continue
		}
		if !seen[entry.URL] {
			seen[entry.URL] = true
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (t *TikTok) ConsoleScripts() []ScriptInfo {
	return []ScriptInfo{
		{
			Platform: "tiktok",
			Label:    "TikTok - Videos",
			Script:   "copy([...document.querySelectorAll('a[href*=\"/video/\"]')].map(a=>a.href.split(\"?\")[0]).filter((v,i,a)=>a.indexOf(v)===i).join('\\n'))",
			Desc:     "Semua video dari profile TikTok",
		},
	}
}
