# kurt

A fast, modular shell prompt with built-in AI — think, chat, diff review, RAG, image analysis, cost tracking, and more. Works with Anthropic, OpenAI, Ollama, Groq, Together, and any OpenAI-compatible API.

```
~/projects/myapp  main ✗  2.3s  ❯
```

---

## Install

### One-line (macOS & Linux — recommended)

```sh
curl -fsSL https://raw.githubusercontent.com/MridulS-R/kurt_v1/main/install.sh | sh
```

Detects your OS and architecture, downloads the right binary from the latest release, installs to `/usr/local/bin/kurt` (falls back to `~/.local/bin/kurt` if not writable).

Pin a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/MridulS-R/kurt_v1/main/install.sh | KURT_VERSION=v0.1.0 sh
```

### From source

Requires Go 1.22+:

```sh
git clone https://github.com/MridulS-R/kurt_v1.git
cd kurt_v1
go build -o kurt .
sudo mv kurt /usr/local/bin/
```

---

## Quick start

**1. Wire into your shell**

```sh
# zsh
kurt init zsh >> ~/.zshrc && source ~/.zshrc

# bash
kurt init bash >> ~/.bashrc && source ~/.bashrc

# fish
kurt init fish >> ~/.config/fish/conf.d/kurt.fish
```

**2. Check your setup**

```sh
kurt doctor
```

**3. Ask an AI question right from your terminal**

```sh
kurt think "what does this error mean?" --last-cmd
```

---

## Prompt modules

| Module | What it shows | Trigger |
|--------|--------------|---------|
| `dir` | Current directory (shortened) | always |
| `git` | Branch, dirty flag, ahead/behind | inside git repo |
| `duration` | Last command time | commands > 500ms |
| `exit` | Exit code | non-zero exit |
| `node` | Node.js version | `.nvmrc` / `.node-version` |
| `python` | Python version | `.python-version` / `pyvenv.cfg` |
| `venv` | Active virtualenv | `$VIRTUAL_ENV` |
| `conda` | Active conda env | `$CONDA_DEFAULT_ENV` |
| `kube` | kubectl context | `~/.kube/config` |
| `cloud` | AWS / GCP / Azure | profile env vars |
| `battery` | Battery % | macOS / Linux |
| `gpu` | GPU usage % | macOS / Linux |
| `time` | Current time | configurable |

---

## AI commands

```sh
kurt think "explain this code"           # ask with shell context
kurt diff                                # AI review of your staged changes
kurt vision image.png "what's in this?" # ask about an image
kurt pipe "fix the bug" < error.log     # pipe stdin through an LLM
kurt bench "write a haiku"              # compare providers side-by-side
kurt eval dataset.csv --prompt review   # batch-evaluate a prompt on CSV data
kurt rag index ./docs                   # index local docs
kurt rag query "how does auth work?"    # search indexed docs
```

---

## Configuration

Config lives at `~/.config/kurt/config.toml`. Generate a starter:

```sh
kurt config view
```

Example:

```toml
style = "minimal"   # or "powerline"

[think]
provider = "anthropic"   # anthropic | openai | ollama | groq | together
model    = "claude-sonnet-4-6"

[module.git]
enabled = true

[module.node]
enabled = true

[module.kube]
enabled = false
```

Set individual keys:

```sh
kurt config set think.provider anthropic
kurt config set think.model claude-sonnet-4-6
```

---

## All commands

```
kurt think       Ask an AI with shell context
kurt agent       Run an AI agent in a sandbox
kurt diff        AI review of git diff
kurt vision      Ask about an image
kurt bench       Compare providers on a prompt
kurt eval        Batch-evaluate a prompt on a CSV
kurt pipe        Pipe stdin through an LLM
kurt rag         Index and query local docs (RAG)
kurt session     Persistent multi-turn conversations
kurt recall      Search shell command history
kurt prompts     Manage reusable prompt templates
kurt tokens      Count tokens (no API call)
kurt cost        Show API usage and cost
kurt models      List and manage local LLM models
kurt cache       Manage LLM response cache
kurt config      View and edit config
kurt explain     Debug active config and modules
kurt doctor      Check environment health
kurt update      Self-update to latest release
kurt init        Print shell init snippet (zsh/bash/fish)
```

---

## Providers

| Provider | Env var | Notes |
|----------|---------|-------|
| Anthropic | `ANTHROPIC_API_KEY` | Default: `claude-sonnet-4-6` |
| OpenAI | `OPENAI_API_KEY` | Default: `gpt-4o` |
| Ollama | — | Local, default host `localhost:11434` |
| Groq | `GROQ_API_KEY` | Set `--provider groq` |
| Together | `TOGETHER_API_KEY` | Set `--provider together` |
| OpenRouter | `OPENROUTER_API_KEY` | Set `--provider openrouter` |
| LM Studio | — | Set `--base-url http://localhost:1234/v1` |

---

## Requirements

- macOS or Linux (Windows: partial support)
- Go 1.22+ (only if building from source)
- An API key for whichever AI provider you use, **or** [Ollama](https://ollama.com) for fully local AI
