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
	"strconv"
	"strings"
	"syscall"
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
	dir := filepath.Join(os.Getenv("APPDATA"), "FetchVid", "bin")
	for _, name := range []string{"yt-dlp.exe", "yt-dlp"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (y *Ytdlp) EnsureDownloaded() error {
	if y.Path != "" {
		// Check if version is recent
		ver, err := y.Version()
		if err == nil && len(ver) > 0 {
			return nil
		}
	}

	dir := filepath.Join(os.Getenv("APPDATA"), "FetchVid", "bin")
	os.MkdirAll(dir, 0755)
	dest := filepath.Join(dir, "yt-dlp.exe")

	url := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	fmt.Printf("Downloading yt-dlp from %s ...\n", url)

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("gagal buat file: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("gagal download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d saat download yt-dlp", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("gagal write: %w", err)
	}

	y.Path = dest
	return nil
}

func (y *Ytdlp) Version() (string, error) {
	cmd := exec.Command(y.Path, "--version")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ExtractPlaylist extracts video URLs from a profile/playlist URL using --flat-playlist
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
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		// Try to parse stderr for useful info
		if stderr, ok := err.(*exec.ExitError); ok {
			errMsg := string(stderr.Stderr)
			return nil, parseYtdlpError(errMsg, url)
		}
		return nil, err
	}

	var entries []VideoEntry
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
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
	return entries, nil
}

var reYtdlpError = regexp.MustCompile(`/people/[^/]+/(\d+)`)

func parseYtdlpError(stderr string, originalURL string) error {
	// Try to extract user ID from Facebook /people/ redirect
	if matches := reYtdlpError.FindStringSubmatch(stderr); len(matches) > 1 {
		uid := matches[1]
		return &ResolvedURLError{
			ResolvedURL: fmt.Sprintf("https://www.facebook.com/profile.php?id=%s", uid),
			UID:         uid,
		}
	}
	return fmt.Errorf("yt-dlp: %s", truncate(stderr, 200))
}

type ResolvedURLError struct {
	ResolvedURL string
	UID         string
}

func (e *ResolvedURLError) Error() string {
	return fmt.Sprintf("resolved to %s", e.ResolvedURL)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// DownloadVideo downloads a single video and sends progress updates to progressCh
func (y *Ytdlp) DownloadVideo(url string, jobID, total int, progressCh chan<- DownloadProgress) error {
	filename := fmt.Sprintf("reel_%03d_%%(title)s_%%(id)s.%%(ext)s", jobID)
	fullPath := filepath.Join(y.OutputDir, filename)

	args := []string{
		"--no-warnings", "--ignore-errors",
		"--no-check-certificates", "--geo-bypass",
		"--restrict-filenames", "--no-playlist",
		"--no-overwrites", "--continue",
		"--newline",
		"-o", fullPath,
	}
	if y.Cookies != "" {
		args = append(args, "--cookies", y.Cookies)
	}
	args = append(args, url)

	cmd := exec.Command(y.Path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

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
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if prog := parseProgress(line); prog != nil && progressCh != nil {
				progressCh <- *prog
			}
		}
	}()

	// Read stderr for errors
	go io.Copy(io.Discard, stderr)

	return cmd.Wait()
}

var reProgress = regexp.MustCompile(`\[download\]\s+([\d.]+)%\s+of\s+~?([\d.]+[KMG]?iB])\s+at\s+([\d.]+[KMG]?iB/s])\s+ETA\s+([\d:]+)`)

func parseProgress(line string) *DownloadProgress {
	if !strings.Contains(line, "[download]") {
		return nil
	}
	matches := reProgress.FindStringSubmatch(line)
	if len(matches) < 5 {
		// Try simpler format: [download] 100% of ...
		reSimple := regexp.MustCompile(`\[download\]\s+([\d.]+)%`)
		if m := reSimple.FindStringSubmatch(line); len(m) > 1 {
			pct, _ := strconv.ParseFloat(m[1], 64)
			return &DownloadProgress{Percent: pct / 100}
		}
		return nil
	}
	pct, _ := strconv.ParseFloat(matches[1], 64)
	return &DownloadProgress{
		Percent: pct / 100,
		Speed:   matches[3],
		ETA:     matches[4],
	}
}

// FormatFileSize returns human-readable file size
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

// Time ago
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
