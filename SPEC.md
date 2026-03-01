# SPEC — kurt_v1 prompt engine

## Scope (v1)
- Shells: zsh (macOS) first.
- Binary: single executable `kurt`.
- Commands:
  - `kurt prompt --shell zsh --cwd <path> --status <code> --duration-ms <n>` → prints prompt string
  - `kurt init zsh` → prints a snippet to add to `~/.zshrc`
  - `kurt explain` (debug) → prints module timings and what was shown

## Prompt layout (recommended defaults)
Two-line prompt (readable + extensible):

Line 1 (context):
- left: `dir` + `git` + (optional) `python` + `node` + `kube`
- right: `duration` + `time` + `battery`

Line 2 (input):
- left: prompt char that changes by status (e.g., `❯` / `✗`)

## Modules (v1 MVP)
### Required
1. `dir`
   - shows shortened cwd, ~ expansion
2. `git`
   - shows branch
   - shows dirty flag
   - optionally ahead/behind (later)
3. `exit`
   - show only on non-zero exit
4. `duration`
   - show when duration exceeds threshold (e.g. 500ms)

### Optional (v1.1+)
- `node` (nvm/asdf detection, package.json presence)
- `python` (venv name)
- `kube` (kubectl context/namespace, cached)
- `aws` (AWS_PROFILE/region, cached)
- `docker` (context)
- `battery` (macOS)

## Performance rules
- Hard budget: target < 30ms typical, < 100ms worst-case.
- Any slow module must be:
  - cached, AND
  - time-limited (timeout per module), AND
  - optionally async-refreshed.

## Config
- Path: `~/.config/kurt/config.toml`
- Controls:
  - module enable/disable
  - order
  - colors
  - thresholds (duration)
  - timeouts

## Output
- ANSI colors by default.
- Provide a `--no-color` flag (later).

