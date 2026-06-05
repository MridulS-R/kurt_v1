# ROADMAP — kurt_v1

## Phase 0 — decisions ✅
- [x] Choose language: Go
- [x] Decide prompt style: two-line (configurable)
- [x] Top integrations for v1.1: node/python/kube/aws

## Phase 1 — MVP prompt engine ✅
- [x] Implement CLI `kurt prompt`
- [x] Implement modules: dir, git (branch+dirty+ahead/behind), exit, duration
- [x] `KURT_DEBUG=1` prints timings to stderr

## Phase 2 — zsh integration ✅
- [x] `kurt init zsh` prints snippet (uses EPOCHREALTIME, no python3)
- [x] preexec/precmd hooks capture exit code + duration
- [x] RPROMPT support (`kurt rprompt`)

## Phase 3 — config ✅
- [x] TOML config at `~/.config/kurt/config.toml`
- [x] Per-module enable/format/threshold
- [x] Module order, colors (minimal + powerline styles)
- [x] `--no-color` flag on prompt + rprompt

## Phase 3.5 — extras ✅
- [x] Powerline style rendering (config `style = "powerline"`)
- [x] Inline command suggestions (`kurt suggest`) + zsh right-arrow accept
- [x] `kurt think` — Ollama AI assistant with git+last-command context
- [x] `kurt explain` — debug config dump
- [x] `kurt version`

## Phase 4 — integrations (next)
- [ ] `node` version module (detect package.json, cached node --version)
- [ ] `python` venv module ($VIRTUAL_ENV name)
- [ ] `kube` context module (read ~/.kube/config directly, no kubectl spawn)
- [ ] `battery` module (macOS pmset)
- [ ] `time` module for left prompt (currently only in RPROMPT)

## Phase 5 — packaging
- [ ] Build release binary
- [ ] Install script (`curl | sh`)
- [ ] Homebrew formula
