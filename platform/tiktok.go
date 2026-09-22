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

	var lastErr string
	maxAttempts := 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(3+attempt*2) * time.Second)
		}

		flat := attempt == 0
		entries, errMsg := t.runExtract(ytdlpPath, rawurl, cookies, flat)
		if errMsg != "" {
			lastErr = errMsg
		}
		if len(entries) > 0 {
			return entries, nil
		}
	}

	if lastErr != "" {
		return nil, fmt.Errorf("%s", lastErr)
	}
	return nil, fmt.Errorf("Tidak ada video ditemukan. Kemungkinan rate-limited atau akun private.")
}

func (t *TikTok) runExtract(ytdlpPath, rawurl, cookies string, flatPlaylist bool) ([]VideoInfo, string) {
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

	output := strings.TrimSpace(string(out))

	var entries []VideoInfo
	seen := make(map[string]bool)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry, perr := parseVideoEntry(line, "tiktok")
		if perr != nil {
			continue
		}
		if entry.URL != "" && !seen[entry.URL] {
			seen[entry.URL] = true
			entries = append(entries, entry)
		}
	}

	if len(entries) > 0 {
		return entries, ""
	}

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	if output != "" {
		errMsg = truncateStr(output, 300)
	}
	if errMsg == "" {
		errMsg = "Tidak ada video ditemukan"
	}

	msg := "yt-dlp: " + errMsg
	if strings.Contains(msg, "secondary user ID") {
		msg += "\n\nTip: Akun ini mematikan embedding. Coba 'Paste Video URL' dengan 1 video dari akun tersebut."
	}
	if strings.Contains(msg, "439") || strings.Contains(msg, "429") {
		msg = "Rate-limited oleh TikTok (429/439).\n\nTip: Tunggu 5-10 menit lalu coba lagi, jangan terlalu sering."
	}
	return nil, msg
}

// ExtractVideoInfo extracts a single video's info and its channel_id in one request.
// Returns the VideoInfo (for immediate result) plus the channel_id for profile extraction.
func (t *TikTok) ExtractVideoInfo(videoURL, cookies string) (VideoInfo, string, error) {
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

	maxRetries := 2
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			time.Sleep(5 * time.Second)
		}

		cmd := exec.Command(ytdlpPath, args...)
		out, _ := cmd.CombinedOutput()

		output := strings.TrimSpace(string(out))
		if output == "" {
			continue
		}

		channelID := ""
		vi := VideoInfo{URL: videoURL, Source: "tiktok"}

		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "{") {
				continue
			}
			if cid, ok := t.channelIDFromLine(line); ok {
				channelID = cid
			}
			if parsed, err := parseVideoEntry(line, "tiktok"); err == nil && parsed.URL != "" {
				vi = parsed
			}
		}

		if channelID != "" {
			// Prefer the direct video URL as webpage url so the user always gets this video
			if vi.URL == "" || !contains(vi.URL, "/video/") {
				vi.URL = videoURL
			}
			if vi.Title == "" {
				vi.Title = "TikTok video"
			}
			return vi, channelID, nil
		}
	}

	return VideoInfo{}, "", fmt.Errorf("channel_id tidak ditemukan. Kemungkinan akun ini mematikan embedding atau rate-limited.")
}

func (t *TikTok) channelIDFromLine(line string) (string, bool) {
	matches := reChannelID.FindStringSubmatch(line)
	if len(matches) >= 2 {
		return matches[1], true
	}
	return "", false
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
