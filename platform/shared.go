package platform

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type VideoInfo struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Source string `json:"source"`
}

// parseVideoEntry parses a yt-dlp JSON line into a VideoInfo
func parseVideoEntry(line, source string) (VideoInfo, error) {
	var raw struct {
		WebpageURL string `json:"webpage_url"`
		URL        string `json:"url"`
		Title      string `json:"title"`
		ID         string `json:"id"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return VideoInfo{}, err
	}
	u := raw.WebpageURL
	if u == "" {
		u = raw.URL
	}
	if u == "" {
		return VideoInfo{}, fmt.Errorf("empty URL in entry")
	}
	return VideoInfo{
		URL:    u,
		Title:  raw.Title,
		Source: source,
	}, nil
}

func configDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "FetchVid", "bin")
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "FetchVid", "bin")
}

func findYtdlpLocate() string {
	for _, name := range []string{"yt-dlp", "yt-dlp.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	dir := configDir()
	for _, name := range []string{"yt-dlp", "yt-dlp.exe"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	p := downloadYtdlp(dir)
	if p != "" {
		return p
	}
	return "yt-dlp"
}

func downloadYtdlp(dir string) string {
	os.MkdirAll(dir, 0755)

	var url, dest string
	if runtime.GOOS == "windows" {
		dest = filepath.Join(dir, "yt-dlp.exe")
		url = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	} else {
		dest = filepath.Join(dir, "yt-dlp")
		url = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	}

	os.Remove(dest)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	out, err := os.Create(dest)
	if err != nil {
		return ""
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(dest)
		return ""
	}
	out.Close()

	if runtime.GOOS != "windows" {
		os.Chmod(dest, 0755)
	}

	return dest
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func containsAny(s, substrs string) bool {
	for _, c := range substrs {
		if strings.ContainsRune(s, c) {
			return true
		}
	}
	return false
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
