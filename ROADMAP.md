# ROADMAP — kurt_v1

## Phase 0 — decisions (today)
- [x] Choose language: Rust vs Go
- [x] Decide prompt style: one-line vs two-line (spec assumes two-line)
- [x] Pick top integrations for v1.1 (node/python/kube/aws?)

## Phase 1 — MVP prompt engine
- [x] Implement CLI `kurt prompt`
- [x] Implement modules: dir, git(branch+dirty), exit, duration
- [x] Add `KURT_DEBUG=1` to print timings to stderr

## Phase 2 — zsh integration
- [x] `kurt init zsh` prints snippet
- [x] preexec/precmd hooks capture exit code + duration
- [x] RPROMPT support (optional)

## Phase 3 — config
- [x] TOML config load + defaults
- [x] Per-module enable/format/threshold/timeout

## Phase 4 — integrations
- [x] python venv (venv + conda modules)
- [x] node version (node module — reads .nvmrc / .node-version / NODE_VERSION)
- [x] kube context — pure Go YAML parsing of ~/.kube/config or $KUBECONFIG; no kubectl subprocess
- [x] battery — macOS pmset / Linux sysfs; threshold config; suppressed when charging at 100%
- [x] time module (wall-clock, configurable format)
- [x] python version module (reads .python-version / PYTHON_VERSION)
- [x] conda environment module (suppresses "base" by default)
- [x] cloud module (AWS_PROFILE, GCP config, Azure subscription)

## Phase 5 — packaging
- [ ] Build release binary
- [ ] Install script

## Phase 6 — polish & ecosystem
- [ ] Release pipeline (GitHub Actions + install.sh)
- [ ] Homebrew formula
- [ ] Windows support (best-effort: disable gpu/battery modules)
- [ ] Shell completions (cobra generates these automatically via `kurt completion zsh`)
- [ ] Web UI for cost dashboard
