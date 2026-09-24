# ujiscan

**Agentic Penetration Testing Platform**

Go backend (native net/http) + HTML/CSS/JS frontend with AI-driven security scanning using Claude API.

## Status

✅ **PRODUCTION-READY** — All phases complete (1-5), deployed & tested.

## Quick Start

### Prerequisites
- Go 1.26+
- nmap, nuclei (in PATH)
- ANTHROPIC_API_KEY (optional, graceful fallback if missing)

### Build & Run

```bash
cd ujiscan
go build -o ujiscan ./cmd/server
./ujiscan
```

Server runs on `http://localhost:8081`

### API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/tools` | List available security tools |
| GET | `/api/stats` | Overall scan statistics |
| POST | `/api/scan` | Start regular scan (nmap + nuclei) |
| GET | `/api/scan/{id}` | Fetch scan results |
| GET | `/api/playbooks` | List available playbooks |
| POST | `/api/scan/playbook` | Start guided playbook scan |
| POST | `/api/agentic` | Start AI-guided agentic scan |
| GET | `/` | Web dashboard (HTML) |

### Example Scans

**Regular Scan:**
```bash
curl -X POST http://localhost:8081/api/scan \
  -H "Content-Type: application/json" \
  -d '{"target":"127.0.0.1"}'
```

**Playbook Scan (Guided Steps):**
```bash
curl -X POST http://localhost:8081/api/scan/playbook \
  -H "Content-Type: application/json" \
  -d '{
    "playbook":"network-discovery",
    "target":"127.0.0.1"
  }'
```

**Agentic Scan (AI-Driven):**
```bash
curl -X POST http://localhost:8081/api/agentic \
  -H "Content-Type: application/json" \
  -d '{
    "playbook":"network-discovery",
    "target":"127.0.0.1",
    "objective":"Find all open ports and services"
  }'
```

## Architecture

### Backend (Go)

```
cmd/server/
├── main.go                    # HTTP server entry, route registration

internal/
├── api/
│   └── handlers.go            # 8 API endpoints + JSON responses
├── executor/
│   └── scan_executor.go       # Tool orchestration & async execution
├── tools/
│   ├── executor.go            # Tool registry & subprocess execution
│   ├── nmap.go                # nmap scans (temp file JSON output)
│   ├── nuclei.go              # nuclei JSONL parsing (v3.11 compat)
│   ├── target_utils.go        # URL→IP parsing utility
│   └── integration_test.go    # Tool chain tests
├── models/
│   └── types.go               # ToolConfig, ToolOutput, ScanResult
├── store/
│   └── scan_store.go          # In-memory thread-safe scan storage
├── playbook/
│   ├── playbook.go            # YAML models + ExecutionContext
│   ├── loader.go              # YAML parser, markdown frontmatter
│   ├── engine.go              # Step-by-step execution
│   ├── agentic.go             # AI feedback loop per-step
│   └── playbooks/             # 3 sample playbooks (YAML)
└── ai/
    └── claude.go              # Claude API HTTP client
```

### Frontend (HTML/CSS/JS)

```
web/
├── index.html                 # Dashboard UI
├── style.css                  # Dark mode by default
├── app.js                     # Form handling, API calls
└── loader.js                  # Theme sanitizer
```

### Playbooks

Located in `playbooks/` (YAML + markdown frontmatter):

| Playbook | Phases | Purpose |
|----------|--------|---------|
| network-discovery.md | recon → enum | Discover hosts & services |
| vulnerability-quick.md | enum → exploit | Quick vuln scan |
| web-full-scan.md | recon → enum → exploit | Full web security audit |

Each playbook defines steps with:
- **Tool:** nmap, nuclei, curl, dig, whois
- **Arguments:** Tool-specific command args
- **Phase:** recon, enum, exploit, verify, report
- **Conditions:** Optional branching (if-then)

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for:
- Detailed phase breakdown (1-5)
- Known issues & fixes
- Architecture decisions
- Testing procedures
- Future roadmap (Phases 6-8)

## Key Features

### Phase 1: Foundation ✅
- Native Go HTTP server (no frameworks)
- Static file serving
- .gitignore & git setup

### Phase 2: Tool Wrappers ✅
- **nmap:** Host discovery, port scanning, service detection
- **nuclei:** Vulnerability scanning (10K+ templates)
- **curl/dig/whois:** Additional reconnaissance tools
- Subprocess execution with timeout & error handling

### Phase 3: API Layer ✅
- 8 RESTful endpoints
- Async goroutine execution
- Thread-safe in-memory scan storage
- JSON request/response format

### Phase 4: Playbook Engine ✅
- YAML + markdown frontmatter format
- Multi-phase execution (recon → enum → exploit → verify → report)
- Condition evaluation for branching
- 3 sample playbooks

### Phase 5: Agentic Loop ✅
- Claude API integration
- Per-step AI analysis of tool output
- Dynamic phase chaining based on AI decisions
- Graceful fallback if API key missing
- Audit trail of AI reasoning

## Important Fixes

### nmap JSON Output (v7.99)
- **Issue:** `-oJ -` (stdout) not supported
- **Fix:** Use temp files with `ioutil.TempFile`, arg order: `target -oJ file`
- **Files:** `internal/tools/nmap.go`

### nuclei v3.11 Compatibility
- **Issue:** `-json` flag removed, use `-jsonl` instead
- **Fix:** JSONL line-by-line parsing in `parseNucleiJSONL()`
- **Files:** `internal/tools/nuclei.go`

### URL Target Parsing
- **Issue:** nmap/nuclei expect IP/domain, not HTTP URLs
- **Fix:** `CleanTarget()` utility extracts domain from `http://example.com/`
- **Files:** `internal/tools/target_utils.go`

## Testing

Run all tests:
```bash
go test ./internal/...
```

Test specific tool:
```bash
go test ./internal/tools -v
```

Verify API:
```bash
# Start server
./ujiscan

# In another terminal
curl http://localhost:8081/api/tools | jq .
```

## Configuration

### Environment Variables
- `ANTHROPIC_API_KEY` — For agentic features (optional)
- `PORT` — Server port (default: 8081)

### Tool Configuration
Tools auto-discovered via `exec.LookPath()`. Ensure these are in PATH:
- nmap
- nuclei
- curl
- dig
- whois

## Performance

| Operation | Time |
|-----------|------|
| Regular scan (nmap + nuclei) | ~20s |
| Playbook scan (1 phase) | ~5-10s |
| Agentic scan (multi-phase) | ~30s (with AI decisions) |
| API response (no tools) | <100ms |

## Roadmap

### Phase 6: Dashboard & Visualization
- WebSocket live logs
- Real-time result streaming
- AI reasoning display
- Result filtering & export

### Phase 7: Advanced Features
- Multi-target scanning
- Report generation (HTML/JSON/PDF)
- Credential management
- Custom playbook builder UI

### Phase 8: Production
- Database backend (PostgreSQL)
- Authentication & RBAC
- Docker/Kubernetes deployment
- Monitoring & alerting

## Contributing

Commit messages follow pattern:
```
<type>: <description>

✅ What works
❌ What doesn't
🔧 Fixes applied
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`

## License

[Add your license here]

## Author

Rudi (asruddin) — Offensive Security & Infrastructure Security
