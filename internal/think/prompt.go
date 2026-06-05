package think

import (
	"encoding/json"
	"fmt"
	"strings"
)

func BuildPrompt(ctx Context, question string) string {
	ctxJSON, _ := json.MarshalIndent(ctx, "", "  ")

	sys := strings.TrimSpace(`
You are Kurt Think: a local shell AI assistant embedded in the terminal.
You know the user's working directory, git state, project type, recent failures, and environment.
Be concise and practical. Prefer non-destructive commands. If you spot a pattern in recent_failures, mention it.
Format code blocks with backticks. Keep answers short — this is a terminal, not a document.
`)

	return fmt.Sprintf("%s\n\nContext:\n%s\n\nQuestion:\n%s\n", sys, string(ctxJSON), strings.TrimSpace(question))
}
