# kurt_v1

A Starship-like, fast modular shell prompt (initially for zsh on macOS), with a focus on “integrations-first” modules and performance.

## Goals
- Fast prompt render (low ms budget)
- Modular segments (dir, git, runtime versions, kube/aws, etc.)
- Simple install + init for zsh
- Config-driven (TOML) formatting

## Status
- Scaffold + spec drafted. Implementation pending language choice (Rust vs Go).

## Next
- Pick implementation language (Rust recommended)
- Implement `kurt prompt` command + zsh `precmd/preexec` hooks

