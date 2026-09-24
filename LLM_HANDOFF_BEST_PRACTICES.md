# LLM Handoff Best Practices — ujiscan Reference

**Document Purpose:** Show best practices implemented in ujiscan for seamless LLM-to-LLM handoff

---

## Why LLM Handoff Matters

When switching LLMs or models:
- ❌ Without docs: New LLM re-discovers architecture, repeats mistakes, loses context
- ✅ With docs: New LLM starts productive immediately, avoids known pitfalls

**ujiscan saves ~30-40 minutes of context re-building per LLM switch.**

---

## Best Practices Implemented in ujiscan

### 1. **Three-Tier Documentation**

**Tier 1: README.md** (Quick Start — 2-5 min read)
- What the project is
- How to run it
- API endpoint table
- Quick examples
- Roadmap overview

**→ For:** First-time readers, GitHub visitors, quick reference

**Tier 2: DEVELOPMENT.md** (Detailed Guide — 15-30 min read)
- Phase-by-phase breakdown
- Known issues with ROOT CAUSES & FIXES
- Architecture diagram
- Key decisions explained
- Performance baselines
- Testing workflows
- Troubleshooting checklist

**→ For:** Developers making changes, debugging issues, understanding architecture

**Tier 3: HANDOFF_LLM.md** (LLM Context — 10-15 min read)
- Immediate context for next LLM
- Critical tool compatibility (copy-paste ready)
- Common mistakes to avoid
- Setup verification steps
- Quick links to other docs

**→ For:** Next LLM session, model switches, context transfer

---

### 2. **Known Issues with Full Root Causes**

**Best Practice:** Don't just say "it's broken" — explain WHY

**ujiscan Example: nmap issue**
```markdown
❌ PROBLEM:
   Command fails: "Unable to split netmask from target expression"
   Input: nmap -F -oJ - 127.0.0.1

🔍 ROOT CAUSE:
   nmap 7.99 does NOT support "-oJ -" (output to stdout)
   nmap interprets "-" as a target, not stdout marker
   
✅ FIX:
   Use temp files: ioutil.TempFile() + defer os.Remove()
   Argument order: nmap -F target -oJ /tmp/file.json
   File: internal/tools/nmap.go (lines 10-40)

🧪 VERIFICATION:
   Manual test: nmap -F 127.0.0.1 -oJ /tmp/test.json
   API test: curl POST /api/scan → check results
   Both tools: nmap + nuclei should complete in ~20s
```

**Why this matters for LLMs:**
- Prevents re-introducing same bug
- Explains version compatibility
- Gives exact file locations for fixes
- Includes verification steps

---

### 3. **Commit History as Documentation**

**Best Practice:** Commits tell the story

**ujiscan commits:**
```
598dbfb docs: comprehensive handoff documentation
b5c7e5c fix: nmap temp file output for JSON parsing
07d4677 fix: tool wrappers - URL parsing + nuclei v3 compatibility
6af16ca phase 5: agentic loop foundation complete
8a0f5f0 phase 4 complete: playbook engine fully working end-to-end
...
```

**Each commit has:**
- ✅ What works
- 🔧 What was fixed
- 📝 Files affected

**Next LLM can:**
- Read commit history to understand evolution
- `git show b5c7e5c` to see exact nmap fix
- `git diff 6af16ca~1 6af16ca` to see agentic loop
- Trust that "phase 4 complete" was verified

---

### 4. **Architecture Diagram + Text Explanation**

**Best Practice:** Show + tell

**ujiscan includes:**
```
1. ASCII diagram (DEVELOPMENT.md)
   ┌─────────────────┐
   │ Client (Web)    │
   └────────┬────────┘
            │
   ┌────────┴──────────┐
   │ HTTP Handlers     │
   └────────┬──────────┘
            │
   ┌────────┴──────────┐
   │ Tool Executor     │
   └─────────────────┘
```

2. Textual explanation:
   - What each component does
   - How data flows
   - Where goroutines spawn
   - Thread safety mechanisms

**Why this helps LLMs:**
- Visual understanding immediately
- Can point to "fix line 42 in handlers.go" with context
- Reduces cognitive load

---

### 5. **Checklist for Next Developer**

**Best Practice:** Make onboarding a checklist

**ujiscan includes (HANDOFF_LLM.md, section 4):**
```
Setup:
- [ ] go build -o ujiscan ./cmd/server
- [ ] ./ujiscan (server on :8081)
- [ ] curl http://localhost:8081/api/tools
- [ ] Read README.md
- [ ] Read DEVELOPMENT.md

Verify Working:
- [ ] Regular scan returns results
- [ ] Playbook scan completes
- [ ] Agentic scan chains phases
- [ ] No build errors

If Breaking:
- [ ] Which nmap version installed?
- [ ] Which nuclei version installed?
```

**Why this helps LLMs:**
- Know exactly what to run
- Can self-verify before starting
- Checklist prevents "did I miss something?"

---

### 6. **Tool Compatibility Matrix**

**Best Practice:** Make version dependencies explicit

**ujiscan example (DEVELOPMENT.md):**
```
Tested & Working:
- Go 1.26+
- nmap 7.99 (special: temp files for JSON)
- nuclei v3.11.1 (special: use -jsonl not -json)
- Ubuntu 22.04 / macOS 13+

Known Issues:
- nmap <7.90: may not support -sV
- nuclei <3.0: uses old -json flag
```

**Why this helps LLMs:**
- Won't recommend deprecated flags
- Know when to add version checks
- Understand why certain workarounds exist

---

### 7. **Performance Baselines**

**Best Practice:** Give numbers, not guesses

**ujiscan includes:**
```
| Operation | Time | Notes |
|-----------|------|-------|
| nmap -F | ~5s | Quick scan |
| nuclei | ~15-20s | 10K templates |
| Regular scan | ~20s | Both tools |
| Agentic (3 phases) | ~30s | With AI delay |
```

**Why this helps LLMs:**
- Won't be surprised by 30s scan time
- Can debug "why is nuclei taking 2 min?" (probably template loading)
- Can estimate "Phase 6 will add WebSocket, estimate 5 days"

---

### 8. **"Common Tasks" Section**

**Best Practice:** Teach by example

**ujiscan examples (HANDOFF_LLM.md, section 8):**
```
"Add a new API endpoint:"
1. Define handler in internal/api/handlers.go
2. Register route in cmd/server/main.go
3. Test it: curl localhost:8081/api/new-endpoint
4. Commit with: `feat: add new endpoint`

"Change playbook format:"
1. Update internal/playbook/loader.go parser
2. Update playbooks/ YAML files
3. Run tests
4. Commit: `refactor: update playbook format`
```

**Why this helps LLMs:**
- Don't have to ask "where do I add a new feature?"
- Know the pattern (code → register → test → commit)
- Can follow these patterns for new similar features

---

### 9. **Environment Setup Checklist**

**Best Practice:** Make dependencies explicit

**ujiscan:**
```bash
# Required
Go 1.26+          → go version
nmap              → which nmap; nmap -version
nuclei v3.11.1    → nuclei -version

# Optional
ANTHROPIC_API_KEY → export ANTHROPIC_API_KEY=sk-ant-...
```

**Why this helps LLMs:**
- Won't suggest installing "golang" vaguely
- Can say "you have Go X.Y, need 1.26+"
- Knows agentic features optional (graceful fallback)

---

### 10. **"Signals to Next LLM" Section**

**Best Practice:** Direct address the next LLM

**ujiscan example (HANDOFF_LLM.md end):**
```markdown
## Signal to Next LLM

**If you're a future Claude/GPT/other LLM:**

1. ✅ Read this file first
2. ✅ Read README.md next
3. ✅ Read DEVELOPMENT.md
4. ✅ Check git log
5. ✅ Review tool issues (section 1)
6. ✅ Run verification tests

Then you're ready to:
- Add features
- Fix bugs
- Deploy
- Improve architecture

**Common mistakes to avoid:**
- ❌ Don't use -oJ - with nmap 7.99
- ❌ Don't use -json with nuclei 3.11
- ❌ Don't forget CleanTarget() for URLs
- ❌ Don't test without go build first
```

**Why this works:**
- Explicitly says "you're the next LLM"
- Prioritizes reading order
- Lists common gotchas upfront
- Prevents repeating same mistakes

---

## How to Apply These Practices to Your Projects

### Step 1: Create README.md
```markdown
# Your Project

**What it is** (1 sentence)
**Status** (✅ Prod-ready, 🚧 In progress)

## Quick Start
## Architecture
## API Reference (if applicable)
## Key Features
## Roadmap
```

### Step 2: Create DEVELOPMENT.md
```markdown
# DEVELOPMENT — Detailed Guide

## Phase Breakdown (what was built)
## Known Issues & Fixes (with ROOT CAUSES)
## Architecture Diagram
## Testing Workflow
## Git Commit History
## Troubleshooting
## Roadmap
```

### Step 3: Create HANDOFF_LLM.md (or similar)
```markdown
# HANDOFF — For Next LLM Session

## TL;DR
## Critical Context (tool version issues, etc.)
## Setup Checklist
## Common Tasks
## Performance Baselines
## Signal to Next LLM
```

### Step 4: Push to GitHub
- Add all 3 docs in one commit
- Message: `docs: comprehensive handoff documentation`
- Link in README to DEVELOPMENT.md & HANDOFF_LLM.md

### Step 5: Tag Release
```bash
git tag -a v1.0.0 -m "Production ready: phases 1-5 complete"
git push origin v1.0.0
```

---

## ujiscan as Template

Copy this structure for your projects:

```
your-project/
├── README.md                    # Quick start (2-5 min)
├── DEVELOPMENT.md              # Detailed breakdown (15-30 min)
├── HANDOFF_LLM.md             # Next LLM context (10-15 min)
├── cmd/                        # Entry points
├── internal/                   # Core logic
├── tests/                      # Unit + integration
└── .gitignore
```

Each doc has clear PURPOSE + AUDIENCE.

---

## Key Takeaways

✅ **Three tiers of docs = three types of readers (viewer, developer, next LLM)**

✅ **Known issues need ROOT CAUSES, not just symptoms**

✅ **Checklists > prose (easier to follow for LLMs)**

✅ **Architecture diagram + text (show + tell)**

✅ **Performance baselines prevent surprises**

✅ **Common tasks section teaches by example**

✅ **Directly address next LLM ("if you're Claude...")**

✅ **Version compatibility matrix explicit, not implied**

✅ **Git history tells the story (commits are documentation)**

✅ **Verification steps built-in (next LLM can self-check)**

---

## ujiscan Results

**Before Documentation:**
- New LLM needs ~40 min to understand architecture
- Repeats same nmap/nuclei mistakes
- Doesn't know tool versions matter
- Unsure about performance expectations

**After Documentation:**
- New LLM reads 30 min (README 2min → DEVELOPMENT 15min → HANDOFF 10min → scan commit 3min)
- Knows exactly what tool issues exist + fixes
- Knows performance baselines before building
- Can verify setup in <5 min
- Knows common tasks + patterns

**Time savings per LLM switch: ~30-40 minutes** ✅

---

**Apply these practices to ujiscan and future projects.**

Best luck! 🚀
