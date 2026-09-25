# HANDOFF_LLM.md — ujiscan Context for Next LLM

**For:** Next LLM session / model switch / device switch  
**Status:** v1.1.0 — Phases 1–11 ✅ + Phase B (agent framework) ✅ + Phase C.1–C.5 (AI orchestration) ✅  
**Date:** 2026-09-25  
**User:** Rudi (asruddin) — Offensive Security / Penetration Tester  

---

## ⚠️ READ FIRST: Parallel Agent + Drive Worktree (2026-09-25)

1. **Canonical source of truth = GitHub origin.** On a fresh device: `git pull origin main` first — the local worktree lives on Google Drive and can be out of sync.
2. **A parallel agent (Claude Haiku, CLI session `20260924_231033_f04bfa`) actively works on this repo.** Before ANY edit/commit: `git log --oneline -3` + `git status`. New commits → pull + adapt; never re-implement what's already in HEAD.
3. **Commit small & fast.** Uncommitted work in this Drive worktree can be silently reverted (repeatedly happened). Commit specific paths — NEVER `git add -A` (may sweep the other agent's in-flight edits).
4. **Google Drive sync quirk:** files may briefly wedge with `resource deadlock avoided` (EDEADLK) while syncing — wait/retry; keep a backup of anything precious.
5. **Go formatting:** `gofmt -w` on shared packages (e.g. `internal/playbook/`) normalizes whitespace in files other agents wrote — harmless but mention it in the commit message.

---

## TL;DR: What Is ujiscan?

**Agentic Penetration Testing Platform**
- Go backend (pure net/http) + HTML/CSS/JS frontend
- Security tools (config/tools.yaml): nmap, nuclei, dig, subfinder, httpx, whatweb, sslscan, gobuster, nikto
- Playbook engine: YAML-driven multi-phase scanning
- **Phase C AI loop**: adaptive execution + service detection + AI reports
- Status: ✅ v1.1.0, all endpoints working

**Server:** localhost:8081 (login `admin@ujiscan.local` / `admin123`)  
**Dashboard:** local `/html/login.html` · public https://rudi-asr.github.io/ujiscan/ (backend must run locally)  
**GitHub:** https://github.com/rudi-asr/ujiscan (main + gh-pages)

---

## Architecture — Phase B & C (as of C.5, 2026-09-25)

| Concern | Files | Notes |
|---|---|---|
| Agent framework (B) | `internal/agent/{queue,manager,types,base}.go` | TaskQueue (priority/retry/cancel-purge), worker Manager |
| Recon agent (B) | `internal/agent/recon_agent.go` | nmap via generic `tools.Executor.Execute`; parses TEXT output |
| Scanner agent (B) | `internal/agent/scanner_agent.go` | nuclei via generic executor; parses TEXT output |
| Analyzer agent (B) | `internal/agent/analyzer_agent.go` | local classification + optional `Chat(ctx, msgs)` enrichment |
| Agent API (B) | `internal/agent/handlers.go` | `/api/agents/*` handlers — **not yet wired into main.go** |
| AI client (C.1/C.2/C.4.4) | `internal/ai/client.go` | `NewClient()` reads `OPENAI_API_KEY`; no key → deterministic mock mode (works offline). `SelectToolsForTarget`, `AnalyzeToolOutput`, `UpdateToolFeedback`, `GetPreferredTools` |
| AI summary (C.5) | `internal/ai/summary.go` | `GenerateExecutiveSummary` (+ `BuildFallbackSummary` fallback) |
| Service detection (C.3) | `internal/ai/service_detection.go` | `DetectServicesFromNmap`, `GetToolsForServices`, `SummarizeServices` |
| Adaptive loop (C.4) | `internal/playbook/adaptive.go` | `ExecuteToolsPhaseParallel` (max 4, 120s/tool), `AggregatePhaseResults`, `ShouldContinueToPhase` |
| Agentic scan (C.4) | `internal/playbook/agentic.go` | `ExecuteAgenticScan`: recon→enum→exploit adaptive loop, real target (NOT hardcoded), final report |
| Report (C.5) | `internal/playbook/report.go` | `PrioritizeFindings`, `SeverityScore`, `BuildReportMetrics`, `ReportData.ToJSON/ToMarkdown/ToHTML` |

**API auth model:** ALL `/api/*` require `Authorization: Bearer <token>` (login via `POST /auth/login`). Public: `/api/status`, `/auth/login`, static assets (GET). CORS outermost middleware → headers on 401 too.

**Phase C progress:** C.1–C.5 = 100% (docs/PHASE-C-ROADMAP.md). Next candidates: real OpenAI API key wiring (see "Integration with Real OpenAI API" in roadmap), wire `/api/agents/*` into main.go, roadmap C.6+ items.

---

## Critical Context for Next LLM

### 1. TOOL COMPATIBILITY ISSUES (MUST KNOW)

**nmap 7.99 Problem:**
- ❌ `-oJ -` (JSON to stdout) NOT supported
- ✅ Fix: Use temp files `ioutil.TempFile + defer os.Remove`
- ✅ Arg order matters: `nmap -F target -oJ /tmp/file.json`
- Location: `internal/tools/nmap.go` (lines 10-40)

**nuclei v3.11 Problem:**
- ❌ `-json` flag removed, use `-jsonl` instead
- ❌ `-stats false` flag not available
- ✅ Fix: Use `-jsonl` for JSONL output, parse line-by-line
- Location: `internal/tools/nuclei.go` (lines 10-31)

**URL Target Parsing:**
- Input: `"http://mitracakra.digital/"` → Tools expect IP/domain
- ✅ Fix: `CleanTarget()` util extracts domain only
- Location: `internal/tools/target_utils.go`

**If you change tools or versions:**
1. Test locally: `nmap -F 127.0.0.1`, `nuclei -u 127.0.0.1 -jsonl`
2. Verify flags work before committing
3. Update DEVELOPMENT.md with version compat info

---

### 2. CURRENT ARCHITECTURE (READ FIRST)

**Directory Structure:**
```
ujiscan/
├── cmd/server/main.go               # Server entry
├── internal/
│   ├── api/handlers.go              # 8 endpoints
│   ├── executor/scan_executor.go    # Async execution
│   ├── tools/                       # Tool wrappers
│   ├── playbook/                    # YAML engine + agentic loop
│   ├── ai/claude.go                 # Claude API client
│   ├── store/scan_store.go          # In-memory storage
│   └── models/types.go              # Structs
├── web/                             # Frontend (HTML/CSS/JS)
├── playbooks/                       # 3 YAML files
├── README.md                        # Quick start
├── DEVELOPMENT.md                   # Detailed guide
├── HANDOFF_LLM.md                   # This file
└── go.mod                           # Go modules
```

**API Endpoints (All Working):**
```
GET  /api/tools                      # List tools
GET  /api/stats                      # Stats
POST /api/scan                       # Regular scan
GET  /api/scan/{id}                  # Get results
GET  /api/playbooks                  # List playbooks
POST /api/scan/playbook              # Guided scan
POST /api/agentic                    # AI-driven scan
GET  /                               # Dashboard
```

---

### 3. HOW SCANNING WORKS

**Regular Scan (POST /api/scan):**
```
1. Handler receives target (IP/domain/URL)
2. Spawns goroutine → scanExecutor.Execute()
3. Executor runs each tool: nmap, nuclei, curl, dig, whois
4. Capture output → parse JSON/text
5. Store in ScanStore (sync.Map + mutex)
6. Return scan ID immediately (status: pending/running/completed)
7. Client polls GET /api/scan/{id} for results
```

**Playbook Scan (POST /api/scan/playbook):**
```
1. Load YAML from playbooks/ directory
2. Parse frontmatter (name, description, entry_phase)
3. Execute phases sequentially: recon → enum → exploit → verify → report
4. Each phase runs only specified tools
5. Accumulate results across phases
6. Return final multi-phase results
```

**Agentic Scan (POST /api/agentic):**
```
1. Same as playbook scan, but after EACH step:
2. Send tool output to Claude API
3. Claude analyzes: "Should we continue enum, switch to exploit, or stop?"
4. AI decision determines next phase
5. Repeat until Claude says stop
6. Audit trail: store AI decisions + reasoning
```

---

### 4. NEXT DEVELOPER CHECKLIST

If you're picking up ujiscan:

**Setup:**
- [ ] `go build -o ujiscan ./cmd/server`
- [ ] `./ujiscan` (server on :8081)
- [ ] Open http://localhost:8081 in browser
- [ ] Read README.md (quick overview)
- [ ] Read DEVELOPMENT.md (detailed guide)

**Verify Working:**
```bash
# All should return 200 + valid JSON
curl http://localhost:8081/api/tools | jq .
curl -X POST http://localhost:8081/api/scan \
  -H "Content-Type: application/json" \
  -d '{"target":"127.0.0.1"}'

# Wait 20s, then check results
curl http://localhost:8081/api/scan/{scan-id} | jq '.status'
```

**If Something Breaks:**
1. Check if tools installed: `which nmap`, `which nuclei`
2. Verify Go build: `go build ./...` (no errors?)
3. Read DEVELOPMENT.md "Troubleshooting" section
4. Check commit history for recent changes
5. Refer to tool compatibility issues (section 1)

---

### 5. RECENT FIXES (KNOW THESE)

**Commit b5c7e5c:** nmap temp file output
- Changed: `-oJ -` → use temp file + read back
- Reason: nmap 7.99 doesn't support stdout JSON
- Affected: All nmap functions (Scan, Quick, PingDiscovery)

**Commit 07d4677:** nuclei v3 + URL parsing
- Changed: `-json` → `-jsonl`, added parseNucleiJSONL()
- Added: CleanTarget() to extract domain from URLs
- Reason: nuclei v3.11 API changes

**Commit 6af16ca:** Agentic loop complete
- Added: internal/ai/claude.go (Claude API client)
- Added: internal/playbook/agentic.go (AI feedback loop)
- Extended: ExecutionContext with AI fields

---

### 6. ENVIRONMENT SETUP

**Required:**
- Go 1.26+
- nmap (system PATH)
- nuclei (system PATH)

**Optional:**
- `ANTHROPIC_API_KEY=sk-ant-xxx...` (for agentic features)
  - Without it: graceful fallback (default to "continue" decision)
  - With it: real Claude API calls

**Verify:**
```bash
go version          # 1.26+
nmap -version       # Should print version
nuclei -version     # v3.11.1 (tested)
echo $PATH          # Includes /usr/local/bin?
```

---

### 7. KEY FILES TO UNDERSTAND

**Start Here:**
1. `README.md` — What it does (2 min read)
2. `cmd/server/main.go` — Server setup (5 min)
3. `internal/api/handlers.go` — All 8 endpoints (10 min)

**Then Deep Dive:**
4. `internal/tools/executor.go` — How tools execute
5. `internal/playbook/engine.go` — Playbook execution
6. `internal/playbook/agentic.go` — AI loop (commented)
7. `internal/ai/claude.go` — Claude API calls

**Frontend:**
8. `web/index.html` — Dashboard UI
9. `web/app.js` — Form handling + API calls
10. `web/style.css` — Dark mode styling

---

### 8. COMMON TASKS

**Add a new API endpoint:**
1. Define handler in `internal/api/handlers.go`
2. Register route in `cmd/server/main.go` (add to mux)
3. Add tests if applicable
4. Commit with message: `feat: add new endpoint`

**Add a new tool (e.g., masscan):**
1. Create `internal/tools/masscan.go` with ScanWithMasscan()
2. Register in `internal/tools/executor.go` InitializeDefaultTools()
3. Test locally before commit
4. Update README.md tool list

**Change playbook format:**
1. Update `internal/playbook/loader.go` parser
2. Update existing playbooks in `playbooks/`
3. Add tests
4. Commit: `refactor: update playbook format`

**Improve agentic loop:**
1. Edit `internal/playbook/agentic.go`
2. Modify Claude prompt in `internal/ai/claude.go`
3. Test with `/api/agentic` endpoint
4. Commit: `refactor: improve AI decision logic`

---

### 9. TESTING WORKFLOW

```bash
# Unit tests
go test ./internal/tools -v
go test ./internal/store -v

# Integration test (start server first)
./ujiscan &
sleep 2

# Regular scan
SCAN_ID=$(curl -s -X POST http://localhost:8081/api/scan \
  -H "Content-Type: application/json" \
  -d '{"target":"127.0.0.1"}' | jq -r '.id')

sleep 20

# Check results
curl http://localhost:8081/api/scan/$SCAN_ID | jq '{status, tools: [.results[].tool_name]}'

# Kill server
pkill ujiscan
```

---

### 10. GIT WORKFLOW

**Commit message format:**
```
<type>: <description>

✅ What works
❌ What doesn't (if applicable)
🔧 Fixes applied (if applicable)

Files changed:
- internal/tools/nmap.go
- internal/api/handlers.go
```

**Types:** feat, fix, docs, test, refactor

**Example:**
```
fix: nmap temp file output parsing

✅ nmap 7.99 now uses temp files for JSON output
✅ Argument order corrected (target before -oJ)
✅ Both nmap + nuclei execute without errors

Files:
- internal/tools/nmap.go (all 3 functions updated)
- DEVELOPMENT.md (added troubleshooting note)
```

---

### 11. PERFORMANCE BASELINES

Know these timings (for debugging slow scans):

| Task | Time | Notes |
|------|------|-------|
| Regular scan (nmap+nuclei) | ~20s | Sequential tools |
| nmap -F 127.0.0.1 | ~5s | Quick scan |
| nuclei -u 127.0.0.1 | ~15-20s | 10K templates |
| Playbook (1 phase) | ~5-10s | Depends on phase tools |
| Agentic (3 phases + AI) | ~30s | nmap→AI→nuclei→AI |
| API response (no tools) | <100ms | Should be instant |

If scan takes longer: nuclei is loading templates (normal for first run).

---

### 12. DEPLOYMENT NOTES

**Current:** Standalone Go binary, no Docker
**Future (Phase 8):** Docker + Kubernetes

**To Deploy:**
```bash
go build -o ujiscan ./cmd/server
scp ujiscan user@production:/opt/ujiscan/
ssh user@production /opt/ujiscan/ujiscan
```

**Monitoring:**
- Logs: stdout only (add logging service in Phase 8)
- Port: 8081 (change in code if needed)
- Data: Lost on restart (add database in Phase 8)

---

### 13. WHAT'S NEXT (ROADMAP)

**Phase 6 (Next ~5 days):**
- WebSocket live logs
- Real-time result streaming
- AI reasoning display in UI
- Result filtering & export

**Phase 7 (~10 days):**
- Multi-target scanning
- Report generation (HTML/PDF)
- Credential management
- Custom playbook UI builder

**Phase 8 (~15 days):**
- PostgreSQL backend
- User authentication
- Docker/K8s deployment
- CI/CD pipeline

---

## Quick Links

- **GitHub:** https://github.com/rudi-asr/ujiscan
- **README:** See README.md (quick start + architecture)
- **Dev Guide:** See DEVELOPMENT.md (phases 1-5 breakdown)
- **Server:** localhost:8081 (when running)

---

## Signal to Next LLM

**If you're a future Claude/GPT/other LLM:**

1. ✅ Read this file first (you're reading it!)
2. ✅ Read README.md next (overview)
3. ✅ Read DEVELOPMENT.md (detailed breakdown)
4. ✅ Check git log for recent changes
5. ✅ Review known tool issues (section 1)
6. ✅ Run verification tests (section 8)

Then you're ready to:
- Add features
- Fix bugs
- Deploy
- Improve architecture

**Common mistakes to avoid:**
- ❌ Don't use `-oJ -` with nmap 7.99
- ❌ Don't use `-json` with nuclei 3.11
- ❌ Don't forget CleanTarget() for URL inputs
- ❌ Don't test without building (`go build` first)

---

**Good luck! 🚀**

—Rudi (original dev)  
ujiscan Team
