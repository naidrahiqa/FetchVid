# AGENTS.md — FetchVid

## Team Registry

| Role | Member | Skills | Responsibility |
|------|--------|--------|---------------|
| **Project Architect** | architect | system-design, tech-stack, module-boundaries | Arsitektur project, tech decisions, code review |
| **Backend Engineer** | backend-engineer | go, wails, yt-dlp, concurrency | Wails app, download engine, API endpoints |
| **UI/UX Designer** | ui-designer | html, css, js, tailwind, ux | Frontend UI, user experience, dark theme |
| **Platform Scraper** | platform-scraper | facebook, instagram, tiktok, scraping | Extract reel/video URLs dari tiap platform |
| **Release Engineer** | release-engineer | build, packaging, ci/cd | Bundle app, distribute, auto-update yt-dlp |

## Available Skills

| Skill ID | File | Description |
|----------|------|-------------|
| `architect` | `.opencode/skills/architect/SKILL.md` | Arsitektur project & code review |
| `backend-engineer` | `.opencode/skills/backend-engineer/SKILL.md` | Go backend, Wails API, download engine |
| `ui-designer` | `.opencode/skills/ui-designer/SKILL.md` | Frontend UI, HTML/CSS/JS/tailwind |
| `platform-scraper` | `.opencode/skills/platform-scraper/SKILL.md` | Scraper FB, IG, TikTok |
| `release-engineer` | `.opencode/skills/release-engineer/SKILL.md` | Build, bundle, release |

## Workflow Definitions

### Workflow 1: New Feature (Feature Request)
```
Feature Request
  ↓
architect: review & design system ← jika butuh arsitektur baru
  ↓
backend-engineer: implement API + download logic
  +  ui-designer: implement UI (parallel)
  +  platform-scraper: implement scraper (parallel)
  ↓
all: review bersama
  ↓
release-engineer: build & test
```

### Workflow 2: Add New Platform (e.g. Twitter)
```
Add Platform Request
  ↓
architect: design platform integration interface
  ↓
platform-scraper: implement platform scraper (interface + extract + test)
  +  backend-engineer: registrasi platform di Go backend (parallel)
  +  ui-designer: add UI tab untuk platform baru (parallel)
  ↓
all: integration test
  ↓
release-engineer: build & release
```

### Workflow 3: Bug Fix / Polish
```
Bug Report
  ↓
architect: triage & assign
  ↓
[owner-skill]: fix bug
  ↓
backend-engineer: test regression
  ↓
release-engineer: patch release
```

### Workflow 4: Release Build
```
Tag Created (v*)
  ↓
release-engineer: run build pipeline
  +  backend-engineer: update version (parallel)
  ↓
backend-engineer: verify downloadable
  ↓
release-engineer: publish release
```

## Skill Routing Logic

### Event-based
- **URL input** → platform-scraper (detect platform dari URL)
- **Download request** → backend-engineer (queue + concurrent download)
- **UI event** → ui-designer (component update)
- **Build request** → release-engineer (compile + bundle)

### Context-based
- New platform → platform-scraper + architect
- Performance issue → backend-engineer
- UI polish → ui-designer

## Team Workflow Orchestration

### Developer Workflow (New Feature)
```
1. architect → buat design doc
2. backend-engineer → implementasi
3. ui-designer → UI (parallel)
4. platform-scraper → scrape logic (parallel)
5. backend-engineer + ui-designer → integration
6. release-engineer → build
```

### Code Review Workflow
```
PR Created
  ↓
architect: architecture review
  +  backend-engineer: code review (parallel)
  ↓
All: approve/request changes
  ↓
Merge
```

## Skill Chaining

```
User Input URL
  ↓
[platform-scraper] detect platform → extract URLs
  ↓  output: {platform: "facebook", urls: [...]}
[backend-engineer] create download jobs
  ↓  output: {jobs: [{id, url, platform, status}]}
[backend-engineer] execute concurrent downloads via yt-dlp
  ↓  output: {job_id: ..., status: "done", file: "path/to/video.mp4"}
[ui-designer] update progress bar + show result
```

## CLI Interface

```bash
# Add platform
opencode-agent workflow run add-platform --platform twitter

# New feature
opencode-agent workflow run new-feature --feature "batch-select-folder"

# Release
opencode-agent workflow run release-build --tag v1.0.0

# Run specific skill
opencode-agent run architect --path ./backend
opencode-agent run platform-scraper --platform facebook

# List skills
opencode-agent skills list
opencode-agent skills info backend-engineer
```

## Configuration

See `.opencode/config/workflow-config.yaml`

## Monitoring & Metrics

| Metric | Source | Alert |
|--------|--------|-------|
| Download success rate | backend-engineer | < 80% |
| Scraper error rate | platform-scraper | > 10% |
| UI response time | ui-designer | > 2s |
| Build duration | release-engineer | > 10min |
| yt-dlp version stale | release-engineer | > 30 days |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| yt-dlp outdated | `release-engineer` update bundled yt-dlp |
| Scraper broken (FB changed HTML) | `platform-scraper` update regex |
| UI not loading | `ui-designer` check console errors |
| Download hangs | `backend-engineer` check goroutine leaks |
| Platform not detected | `platform-scraper` update URL pattern |
