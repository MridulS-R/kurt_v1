package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "init",
		Short: "Print shell init snippet",
	}

	c.AddCommand(initZshCmd())
	return c
}

func initZshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Print zsh snippet for precmd/preexec hooks",
		Run: func(cmd *cobra.Command, args []string) {
			// Keep it copy/paste friendly.
			fmt.Printf("%s", `# kurt_v1 (kurt) prompt init
# Add this to your ~/.zshrc

# Path to kurt binary (adjust if needed)
# export PATH="$HOME/.local/bin:$PATH"

function __kurt_preexec() {
  # Zsh passes the full command line as $1
  export __KURT_CMD_START_MS=$(python3 - <<'PY'
import time
print(int(time.time()*1000))
PY
)
  export __KURT_LAST_CMD="$1"
}

function __kurt_precmd() {
  local exit_code=$?
  local now_ms=$(python3 - <<'PY'
import time
print(int(time.time()*1000))
PY
)
  local start_ms=${__KURT_CMD_START_MS:-$now_ms}
  local dur_ms=$(( now_ms - start_ms ))

  export __KURT_LAST_EXIT=$exit_code
  export __KURT_LAST_DURATION_MS=$dur_ms

  # Prompt: first line context, second line input
  local p=$(kurt prompt --shell zsh --cwd "$PWD" --status $exit_code --duration-ms $dur_ms)
  PROMPT="$p"

  # Right prompt (optional, controlled by config)
  local rp=$(kurt rprompt --shell zsh --cwd "$PWD" --status $exit_code --duration-ms $dur_ms 2>/dev/null)
  RPROMPT="$rp"
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec __kurt_preexec
add-zsh-hook precmd __kurt_precmd

# --- Inline suggestions (kurt suggest) ---
# Requires: zsh line editor (ZLE). Shows hint as faint inline text.
# Accept with Right Arrow.

# Enable/disable
: ${KURT_SUGGEST:=1}

function __kurt_suggest_update() {
  if [[ "$KURT_SUGGEST" != "1" ]]; then
    POSTDISPLAY=""
    return
  fi
  # Only suggest when cursor is at end (simple, avoids complex editing cases)
  if (( CURSOR != ${#BUFFER} )); then
    POSTDISPLAY=""
    return
  fi
  local s=$(kurt suggest --buffer "$BUFFER" --cwd "$PWD" 2>/dev/null)
  if [[ -n "$s" ]]; then
    # faint gray (ANSI 256 fg)
    POSTDISPLAY=$'\e[38;5;244m'"$s"$'\e[0m'
  else
    POSTDISPLAY=""
  fi
}

function __kurt_accept_suggest() {
  if [[ -n "$POSTDISPLAY" ]]; then
    # strip color codes by re-running suggest (plain) for accurate append
    local s=$(kurt suggest --buffer "$BUFFER" --cwd "$PWD" 2>/dev/null)
    BUFFER+="$s"
    CURSOR=${#BUFFER}
    POSTDISPLAY=""
    zle redisplay
  else
    zle forward-char
  fi
}

# Hook updates during redraw
zle -N zle-line-pre-redraw __kurt_suggest_update
# Bind right arrow
zle -N __kurt_accept_suggest __kurt_accept_suggest
bindkey '^[[C' __kurt_accept_suggest
`)
		},
	}
}
