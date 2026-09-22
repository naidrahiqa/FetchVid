package platform

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type TikTok struct{}

var reChannelID = regexp.MustCompile(`"channel_id"\s*:\s*"([^"]+)"`)

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

	// Wait before retry to avoid 429
	time.Sleep(2 * time.Second)

	// Try 2: without --flat-playlist (direct URL or single video)
	entries, err2 := t.tryExtract(ytdlpPath, rawurl, cookies, false)
	if err2 == nil && len(entries) > 0 {
		return entries, nil
	}

	// All failed - return error
	errMsg := ""
	if err1 != nil {
		errMsg = err1.Error()
	}
	if errMsg == "" && err2 != nil {
		errMsg = err2.Error()
	}
	if errMsg == "" {
		errMsg = "Tidak ada video ditemukan"
	}

	if strings.Contains(errMsg, "secondary user ID") {
		errMsg += "\n\nTip: Jika ini akun dengan embedding disabled, klik 'Paste URLs' lalu paste 1 URL video dari akun tersebut."
	}
	if strings.Contains(errMsg, "429") {
		errMsg += "\n\nTip: Terlalu banyak request. Tunggu 1-2 menit lalu coba lagi."
	}

	return nil, fmt.Errorf("%s", errMsg)
}

func (t *TikTok) ExtractChannelIDFromVideo(videoURL, cookies string) (string, error) {
	ytdlpPath := findYtdlpLocate()

	args := []string{
		"--dump-json",
		"--no-warnings", "--ignore-errors",
		"--no-check-certificates", "--geo-bypass",
	}
	if cookies != "" {
		args = append(args, "--cookies", cookies)
	}
	args = append(args, videoURL)

	var out []byte
	for retry := 0; retry < 3; retry++ {
		cmd := exec.Command(ytdlpPath, args...)
		out, _ = cmd.CombinedOutput()
		matches := reChannelID.FindStringSubmatch(string(out))
		if len(matches) >= 2 {
			return matches[1], nil
		}
		if retry < 2 {
			time.Sleep(3 * time.Second)
		}
	}

	return "", fmt.Errorf("channel_id not found in video data")
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
		if entry.URL != "" && !seen[entry.URL] {
			seen[entry.URL] = true
			entries = append(entries, entry)
		}
	}

	return entries, nil
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
