# Changelog

## [0.1.0-alpha] - 2026-06-30

### Added
- Multi-platform support: Facebook, Instagram, TikTok
- Share URL auto-resolve (facebook.com/share/xxx)
- HTML scraping with cookies for private profiles
- Concurrent downloads (1-10 simultaneous)
- Dark theme UI with inline CSS (no Tailwind dependency)
- Console scripts for browser extraction (click-to-copy modal)
- yt-dlp auto-download on first run
- Folder picker dialog (settings persisted)
- Progress bars and realtime log
- Direct URL paste support (multiline/space-separated)
- Facebook direct reel/video URL detection
- Window hidden for yt-dlp processes (no CMD popup)

### Fixed
- Progress bar JSON field name mismatch (lowercase vs PascalCase)
- Script button now copies to clipboard instead of uncopyable prompt()
- yt-dlp CMD window spam when downloading
- Queue progress counter not updating (success/failed counts)

## [0.0.1] - 2026-06-29

### Added
- Initial release
- Basic Facebook reel download
- yt-dlp integration
