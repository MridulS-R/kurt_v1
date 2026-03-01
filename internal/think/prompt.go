package think

import (
	"encoding/json"
	"fmt"
	"strings"
)

func BuildPrompt(ctx Context, question string) string {
	// Keep it compact and safe: only include what we explicitly collected.
	ctxJSON, _ := json.MarshalIndent(ctx, "", "  ")

	sys := "You are Kurt Think: a local shell assistant. Be concise, practical, and safe. " +
		"If you suggest shell commands, prefer non-destructive commands and explain what they do."

	user := strings.TrimSpace(question)
	return fmt.Sprintf("%s\n\nContext (JSON):\n%s\n\nQuestion:\n%s\n", sys, string(ctxJSON), user)
}
