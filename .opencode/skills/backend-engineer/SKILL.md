# SKILL.md — Backend Engineer

## Role
Implementasi backend Go + Wails. Semua logika backend ada di sini.

## Key Files

| File | Responsibility |
|------|---------------|
| `main.go` | Wails app init, window config |
| `app/app.go` | Wails struct binding: semua fungsi yang dipanggil JS |
| `app/download.go` | Download queue, concurrent execution, progress callback |
| `app/settings.go` | Settings manager (save/load config file) |
| `engine/ytdlp.go` | yt-dlp wrapper: check version, download, progress parsing |
| `engine/queue.go` | Job queue dengan progress tracking |
| `engine/concurrent.go` | Goroutine pool manager untuk concurrent download |

## Wails App Struct

```go
type App struct {
    ctx         context.Context
    downloader  *engine.Downloader
    settings    *engine.Settings
    queue       *engine.Queue
    mu          sync.Mutex
}

// — BINDINGS (dipanggil dari JS) —

func (a *App) Startup(ctx context.Context)           // Init app, check yt-dlp
func (a *App) DetectPlatform(url string) string       // Return "facebook" / "instagram" / "tiktok" / "unknown"
func (a *App) ExtractURLs(url string) []VideoInfo     // Scrape URLs dari profile
func (a *App) QueueDownload(urls []VideoInfo) Response    // Add to queue
func (a *App) StartDownload(concurrent int) Response      // Start with N concurrent
func (a *App) PauseDownload() Response
func (a *App) ResumeDownload() Response
func (a *App) GetProgress() ProgressInfo              // Poll from JS
func (a *App) SelectFolder() string                   // Native folder picker
func (a *App) SaveCookies(path string) Response       // Save cookies file
func (a *App) LoadScripts() []ScriptInfo              // Return console scripts per platform
```

## Download Engine

```go
type Downloader struct {
    ytdlpPath   string
    CookiesFile string
    OutputDir   string
}

func (d *Downloader) IsAvailable() bool
func (d *Downloader) Download(url string, jobID int, total int, progress chan Progress) error
func FindOrDownloadYtdlp() (string, error)  // Cari di PATH atau download
```

## Progress Tracking

```go
type ProgressInfo struct {
    Total       int     `json:"total"`
    Completed   int     `json:"completed"`
    Success     int     `json:"success"`
    Failed      int     `json:"failed"`
    Percent     float64 `json:"percent"`
    CurrentURL  string  `json:"currentUrl"`
}
```

Progress dikirim via Wails Events:
```go
runtime.EventsEmit(ctx, "download-progress", progressInfo)
```

Frontend listen:
```javascript
wails.EventsOn("download-progress", (data) => { updateUI(data) });
```

## Concurrency

- Default: 3 concurrent downloads
- Range: 1-10
- Package: `engine/concurrent.go` pake pattern worker pool
- Goroutine + channel untuk progress reporting

## Error Handling

- Semua binding return `Response{Success, Message}`
- yt-dlp error parsed dari stderr
- Timeout: 300s per file
- Network error: retry 2x
