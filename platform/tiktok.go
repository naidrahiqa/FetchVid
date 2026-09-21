package platform

import (
	"fmt"
	"os/exec"
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

	// Both failed - return detailed error
	errMsg := "yt-dlp gagal extract TikTok URLs"
	if err1 != nil {
		errMsg += "\n" + err1.Error()
	}
	if err2 != nil {
		errMsg += "\n" + err2.Error()
	}
	errMsg += "\n\nTip: Login TikTok di browser, buka profile, pakai Script dari FetchVid, lalu Paste URLs."

	return nil, fmt.Errorf(errMsg)
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
