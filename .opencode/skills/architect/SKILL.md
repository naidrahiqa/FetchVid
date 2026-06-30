# SKILL.md — Project Architect

## Role
Arsitek utama FetchVid. Bertanggung jawab atas struktur project, tech stack, dan design decisions.

## Tech Stack Decision

```
Language:    Go 1.22+
Framework:   Wails v2 (https://wails.io)
Frontend:    HTML + Tailwind CSS + Vanilla JS
Download:    yt-dlp (bundled, auto-update first run)
Packaging:   Wails build (single .exe)
```

## Architecture

```
FetchVid/
├── main.go                  # Entry point, Wails app init
├── app/                     # Wails App struct (bindings)
│   ├── app.go
│   ├── download.go
│   └── settings.go
├── frontend/                # Frontend assets (Wails embed)
│   ├── index.html
│   ├── dist/
│   │   ├── style.css       # Tailwind compiled
│   │   └── app.js
│   └── src/
│       ├── main.js
│       └── components/
│           ├── UrlInput.js
│           ├── PlatformTab.js
│           ├── ReelList.js
│           ├── DownloadQueue.js
│           └── ScriptsDialog.js
├── platform/               # Platform scraper interfaces
│   ├── platform.go
│   ├── facebook.go
│   ├── instagram.go
│   └── tiktok.go
├── engine/                 # Download engine
│   ├── ytdlp.go
│   ├── queue.go
│   └── concurrent.go
├── util/                   # Utilities
│   ├── cookies.go
│   └── path.go
├── wails.json
├── go.mod
└── go.sum
```

## Design Principles

1. **Single binary** — Wails compile frontend + backend jadi satu exe
2. **No runtime dependency** — yt-dlp di-download first run, disimpan di AppData
3. **Platform modular** — Setiap platform implement interface yang sama (`Platform`)
4. **Concurrent by default** — Download pake goroutine pool
5. **Dark theme native** — Wails native window + dark mode

## Platform Interface

```go
type PlatformInfo struct {
    Name        string
    Color       string
    Icon        string
    MatchRegex  string
}

type Platform interface {
    Detect(url string) bool
    ExtractURLs(url string, cookies string) ([]VideoInfo, error)
}

type VideoInfo struct {
    URL     string
    Title   string
    Source  string // "facebook", "instagram", "tiktok"
}
```

## Code Review Checklist

- [ ] Platform interface implemented correctly?
- [ ] No goroutine leak (context cancel)?
- [ ] Wails binding exported correctly?
- [ ] Frontend using correct Wails runtime calls?
- [ ] Error handling: user gets message, not crash?
- [ ] File paths: platform-aware (Windows OK)?
- [ ] yt-dlp args: no dangerous flags?
