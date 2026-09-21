package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Ytdlp struct {
	Path      string
	Cookies   string
	OutputDir string
}

type VideoEntry struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	ID    string `json:"id"`
}

type DownloadProgress struct {
	Percent  float64
	Speed    string
	ETA      string
	Filename string
}

var (
	reProgress     = regexp.MustCompile(`\[download\]\s+([\d.]+)%\s+of\s+~?([\d.]+[KMG]?iB)\s+at\s+([\d.]+[KMG]?iB/s)\s+ETA\s+([\d:]+)`)
	reProgressSimple = regexp.MustCompile(`\[download\]\s+([\d.]+)%`)
)

func NewYtdlp() (*Ytdlp, error) {
	yt := &Ytdlp{}
	if p := findInPath(); p != "" {
		yt.Path = p
		return yt, nil
	}
	if p := findInAppData(); p != "" {
		yt.Path = p
		return yt, nil
	}
	return nil, fmt.Errorf("yt-dlp tidak ditemukan")
}

func findInPath() string {
	for _, name := range []string{"yt-dlp", "yt-dlp.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func findInAppData() string {
	dir := configDir()
	for _, name := range []string{"yt-dlp", "yt-dlp.exe"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
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

func (y *Ytdlp) EnsureDownloaded() error {
	if y.Path != "" {
		ver, err := y.Version()
		if err == nil && len(ver) > 0 {
			return nil
		}
	}

	dir := configDir()
	os.MkdirAll(dir, 0755)

	var url, dest string
	if runtime.GOOS == "windows" {
		dest = filepath.Join(dir, "yt-dlp.exe")
		url = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	} else {
		dest = filepath.Join(dir, "yt-dlp")
		url = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	}

	// Remove partial file if exists
	os.Remove(dest)

	fmt.Printf("Downloading yt-dlp from %s ...\n", url)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("gagal download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d saat download yt-dlp", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("gagal buat file: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		os.Remove(dest)
		return fmt.Errorf("gagal write: %w", err)
	}
	out.Close()

	if runtime.GOOS != "windows" {
		if err := os.Chmod(dest, 0755); err != nil {
			return fmt.Errorf("gagal chmod: %w", err)
		}
	}

	y.Path = dest
	return nil
}

func (y *Ytdlp) Version() (string, error) {
	cmd := exec.Command(y.Path, "--version")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (y *Ytdlp) ExtractPlaylist(url string) ([]VideoEntry, error) {
	args := []string{
		"--flat-playlist", "--dump-json",
		"--no-warnings", "--ignore-errors",
		"--no-check-certificates", "--geo-bypass",
	}
	if y.Cookies != "" {
		args = append(args, "--cookies", y.Cookies)
	}
	args = append(args, url)

	cmd := exec.Command(y.Path, args...)
	hideWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var entries []VideoEntry
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var info struct {
			WebpageURL string `json:"webpage_url"`
			URL        string `json:"url"`
			Title      string `json:"title"`
			ID         string `json:"id"`
		}
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			continue
		}
		u := info.WebpageURL
		if u == "" {
			u = info.URL
		}
		if u != "" {
			entries = append(entries, VideoEntry{
				URL:   u,
				Title: info.Title,
				ID:    info.ID,
			})
		}
	}

	cmd.Wait()
	return entries, nil
}

func (y *Ytdlp) DownloadVideo(url string, jobID, total int, progressCh chan<- DownloadProgress) error {
	filename := "%(title).80s [%(id)s].%(ext)s"
	fullPath := filepath.Join(y.OutputDir, filename)

	args := []string{
		"--no-warnings",
		"--no-check-certificates", "--geo-bypass",
		"--restrict-filenames", "--no-playlist",
		"--no-overwrites", "--continue",
		"--newline",
		"-f", "best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"-o", fullPath,
	}
	if y.Cookies != "" {
		args = append(args, "--cookies", y.Cookies)
	}
	args = append(args, url)

	cmd := exec.Command(y.Path, args...)
	hideWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Parse progress from stdout
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if prog := parseProgress(line); prog != nil && progressCh != nil {
				progressCh <- *prog
			}
		}
	}()

	// Capture stderr
	var stderrBuf strings.Builder
	go io.Copy(&stderrBuf, stderr)

	waitErr := cmd.Wait()
	<-done // Wait for stdout reader to finish

	if waitErr != nil {
		errMsg := stderrBuf.String()
		if errMsg != "" {
			return fmt.Errorf("%s\n%s", waitErr.Error(), truncate(errMsg, 300))
		}
		return waitErr
	}
	return nil
}

func parseProgress(line string) *DownloadProgress {
	if !strings.Contains(line, "[download]") {
		return nil
	}
	matches := reProgress.FindStringSubmatch(line)
	if len(matches) >= 5 {
		pct, _ := strconv.ParseFloat(matches[1], 64)
		return &DownloadProgress{
			Percent: pct / 100,
			Speed:   matches[3],
			ETA:     matches[4],
		}
	}
	// Fallback: simpler format
	if m := reProgressSimple.FindStringSubmatch(line); len(m) > 1 {
		pct, _ := strconv.ParseFloat(m[1], 64)
		return &DownloadProgress{Percent: pct / 100}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func TimeAgo(t time.Time) string {
	d := time.Since(t)
	if d.Hours() > 24*30 {
		return fmt.Sprintf("%.0f bulan", d.Hours()/(24*30))
	}
	if d.Hours() > 24 {
		return fmt.Sprintf("%.0f hari", d.Hours()/24)
	}
	if d.Hours() > 1 {
		return fmt.Sprintf("%.0f jam", d.Hours())
	}
	return fmt.Sprintf("%.0f menit", d.Minutes())
}
