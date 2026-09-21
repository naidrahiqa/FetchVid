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
)

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
		return VideoInfo{}, nil
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

// findYtdlpLocate returns the path to yt-dlp binary.
// Checks PATH first, then app config dir. Auto-downloads if missing.
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
	// Auto-download yt-dlp
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

	fmt.Printf("Downloading yt-dlp from %s ...\n", url)

	out, err := os.Create(dest)
	if err != nil {
		return ""
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return ""
	}

	if runtime.GOOS != "windows" {
		os.Chmod(dest, 0755)
	}

	return dest
}
