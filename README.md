# kurt_v1

A Starship-like, fast modular shell prompt (initially for zsh on macOS), with a focus on “integrations-first” modules and performance.

## Goals
- Fast prompt render (low ms budget)
- Modular segments (dir, git, runtime versions, kube/aws, etc.)
- Simple install + init for zsh
- Config-driven (TOML) formatting

## Install

### One-line install (recommended)

```sh
curl -fsSL https://raw.githubusercontent.com/strk/kurt/main/install.sh | sh
```

This downloads the latest release binary for your platform (macOS or Linux,
amd64 or arm64) and installs it to `/usr/local/bin/kurt` (or
`$HOME/.local/bin/kurt` if `/usr/local/bin` is not writable).

To pin a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/strk/kurt/main/install.sh | KURT_VERSION=v1.2.3 sh
```

### Homebrew (macOS / Linux)

```sh
brew install strk/tap/kurt
```

### From source

```sh
git clone https://github.com/strk/kurt.git
cd kurt
go build -o kurt .
```

## Quick start

After installing, wire kurt into your shell:

```sh
kurt init zsh >> ~/.zshrc && source ~/.zshrc
```

Then run `kurt doctor` to check your setup and see installation hints if
anything is missing.

## Status
- Scaffold + spec drafted. Implementation pending language choice (Rust vs Go).

## Next
- Pick implementation language (Rust recommended)
- Implement `kurt prompt` command + zsh `precmd/preexec` hooks
