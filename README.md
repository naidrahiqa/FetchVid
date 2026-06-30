# FetchVid

Batch downloader untuk Facebook Reels, Instagram Reels, dan TikTok Videos. Desktop app built with Go + Wails, single `.exe` tanpa dependency.

![Platform](https://img.shields.io/badge/platform-Windows-blue)
![Go](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)
![Wails](https://img.shields.io/badge/wails-v2.12-purple)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

- Multi-platform: Facebook (Reels & Videos), Instagram (Reels), TikTok (Videos)
- Concurrent downloads (1-10 simultaneous)
- yt-dlp auto-download on first run
- Dark theme UI
- Progress bar + realtime log
- Folder picker untuk penyimpanan
- Console scripts untuk extract URLs dari browser
- Share URL auto-resolve (`facebook.com/share/xxx`)

## Screenshot

<p align="center">
  <img src="https://raw.githubusercontent.com/naidrahiqa/FetchVid/main/screenshot.png" width="700" alt="FetchVid UI">
</p>

## Download

### Stable Release
Download dari [GitHub Releases](https://github.com/naidrahiqa/FetchVid/releases/latest)

### Build from Source

**Prerequisites:**
- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone
git clone https://github.com/naidrahiqa/FetchVid.git
cd FetchVid

# Build
wails build -o FetchVid.exe
```

Output: `build/bin/FetchVid.exe`

## Usage

### Cara Pakai

1. Jalankan `FetchVid.exe`
2. First run akan otomatis download yt-dlp (~10MB)
3. Masukkan URL profile atau paste langsung reel URLs
4. Pilih folder penyimpanan (Settings tersimpan)
5. Klik **DOWNLOAD VIDEO**

### Extract dari Profile

1. Login ke Facebook/Instagram di browser
2. Buka profile yang ingin didownload
3. Klik **Scripts** di FetchVid → pilih script yang sesuai
4. Copy script → paste di browser console (F12 → Console)
5. Script akan copy semua URL reel ke clipboard
6. Klik **Paste URLs** di FetchVid

### Console Scripts Tersedia

| Script | Untuk |
|--------|-------|
| Profile - Reels | Reels dari akun personal Facebook |
| Fanpage - Reels + Video | Reels & Video dari fanpage Facebook |
| Semua Video | Semua link video (reel + video + watch) |
| Instagram - Reels | Reels & posts dari profile Instagram |
| TikTok - Videos | Semua video dari profile TikTok |

### Direct URL

Langsung paste URL reel/video:
```
https://www.facebook.com/reel/123456
https://www.instagram.com/reel/ABC123
https://www.tiktok.com/@user/video/123456
```

## Configuration

Settings tersimpan di:
```
%APPDATA%/FetchVid/config.json
```

| Field | Default | Description |
|-------|---------|-------------|
| `OutputDir` | `~/Downloads/FetchVid` | Folder penyimpanan |
| `CookiesFile` | - | Path ke file cookies (Netscape format) |
| `Concurrent` | 3 | Jumlah download simultan |
| `Theme` | dark | UI theme |

### Cookies

Untuk download dari private profile, buat file cookies:
1. Install extension [Get cookies.txt](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc)
2. Login ke Facebook/Instagram
3. Klik extension → Export cookies
4. Simpan sebagai `cookies.txt`
5. Klik **Cookies** di FetchVid → pilih file

## Tech Stack

- **Backend:** Go + Wails v2
- **Frontend:** Vanilla HTML/CSS/JS (inline, no framework)
- **Download Engine:** yt-dlp
- **CI/CD:** GitHub Actions

## Project Structure

```
FetchVid/
├── main.go                 # Entry point, Wails config
├── app/
│   ├── app.go              # Wails bindings
│   └── settings.go         # Settings load/save
├── engine/
│   ├── ytdlp.go            # yt-dlp wrapper, download logic
│   └── queue.go            # Download queue with worker pool
├── platform/
│   ├── facebook.go         # Facebook scraping + URL extraction
│   ├── instagram.go        # Instagram yt-dlp extraction
│   ├── tiktok.go           # TikTok yt-dlp extraction
│   └── shared.go           # Common helpers
├── frontend/
│   ├── dist/
│   │   └── index.html      # Embedded UI (inline CSS+JS)
│   ├── src/                # Source files
│   └── wailsjs/            # Auto-generated Wails bindings
├── .github/workflows/
│   └── release.yml         # CI: build + release on tag
└── go.mod
```

## Contributing

1. Fork repository
2. Create branch (`git checkout -b feature/xxx`)
3. Commit (`git commit -m 'Add xxx'`)
4. Push (`git push origin feature/xxx`)
5. Open Pull Request

### Development

```bash
# Run in dev mode
wails dev

# Build for production
wails build -trimpath -o FetchVid.exe

# Go vet
go vet ./...
```

## License

[MIT](LICENSE) - Feel free to use, modify, and distribute.

## Acknowledgments

- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - Download engine
- [Wails](https://wails.io/) - Go desktop framework
- [Facebook](https://github.com/nickoala/nickoala.github.io) - Console script reference
