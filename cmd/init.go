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
			fmt.Print(`# kurt_v1 (kurt) prompt init
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

  # Two-line prompt: first line context, second line input
  local p=$(kurt prompt --shell zsh --cwd "$PWD" --status $exit_code --duration-ms $dur_ms)
  PROMPT="$p"
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec __kurt_preexec
add-zsh-hook precmd __kurt_precmd
`)
		},
	}
}
