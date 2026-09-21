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
	ctx      context.Context
	queue    *engine.Queue
	ytdlp    *engine.Ytdlp
	settings *Settings
	mu       sync.Mutex
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

	yt, err := engine.NewYtdlp()
	if err != nil {
		wailsRuntime.LogInfo(a.ctx, "yt-dlp not found, will download on first use")
	} else {
		a.ytdlp = yt
		ver, _ := yt.Version()
		wailsRuntime.LogInfo(a.ctx, "yt-dlp: "+ver)
	}

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

func (a *App) DetectPlatform(rawurl string) string {
	p := platform.Detect(rawurl)
	if p == nil {
		return "unknown"
	}
	return p.Name()
}

func (a *App) ExtractURLs(rawurl string) Response {
	// Parse multiple URLs without holding the lock
	if strings.ContainsAny(rawurl, "\n\r ") {
		return a.parseURLs(rawurl)
	}

	p := platform.Detect(rawurl)
	if p == nil {
		return Response{Success: false, Message: "URL tidak dikenali. Support: Facebook, Instagram, TikTok"}
	}

	// Don't hold lock during network call
	cookiesFile := a.settings.CookiesFile
	entries, err := p.ExtractURLs(rawurl, cookiesFile)
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

func (a *App) parseURLs(rawtext string) Response {
	lines := strings.Fields(rawtext)
	var entries []VideoInfo
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, "http") {
			line = "https://" + line
		}

		if seen[line] {
			continue
		}
		seen[line] = true

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
			continue
		}

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
	if concurrent < 1 {
		concurrent = 1
	}
	if concurrent > 10 {
		concurrent = 10
	}

	a.mu.Lock()
	if a.ytdlp == nil {
		yt, err := engine.NewYtdlp()
		if err != nil {
			yt = &engine.Ytdlp{}
			if dlErr := yt.EnsureDownloaded(); dlErr != nil {
				a.mu.Unlock()
				return Response{Success: false, Message: "Gagal download yt-dlp: " + dlErr.Error()}
			}
			a.ytdlp = yt
		} else {
			a.ytdlp = yt
		}
	}
	ytdlpPath := a.ytdlp.Path
	a.mu.Unlock()

	outDir := a.settings.OutputDir
	if outDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Response{Success: false, Message: "Gagal dapat home dir: " + err.Error()}
		}
		outDir = filepath.Join(home, "Downloads", "FetchVid")
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return Response{Success: false, Message: "Gagal buat folder: " + err.Error()}
	}

	go a.queue.Start(concurrent, outDir, a.settings.CookiesFile, ytdlpPath)
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
	a.mu.Lock()
	a.ytdlp = yt
	a.mu.Unlock()
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

func (a *App) ExtractTikTokFromVideo(videoURL string) Response {
	p := platform.Detect(videoURL)
	if p == nil || p.Name() != "tiktok" {
		return Response{Success: false, Message: "URL bukan TikTok video"}
	}

	tiktok, ok := p.(*platform.TikTok)
	if !ok {
		return Response{Success: false, Message: "Platform error"}
	}

	cookiesFile := a.settings.CookiesFile
	channelID, err := tiktok.ExtractChannelIDFromVideo(videoURL, cookiesFile)
	if err != nil {
		return Response{Success: false, Message: "Gagal extract channel_id: " + err.Error()}
	}
	if channelID == "" {
		return Response{Success: false, Message: "channel_id tidak ditemukan di video ini"}
	}

	profileURL := "tiktokuser:" + channelID
	entries, err := tiktok.ExtractURLs(profileURL, cookiesFile)
	if err != nil {
		return Response{Success: false, Message: "Gagal ambil video dari profile: " + err.Error()}
	}

	if len(entries) == 0 {
		return Response{Success: false, Message: "Tidak ada video ditemukan"}
	}

	info := make([]VideoInfo, len(entries))
	for i, e := range entries {
		info[i] = VideoInfo{URL: e.URL, Title: e.Title, Source: e.Source}
	}

	return Response{Success: true, Message: fmt.Sprintf("OK %d video", len(entries)), Data: info}
}
