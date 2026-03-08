# Ass Scraper 4000

No-Code Scrape-to-API platform which lists element using Go/Rod at every step and creates and MD file. Thus makes decision making for an llm both easier and cheaper. Supports **SQLite persistence**, **cron scheduling**, and a retro terminal dashboard. Record browser automations visually, save them as reusable "recipes" (JSON), and serve them as REST API endpoints — all with anti-bot stealth built in.

## Modes

| Mode | How it works | AI needed? |
|------|-------------|------------|
| **AI Mode** (`run`) | Gemini decides which elements to interact with | Yes (first run) |
| **Replay Mode** (`run`) | Replays a previously saved flow | No |
| **Manual Mode** (`manual`) | You type commands in the terminal after seeing the UI map | No |
| **Learning Mode** (`learn`) | You click around in Chrome; the agent silently records everything | No |
| **API Server** (`serve`) | REST API + retro dashboard + scheduler to run recipes headlessly | No |

All modes produce the same JSON recipe format. A recipe recorded in any mode can be replayed with `run` or the API server.

## Installation & Setup

### Prerequisites

- **Go 1.22+** — [Download](https://go.dev/dl/)
- **Google Chrome** — any recent version (the agent controls it via DevTools Protocol)
- **Google Gemini API key** — only needed for AI Mode; all other modes work without it. Get one free at [Google AI Studio](https://aistudio.google.com/apikey)

### 1. Clone the repository

```bash
git clone https://github.com/aliyavuzz/ass_scraper_4000.git
cd ass_scraper_4000
```

### 2. Install Go dependencies

```bash
go mod download
```

### 3. Build the binary

```bash
# Linux / macOS
go build -o agent .

# Windows
go build -o agent.exe .
```

Single standalone binary — no runtime or interpreter needed.

### 4. Configure environment (optional)

```bash
cp .env.example .env
```

Edit `.env`:

```
GEMINI_API_KEY=your_gemini_api_key_here   # Optional — only for AI Mode
API_KEY=your_api_key_here                 # Optional — protects API server endpoints
```

### 5. Verify the installation

```bash
./agent screenshot -url "https://example.com"
```

### Directory structure

```
web-automation-agent/
  main.go              # thin CLI dispatcher
  internal/
    engine/            # core types, browser, actions, AI, recipe, learn, manual, logging
    db/                # SQLite schema, migrations, recipe CRUD, scrape data archive
    scheduler/         # cron scheduler for background recipe execution
    server/            # REST API server, HTTP handlers, SSE sessions
    dashboard/         # embedded retro terminal HTML/CSS/JS
  flows/               # saved recipes (JSON) — auto-created
  screenshots/         # saved screenshots — auto-created
  scrapematic.db       # SQLite database — auto-created by `serve`
  .env                 # your API keys (git-ignored)
  go.mod / go.sum
```

### Troubleshooting

| Problem | Fix |
|---------|-----|
| `failed to launch browser` | Make sure Chrome/Chromium is installed |
| `GEMINI_API_KEY` not found | Set in `.env`, export it, or pass `-key` to `run` |
| Port already in use | `./agent serve -port 3000` | 'or some other random port you are definitely not using'
| Chrome opens but blank | Try without `-headless` first |

## Quick Start

```bash
# 1. Build
go build -o agent .

# 2. Record a recipe
./agent learn -url "https://example.com/login" -objective "login"

# 3. Replay headlessly
./agent run -url "https://example.com/login" -objective "login" -headless

# 4. Start the API server (with DB + scheduler)
export API_KEY=my-secret-key
./agent serve -port 8080

# 5. Run recipe via API
curl -X POST http://localhost:8080/api/v1/recipes/login/run \
  -H "X-API-Key: my-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"variables":{"username":"admin","password":"secret"}}'

# 6. Get cached data instantly (no browser launched)
curl http://localhost:8080/api/v1/recipes/login/data \
  -H "X-API-Key: my-secret-key"

# 7. Schedule a recipe to run every 10 minutes
curl -X PUT http://localhost:8080/api/v1/recipes/price-scraper/schedule \
  -H "X-API-Key: my-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"cron":"0 */10 * * * *","enabled":true}'
```

## Architecture

```
  Record (3 ways)                Execute               Serve + Persist
  ┌──────────┐                ┌──────────┐         ┌──────────────────┐
  │ AI Mode  │                │ Replay   │         │ REST API         │
  │ Manual   │──> recipe.json │ Headless │──> JSON │ Dashboard (3-tab)│
  │ Learning │                │ Stealth  │         │ SSE Live Matrix  │
  └──────────┘                └──────────┘         │ Cron Scheduler   │
                                                   │ SQLite Database  │
                                                   └──────────────────┘
```

## Actions

| Action | Description | Example |
|--------|------------|---------|
| `CLICK` | Click element at coordinates | `click 3` |
| `TYPE` | Type text into input field | `type 1 admin@test.com` |
| `SELECT` | Select dropdown option by text | `select 5 California` |
| `HOVER` | Hover over element | `hover 2` |
| `UPLOAD` | Upload file to input | `upload 4 /path/to/file.pdf` |
| `SCRAPE` | Extract data via CSS selectors | `scrape {"title":".main-title"}` |
| `PRESS_KEY` | Press keyboard key | `press enter` |
| `SCROLL` | Scroll page down | `scroll` |
| `WAIT_URL` | Wait for URL to contain string | `wait_url /dashboard` |
| `WAIT_ELEMENT` | Wait for text to appear | `wait_element Welcome` |
| `WAIT_IDLE` | Wait for network/DOM idle | `wait_idle` |
| `WAIT_2FA` | Pause for 2FA code entry | `wait_2fa 3` |
| `SCREENSHOT` | Take a screenshot | `screenshot result` |
| `DONE` | Mark objective complete | `done` |

## SCRAPE Action

```json
// Single selector — returns first match text
{"action":"SCRAPE","value":".product-title"}

// Object schema — returns structured data
{"action":"SCRAPE","value":"{\"title\":\".main-title\",\"price\":\".price-tag\",\"items\":[\".list-item\"]}"}
```

## Recipe Format

```json
{
  "site": "example.com",
  "objective": "login",
  "recorded_at": "2024-01-15T10:30:00Z",
  "page_signatures": [42, 38],
  "steps": [
    {"action": "TYPE", "x": 640, "y": 300, "value": "{{username}}"},
    {"action": "TYPE", "x": 640, "y": 360, "value": "{{password}}"},
    {"action": "CLICK", "x": 640, "y": 420},
    {"action": "WAIT_URL", "value": "/dashboard"},
    {"action": "DONE"}
  ],
  "variables": ["username", "password"]
}
```

Variables use `{{name}}` placeholders — actual values are never saved to disk.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Retro dashboard (3-tab UI) |
| GET | `/api/v1/recipes` | List all recipes |
| GET | `/api/v1/recipes/{name}` | Get recipe details |
| POST | `/api/v1/recipes/{name}/run` | Run recipe (sync or SSE) |
| POST | `/api/v1/run` | Run recipe by name in body |
| GET | `/api/v1/recipes/{name}/data` | Get latest cached scrape data |
| GET | `/api/v1/recipes/{name}/data?history=N` | Get last N scrape results |
| GET | `/api/v1/recipes/{name}/schedule` | Get recipe schedule |
| PUT | `/api/v1/recipes/{name}/schedule` | Set cron schedule |
| GET | `/api/v1/sessions/{id}/events` | SSE stream for running session |
| GET | `/api/v1/sessions/{id}/status` | Get session status + logs |
| POST | `/api/v1/sessions/{id}/2fa` | Submit 2FA code |

**Authentication:** Set `API_KEY` env var or pass `-key` flag. Send as `X-API-Key` header or `?api_key=` query param. Dashboard is accessible without a key.

**SSE streaming:** Send `Accept: text/event-stream` header to `/run` endpoint to get a session ID, then connect to the events endpoint for real-time logs.

**Cached data:** `GET /api/v1/recipes/{name}/data` returns the latest successful scrape result from SQLite instantly — no browser launched.

**Scheduling:** `PUT /api/v1/recipes/{name}/schedule` with `{"cron":"0 */10 * * * *","enabled":true}` sets a recurring cron job. Uses 6-field cron (seconds included).

## Dashboard

The web dashboard at `http://localhost:8080` has three tabs:

| Tab | Description |
|-----|-------------|
| **Recipe Gallery** | List all recipes with status badges, launch panel, schedule controls, 2FA input |
| **Live Matrix** | Real-time SSE event stream showing logs from running sessions |
| **Data Lab** | Historical scrape data viewer with recipe selector and time range filters |

## Database

The `serve` command auto-creates `scrapematic.db` (SQLite) with:
- **recipes** table — synced from `./flows/*.json` on startup, stores schedule config and version
- **scrape_data** table — every recipe execution (API, cron, or manual) saves results here
- WAL mode enabled for concurrent read performance

## Cron Scheduler

Recipes can be scheduled via API or dashboard. The scheduler:
- Runs headless recipe execution in background goroutines
- Saves results to SQLite after each run
- Supports 6-field cron expressions (with seconds): `"0 */10 * * * *"` = every 10 minutes
- Syncs from DB on startup — schedules persist across restarts
- Standard cron descriptors also work: `"@hourly"`, `"@daily"`, `"@every 5m"`

## Anti-Bot Stealth

All browser instances use [go-rod/stealth](https://github.com/go-rod/stealth) plus:
- **Viewport jitter** — random ±20px on width/height to avoid fingerprinting
- **User-Agent rotation** — randomly selected from a pool of 6 common Chrome UAs
- **Mouse jitter** — ±2px on click coordinates for human-like interaction
- Navigator property overrides, WebGL vendor spoofing, Chrome runtime injection

## CLI Reference

```
USAGE:
  agent list                    List entries from credentials.csv
  agent run [flags]             Run automation (AI record + replay)
  agent manual [flags]          Manual mode — type commands
  agent learn [flags]           Learning mode — click in Chrome, agent records
  agent serve [flags]           Start API server + dashboard + scheduler
  agent screenshot [flags]      Take a screenshot
  agent interactive             Interactive mode (prompts for input)

RUN FLAGS:
  -url        string   URL to automate
  -objective  string   Recipe label (default: "login")
  -user       string   Username variable
  -pass       string   Password variable
  -headless   bool     Headless mode (default: false)
  -screenshot bool     Screenshot after completion (default: true)
  -key        string   Gemini API key (run only)
  -carrier    string   Entry name from CSV
  -tenant     string   Tenant filter
  -csv        string   CSV path (default: "credentials.csv")

SERVE FLAGS:
  -port    string   Port (default: "8080")
  -key     string   API key (or set API_KEY env var)
```

## Tech Stack

- **Go** + [go-rod/rod](https://github.com/go-rod/rod) — browser automation via Chrome DevTools Protocol
- **go-rod/stealth** — anti-bot fingerprint evasion
- **modernc.org/sqlite** — CGO-free SQLite driver (cross-compilable, no C compiler needed)
- **robfig/cron/v3** — cron scheduler with second-level precision
- **Google Gemini 2.5 Flash** — AI decision engine (optional, first-run only)
- **NES.css** — retro 8-bit dashboard UI
