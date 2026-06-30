package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

	// Ensure yt-dlp is available
	if a.ytdlp == nil {
		yt, err := engine.NewYtdlp()
		if err != nil {
			return Response{Success: false, Message: "yt-dlp belum tersedia, coba download dulu"}
		}
		a.ytdlp = yt
	}

	p := platform.Detect(rawurl)
	if p == nil {
		return Response{Success: false, Message: "URL tidak dikenali. Support: Facebook, Instagram, TikTok"}
	}

	var entries []platform.VideoInfo
	var err error

	// Try to resolve share URL via yt-dlp first
	if p.Name() == "facebook" {
		resolved := a.resolveViaYtdlp(rawurl)
		if resolved != "" {
			rawurl = resolved
		}
	}

	entries, err = p.ExtractURLs(rawurl, a.settings.CookiesFile)
	if err != nil {
		return Response{Success: false, Message: err.Error()}
	}

	if len(entries) == 0 {
		return Response{Success: false, Message: "Tidak ada video ditemukan. Coba paste manual via 'Paste URLs'"}
	}

	// Convert to VideoInfo
	info := make([]VideoInfo, len(entries))
	for i, e := range entries {
		info[i] = VideoInfo{URL: e.URL, Title: e.Title, Source: e.Source}
	}

	return Response{Success: true, Message: "OK", Data: info}
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
			return Response{Success: false, Message: "yt-dlp tidak ditemukan. Klik 'Download yt-dlp' dulu"}
		}
		a.ytdlp = yt
	}

	outDir := a.settings.OutputDir
	if outDir == "" {
		outDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads", "FetchVid")
	}
	os.MkdirAll(outDir, 0755)

	go a.queue.Start(concurrent, outDir, a.settings.CookiesFile)
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
