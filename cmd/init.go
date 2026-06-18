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
	c.AddCommand(initBashCmd())
	c.AddCommand(initFishCmd())
	return c
}

func initBashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Print bash snippet for PROMPT_COMMAND integration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s", `# kurt prompt init for bash
# Add this to your ~/.bashrc or ~/.bash_profile

__kurt_last_cmd=""
__kurt_cmd_start_ms=0

__kurt_preexec() {
  __kurt_last_cmd="$BASH_COMMAND"
  __kurt_cmd_start_ms=$(date +%s%3N 2>/dev/null || echo 0)
}
trap '__kurt_preexec' DEBUG

__kurt_precmd() {
  local exit_code=$?
  local now_ms
  now_ms=$(date +%s%3N 2>/dev/null || echo 0)
  local dur_ms=$(( now_ms - __kurt_cmd_start_ms ))

  export KURT_LAST_EXIT=$exit_code
  export KURT_LAST_DURATION_MS=$dur_ms

  if [[ $exit_code -ne 0 && -n "$__kurt_last_cmd" ]]; then
    kurt log-failure --exit $exit_code --cwd "$PWD" "$__kurt_last_cmd" >/dev/null 2>&1 &
  fi

  if [[ -n "$__kurt_last_cmd" ]]; then
    kurt log-cmd --exit $exit_code --cwd "$PWD" --duration-ms $dur_ms "$__kurt_last_cmd" >/dev/null 2>&1 &
  fi

  local p
  p=$(kurt prompt --shell bash --cwd "$PWD" --status $exit_code --duration-ms $dur_ms 2>/dev/null)
  PS1="$p"
  __kurt_last_cmd=""
}

PROMPT_COMMAND="__kurt_precmd"
`)
		},
	}
}

func initFishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fish",
		Short: "Print fish functions for prompt integration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s", `# kurt prompt init for fish
# Save to ~/.config/fish/conf.d/kurt.fish

set -g __kurt_cmd_start_ms 0
set -g __kurt_last_cmd ""

function __kurt_preexec --on-event fish_preexec
    set __kurt_last_cmd $argv[1]
    set __kurt_cmd_start_ms (date +%s%3N 2>/dev/null; or echo 0)
end

function fish_prompt
    set -l exit_code $status
    set -l now_ms (date +%s%3N 2>/dev/null; or echo 0)
    set -l dur_ms (math $now_ms - $__kurt_cmd_start_ms)

    if test $exit_code -ne 0; and test -n "$__kurt_last_cmd"
        kurt log-failure --exit $exit_code --cwd (pwd) "$__kurt_last_cmd" >/dev/null 2>&1 &
    end

    if test -n "$__kurt_last_cmd"
        kurt log-cmd --exit $exit_code --cwd (pwd) --duration-ms $dur_ms "$__kurt_last_cmd" >/dev/null 2>&1 &
    end

    set __kurt_last_cmd ""
    kurt prompt --shell fish --cwd (pwd) --status $exit_code --duration-ms $dur_ms 2>/dev/null
end

function fish_right_prompt
    kurt rprompt --shell fish --cwd (pwd) --status $status 2>/dev/null
end

# Inline suggestion (fish has native autosuggestions; kurt suggest can feed them)
# Uncomment to override fish's built-in suggestions with kurt suggest:
# function fish_command_not_found
#     echo "kurt: command not found: $argv"
# end
`)
		},
	}
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

# Load zsh datetime module (provides EPOCHREALTIME) — safe to call multiple times.
zmodload zsh/datetime

# Milliseconds since epoch using zsh's built-in EPOCHREALTIME (no subprocess).
# EPOCHREALTIME = "1234567890.123456" → seconds.microseconds
__kurt_ms() { echo $(( ${EPOCHREALTIME%.*} * 1000 + ${EPOCHREALTIME#*.} / 1000 )) }

function __kurt_preexec() {
  export __KURT_CMD_START_MS=$(__kurt_ms)
  export __KURT_LAST_CMD="$1"
}

function __kurt_precmd() {
  local exit_code=$?
  local now_ms=$(__kurt_ms)
  local start_ms=${__KURT_CMD_START_MS:-$now_ms}
  local dur_ms=$(( now_ms - start_ms ))

  export __KURT_LAST_EXIT=$exit_code
  export __KURT_LAST_DURATION_MS=$dur_ms

  # Log failures so kurt think can learn from them.
  if [[ $exit_code -ne 0 && -n "$__KURT_LAST_CMD" ]]; then
    kurt log-failure --exit $exit_code --cwd "$PWD" "$__KURT_LAST_CMD" &>/dev/null &!
  fi

  # Log all commands for kurt recall (shell memory).
  if [[ -n "$__KURT_LAST_CMD" ]]; then
    kurt log-cmd --exit $exit_code --cwd "$PWD" --duration-ms $dur_ms "$__KURT_LAST_CMD" &>/dev/null &!
  fi

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
  # Only suggest when cursor is at end
  if (( CURSOR != ${#BUFFER} )); then
    POSTDISPLAY=""
    return
  fi
  local s=$(kurt suggest --buffer "$BUFFER" --cwd "$PWD" 2>/dev/null)
  POSTDISPLAY="${s}"
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

# Color the suggestion suffix (POSTDISPLAY) using ZLE's highlight array.
# This is how zsh-autosuggestions colors suggestions — no raw ANSI needed.
zle_highlight+=( suffix:fg=244 )

# Hook updates during redraw
zle -N zle-line-pre-redraw __kurt_suggest_update
# Bind right arrow
zle -N __kurt_accept_suggest __kurt_accept_suggest
bindkey '^[[C' __kurt_accept_suggest
`)
		},
	}
}
