# SKILL.md — UI/UX Designer

## Role
Frontend developer untuk FetchVid. Semua HTML, CSS, JS, dan UX.

## Tech Stack
- HTML5 + Tailwind CSS (CDN atau compiled)
- Vanilla JavaScript (no framework — biar ringan)
- Wails runtime JS API (`window.runtime`)

## Layout

```
+------------------------------------------+
| FetchVid                             _ □ X |  <- Wails native window
+------------------------------------------+
| [🔍 Paste URL] [📂 Folder] [🍪 Cookies] | |  <- Top bar
+------------------------------------------+
|  [ FB ] [ IG ] [ TT ]  [ Console Scripts]|  <- Platform tabs + script button
+------------------------------------------+
|                                          |
| Daftar Video (25 ditemukan)              |
| Select All | Deselect | Clear            |
| +--------------------------------------+ |
| | reel_001: Jihan on Reels             | |
| | reel_002: Enyak enyak                | |
| | reel_003: ...                        | |
| | ...                                  | |
| +--------------------------------------+ |
|                                          |
| ⚡ Download bareng: [3] reel sekaligus   |
| [█████████████░░░░░░░░░░] 57%            |
| [57/1532] Downloading... Jihan on Reels |
|                                          |
| Log:                                     |
| +--------------------------------------+ |
| | [56/1532] SELESAI (4.05MB)           | |
| | [57/1532] Downloading... 780KB/s     | |
| +--------------------------------------+ |
+------------------------------------------+
```

## Color Palette (Dark Theme)

| Token | Hex | Usage |
|-------|-----|-------|
| `--bg-primary` | `#0f0f13` | Main background |
| `--bg-secondary` | `#1a1a23` | Card/section bg |
| `--bg-tertiary` | `#252533` | Input/hover bg |
| `--text-primary` | `#e8e8ed` | Main text |
| `--text-secondary` | `#8b8ba0` | Subtitle/label |
| `--accent` | `#6c5ce7` | Buttons, active |
| `--accent-hover` | `#7f6ff0` | Button hover |
| `--success` | `#00b894` | SELESAI |
| `--error` | `#d63031` | GAGAL |
| `--warning` | `#fdcb6e` | Warning text |

## Key JavaScript Functions

```javascript
// Import Wails runtime
import { EventsOn, EventsEmit } from 'wailsjs/runtime';

// Call Go bindings
import { DetectPlatform, ExtractURLs, QueueDownload, StartDownload, GetProgress } from 'wailsjs/go/app/App';

// Listen for progress from Go
EventsOn("download-progress", (data) => {
    updateProgressBar(data.percent);
    updateStats(data.completed, data.total, data.success, data.failed);
});

// Call Go functions
async function extractURLs(url) {
    const result = await ExtractURLs(url);
    if (result.success) {
        renderVideoList(result.data);
    }
}
```

## Pages / Dialogs

### Main Page
- URL input with paste detection
- Folder picker button
- Platform tabs (FB / IG / TT)
- Video list (selectable)
- Download control (concurrent spinner + start/stop)
- Progress bar
- Realtime log

### Scripts Dialog (modal)
- 3 scripts: Profile, Fanpage, All
- Copy button each
- Instructions

### Paste URLs Dialog
- Textarea for pasting
- Tips section
- Add button

### Settings Page
- Cookies file path
- Default output folder
- Default concurrent count
- Check yt-dlp version
- Reset settings
