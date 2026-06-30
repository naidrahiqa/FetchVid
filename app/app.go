package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/naidrahiqa/FetchVid/engine"
	"github.com/naidrahiqa/FetchVid/platform"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	queue      *engine.Queue
	ytdlp      *engine.Ytdlp
	settings   *Settings
	mu         sync.Mutex
}

type VideoInfo struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Source string `json:"source"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ScriptInfo struct {
	Platform string `json:"platform"`
	Label    string `json:"label"`
	Script   string `json:"script"`
	Desc     string `json:"desc"`
}

func NewApp() *App {
	return &App{
		queue:    engine.NewQueue(),
		settings: LoadSettings(),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Init yt-dlp
	yt, err := engine.NewYtdlp()
	if err != nil {
		wailsRuntime.LogInfo(a.ctx, "yt-dlp not found, will download on first use")
		a.ytdlp = nil
	} else {
		a.ytdlp = yt
		ver, _ := yt.Version()
		wailsRuntime.LogInfo(a.ctx, "yt-dlp: "+ver)
	}

	// Setup download callbacks
	a.queue.OnProgress = func(p engine.QueueProgress) {
		wailsRuntime.EventsEmit(a.ctx, "download-progress", p)
	}
	a.queue.OnJobDone = func(j engine.Job) {
		wailsRuntime.EventsEmit(a.ctx, "download-job-done", j)
	}
	a.queue.OnComplete = func(p engine.QueueProgress) {
		wailsRuntime.EventsEmit(a.ctx, "download-complete", p)
	}
}

// ============================================================
//  Wails Bindings
// ============================================================

func (a *App) DetectPlatform(rawurl string) string {
	p := platform.Detect(rawurl)
	if p == nil {
		return "unknown"
	}
	return p.Name()
}

func (a *App) ExtractURLs(rawurl string) Response {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if input contains multiple URLs (multiline or space-separated)
	if strings.ContainsAny(rawurl, "\n\r ") {
		return a.ParseURLs(rawurl)
	}

	p := platform.Detect(rawurl)
	if p == nil {
		return Response{Success: false, Message: "URL tidak dikenali. Support: Facebook, Instagram, TikTok"}
	}

	var entries []platform.VideoInfo
	var err error

	entries, err = p.ExtractURLs(rawurl, a.settings.CookiesFile)
	if err != nil {
		return Response{Success: false, Message: err.Error()}
	}

	if len(entries) == 0 {
		return Response{Success: false, Message: "Tidak ada video ditemukan. Coba paste manual via 'Paste URLs'"}
	}

	info := make([]VideoInfo, len(entries))
	for i, e := range entries {
		info[i] = VideoInfo{URL: e.URL, Title: e.Title, Source: e.Source}
	}

	return Response{Success: true, Message: "OK", Data: info}
}

// ParseURLs parses raw text (multiline/space-separated) into video entries
func (a *App) ParseURLs(rawtext string) Response {
	lines := strings.Fields(rawtext)
	var entries []VideoInfo
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Normalize URL
		if !strings.HasPrefix(line, "http") {
			line = "https://" + line
		}

		// Skip if already seen
		if seen[line] {
			continue
		}
		seen[line] = true

		// Detect platform
		p := platform.Detect(line)
		source := "unknown"
		if p != nil {
			source = p.Name()
		} else if strings.Contains(line, "facebook.com") || strings.Contains(line, "fb.com") || strings.Contains(line, "fb.watch") {
			source = "facebook"
		} else if strings.Contains(line, "instagram.com") || strings.Contains(line, "instagr.am") {
			source = "instagram"
		} else if strings.Contains(line, "tiktok.com") || strings.Contains(line, "vm.tiktok") {
			source = "tiktok"
		} else {
			continue // skip non-video URLs
		}

		// Extract title from URL
		title := extractTitleFromURL(line)
		entries = append(entries, VideoInfo{URL: line, Title: title, Source: source})
	}

	if len(entries) == 0 {
		return Response{Success: false, Message: "Tidak ada URL video ditemukan"}
	}

	return Response{Success: true, Message: fmt.Sprintf("OK %d video", len(entries)), Data: entries}
}

func (a *App) QueueDownload(videos []VideoInfo) Response {
	jobs := make([]engine.Job, len(videos))
	for i, v := range videos {
		jobs[i] = engine.Job{
			Index:  i + 1,
			URL:    v.URL,
			Title:  v.Title,
			Source: v.Source,
		}
	}
	a.queue.Add(jobs)
	return Response{Success: true, Message: "OK"}
}

func (a *App) StartDownload(concurrent int) Response {
	if a.ytdlp == nil {
		yt, err := engine.NewYtdlp()
		if err != nil {
			// Auto-download yt-dlp
			yt = &engine.Ytdlp{}
			if dlErr := yt.EnsureDownloaded(); dlErr != nil {
				return Response{Success: false, Message: "Gagal download yt-dlp: " + dlErr.Error()}
			}
			a.ytdlp = yt
		} else {
			a.ytdlp = yt
		}
	}

	outDir := a.settings.OutputDir
	if outDir == "" {
		outDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads", "FetchVid")
	}
	os.MkdirAll(outDir, 0755)

	go a.queue.Start(concurrent, outDir, a.settings.CookiesFile, a.ytdlp.Path)
	return Response{Success: true, Message: "Download dimulai"}
}

func (a *App) PauseDownload() Response {
	a.queue.Pause()
	return Response{Success: true, Message: "Di-pause"}
}

func (a *App) ResumeDownload() Response {
	a.queue.Resume()
	return Response{Success: true, Message: "Di-resume"}
}

func (a *App) StopDownload() Response {
	a.queue.Stop()
	return Response{Success: true, Message: "Di-stop"}
}

func (a *App) SelectFolder() string {
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Pilih folder penyimpanan",
	})
	if err != nil || dir == "" {
		return ""
	}
	a.settings.OutputDir = dir
	SaveSettings(a.settings)
	return dir
}

func (a *App) SelectCookiesFile() string {
	file, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Pilih file cookies (.txt format Netscape)",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Cookies", Pattern: "*.txt"},
			{DisplayName: "Semua", Pattern: "*.*"},
		},
	})
	if err != nil || file == "" {
		return ""
	}
	a.settings.CookiesFile = file
	SaveSettings(a.settings)
	return file
}

func (a *App) GetScripts() []ScriptInfo {
	var all []ScriptInfo
	for _, p := range platform.All() {
		for _, s := range p.ConsoleScripts() {
			all = append(all, ScriptInfo{
				Platform: s.Platform,
				Label:    s.Label,
				Script:   s.Script,
				Desc:     s.Desc,
			})
		}
	}
	return all
}

func (a *App) DownloadYtdlp() Response {
	yt := &engine.Ytdlp{}
	if err := yt.EnsureDownloaded(); err != nil {
		return Response{Success: false, Message: "Gagal download yt-dlp: " + err.Error()}
	}
	a.ytdlp = yt
	ver, _ := yt.Version()
	return Response{Success: true, Message: "yt-dlp " + ver + " siap!"}
}

func (a *App) GetSettings() *Settings {
	return a.settings
}

func (a *App) SaveSettingsData(data string) Response {
	var s Settings
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return Response{Success: false, Message: err.Error()}
	}
	a.settings = &s
	SaveSettings(a.settings)
	return Response{Success: true, Message: "Disimpan"}
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
			if strings.Contains(rawurl, "/video/") {
				return "Video " + id
			}
			return id
		}
	}
	return "Video"
}

// resolveViaYtdlp uses yt-dlp to resolve Facebook share URLs
func (a *App) resolveViaYtdlp(rawurl string) string {
	if a.ytdlp == nil {
		return ""
	}
	entries, err := a.ytdlp.ExtractPlaylist(rawurl)
	if err != nil {
		if resolvedErr, ok := err.(*engine.ResolvedURLError); ok {
			return resolvedErr.ResolvedURL
		}
		return ""
	}
	if len(entries) > 0 {
		return rawurl
	}
	return ""
}
