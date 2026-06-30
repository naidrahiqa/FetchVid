# SKILL.md — Release Engineer

## Role
Build, bundle, dan distribusi FetchVid.

## Build Commands

```bash
# Development
wails dev              # Hot reload mode

# Production build
wails build -production -upx -trimpath -s

# Output: FetchVid.exe (~15-20MB)
```

## Bundled Assets

Semua file yang perlu dibundle di dalam binary:

| Asset | Source | Destination |
|-------|--------|-------------|
| Frontend (HTML/JS/CSS) | `frontend/dist/` | Embedded by Wails |
| Platform icons | `frontend/src/assets/` | Embedded |
| yt-dlp | Downloaded first run | `%AppData%/FetchVid/bin/yt-dlp.exe` |

## First Run Setup

```
App Start
  ↓
Check: yt-dlp.exe exists di AppData/bin?
  ├─ No → Download latest yt-dlp from GitHub
  │        URL: https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe
  │        Save to: %AppData%/FetchVid/bin/yt-dlp.exe
  └─ Yes → Check version (`yt-dlp --version`)
          └─ If outdated (>30 days) → Background update

Check: ffmpeg.exe exists?
  ├─ No → Download ffmpeg
  └─ Yes → Done
```

## Directory Structure (User Machine)

```
%AppData%/FetchVid/
├── bin/
│   ├── yt-dlp.exe
│   └── ffmpeg.exe          # Untuk merger video+audio
├── config.json              # User settings
├── cookies/                 # Saved cookies files
└── logs/                    # Download logs
```

## Update Strategy

1. **yt-dlp**: Auto-update di background (check version setiap 7 hari)
2. **FetchVid**: Manual update via GitHub releases (lo bisa kalo mau)
3. **Version file**: `util/version.go` dengan `var Version = "1.0.0"`

## Distribution

```bash
# Build for distribution
wails build -production -upx -trimpath -o FetchVid.exe

# Zip for release
Compress-Archive -Path FetchVid.exe -DestinationPath FetchVid-v1.0.0-windows-amd64.zip
```

## Pre-Release Checklist

- [ ] `wails build` successful
- [ ] yt-dlp bundled/auto-download working
- [ ] Anti-virus scan: no false positive
- [ ] Windows 10/11 test: minimal
- [ ] UI text: no hardcoded debug strings
- [ ] Version number: updated in `util/version.go`
- [ ] CHANGELOG updated
