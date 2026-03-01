# ROADMAP — kurt_v1

## Phase 0 — decisions (today)
- [ ] Choose language: Rust vs Go
- [ ] Decide prompt style: one-line vs two-line (spec assumes two-line)
- [ ] Pick top integrations for v1.1 (node/python/kube/aws?)

## Phase 1 — MVP prompt engine
- [ ] Implement CLI `kurt prompt`
- [ ] Implement modules: dir, git(branch+dirty), exit, duration
- [ ] Add `KURT_DEBUG=1` to print timings to stderr

## Phase 2 — zsh integration
- [ ] `kurt init zsh` prints snippet
- [ ] preexec/precmd hooks capture exit code + duration
- [ ] RPROMPT support (optional)

## Phase 3 — config
- [ ] TOML config load + defaults
- [ ] Per-module enable/format/threshold/timeout

## Phase 4 — integrations
- [ ] python venv
- [ ] node version
- [ ] kube context (cached)
- [ ] battery/time

## Phase 5 — packaging
- [ ] Build release binary
- [ ] Install script

