# SKILL.md — Platform Scraper

## Role
Implementasi scraping/extraction untuk Facebook, Instagram, dan TikTok. Modul platform di `platform/`.

## Interface

Setiap platform harus implement:

```go
type Platform interface {
    Name() string
    Match(url string) bool                    // Detect if URL belongs to this platform
    ExtractURLs(url string, cookies string) ([]VideoInfo, error)  // Get video URLs
    ConsoleScript() ScriptInfo                // Return F12 console script instructions
}

type VideoInfo struct {
    URL     string
    Title   string
    Source  string
}

type ScriptInfo struct {
    Platform string
    Label    string
    Script   string
    Desc     string
}
```

## Deteksi URL

```
Facebook:  facebook.com, fb.com, web.facebook.com
Instagram: instagram.com, instagr.am
TikTok:    tiktok.com, vm.tiktok.com
```

## Platform Details

### Facebook (`platform/facebook.go`)

```
Resolve Share URL → profile.php?id=XXX
  ↓
Extract via HTML scraping (with cookies):
  - /reel/ID → https://www.facebook.com/reel/ID
  - /watch/?v=ID → https://www.facebook.com/watch/?v=ID
  - /video/ID → https://www.facebook.com/video/ID
  ↓
Pass to yt-dlp as individual URLs
```

Console script for FB:
```javascript
// Profile - Reels
copy([...document.querySelectorAll('a[href*="/reel/"]')].map(a=>a.href.split("?")[0]).filter((v,i,a)=>a.indexOf(v)===i).join('\n'))

// Fanpage - Reels + Video
copy([...document.querySelectorAll('a[href*="/reel/"], a[href*="/video/"]')].map(a=>a.href.split("?")[0]).filter((v,i,a)=>a.indexOf(v)===i).join('\n'))
```

### Instagram (`platform/instagram.go`)

```
Profile URL → yt-dlp --flat-playlist --dump-json
  ↓
Parse JSON output → extract each video URL
  ↓
Format: https://www.instagram.com/reel/XXXXX/
```

y-dlp extractor `instagram` supports:
- `instagram.com/reel/XXXXX`
- `instagram.com/p/XXXXX` 
- `instagram.com/username/reels/`

Console script for IG:
```javascript
copy([...document.querySelectorAll('a[href*="/reel/"], a[href*="/p/"]')].map(a=>a.href.split("?")[0].split("/").slice(0,7).join("/")+"/").filter((v,i,a)=>a.indexOf(v)===i).join('\n'))
```

### TikTok (`platform/tiktok.go`)

```
Profile URL → yt-dlp --flat-playlist --dump-json
  ↓
Parse JSON output → extract each video URL
  ↓
Format: https://www.tiktok.com/@username/video/XXXXX
```

yt-dlp extractor `tiktok` supports:
- `tiktok.com/@username`
- `tiktok.com/@username/video/XXXXX`
- `vm.tiktok.com/XXXXX`

Console script for TT:
```javascript
copy([...document.querySelectorAll('a[href*="/video/"]')].map(a=>a.href.split("?")[0]).filter((v,i,a)=>a.indexOf(v)===i).join('\n'))
```

## Implementation Notes

1. **Instagram & TikTok** → yt-dlp `--flat-playlist` langsung extract dari profile URL
2. **Facebook** → lebih kompleks, perlu cookies + HTML scraping karena yt-dlp gak support playlist extraction FB
3. **Cookies** → Semua platform bisa pake cookies untuk profile private
4. **Timeout** → 30s per request scraping
5. **Rate limit** → Jeda 1s antara request scraping biar gak kena blok
