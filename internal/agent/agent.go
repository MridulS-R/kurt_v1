package agent

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"kurt_v1/internal/think"
)

const agentSystem = `You are an AI agent with access to a sandboxed shell environment.

When you need to run a command, output it in a fenced shell block:
` + "```shell" + `
command here
` + "```" + `

The output will be shown to you. Keep working until the task is complete.
When done, write DONE: followed by a one-line summary.

Rules:
- One shell block per response
- If a command fails, adapt — do not repeat the same failure
- Write files with cat or echo into the block
- Do not try to reach the network unless the task requires it`

// Block is a fenced code block extracted from model output.
type Block struct {
	Lang    string
	Content string
}

var fenceRe = regexp.MustCompile("(?ms)```([a-zA-Z]*)\n(.*?)```")

func extractBlocks(text string) []Block {
	var out []Block
	for _, m := range fenceRe.FindAllStringSubmatch(text, -1) {
		lang := strings.ToLower(strings.TrimSpace(m[1]))
		content := strings.TrimSpace(m[2])
		if content != "" {
			out = append(out, Block{Lang: lang, Content: content})
		}
	}
	return out
}

func isShell(b Block) bool {
	switch b.Lang {
	case "shell", "bash", "sh", "zsh", "":
		return true
	}
	return false
}

// Runner executes the ReAct agent loop.
type Runner struct {
	Provider think.Provider
	Sandbox  Sandbox
	MaxSteps int
	AutoExec bool // skip confirmation prompts
	Out      io.Writer
	ErrOut   io.Writer
	In       *bufio.Reader
}

func (r *Runner) Run(task string) error {
	messages := []think.ChatMsg{
		{Role: "system", Content: agentSystem},
		{Role: "user", Content: "Task: " + task},
	}

	fmt.Fprintf(r.ErrOut, "Sandbox: %s  path: %s\n\n", r.Sandbox.Kind(), r.Sandbox.Dir())

	for step := 0; step < r.MaxSteps; step++ {
		// ── model response ────────────────────────────────────────────────
		fmt.Fprint(r.ErrOut, "agent> ")
		var buf bytes.Buffer
		if err := r.Provider.ChatStream(messages, io.MultiWriter(r.Out, &buf)); err != nil {
			return err
		}
		fmt.Fprintln(r.Out)

		reply := strings.TrimSpace(buf.String())
		if reply == "" {
			break
		}
		messages = append(messages, think.ChatMsg{Role: "assistant", Content: reply})

		if strings.Contains(reply, "DONE:") {
			break
		}

		// ── execute shell blocks ──────────────────────────────────────────
		blocks := extractBlocks(reply)
		var observations []string

		for _, b := range blocks {
			if !isShell(b) {
				continue
			}

			if !r.AutoExec {
				fmt.Fprintf(r.ErrOut, "\n[run in %s?] %s\ny/N> ", r.Sandbox.Kind(), b.Content)
				line, _ := r.In.ReadString('\n')
				line = strings.TrimSpace(strings.ToLower(line))
				if line != "y" && line != "yes" {
					observations = append(observations, fmt.Sprintf("$ %s\n[skipped by user]", b.Content))
					continue
				}
			}

			fmt.Fprintf(r.ErrOut, "\n$ %s\n", b.Content)
			output, err := r.Sandbox.Run(b.Content)
			exitNote := "exit: 0"
			if err != nil {
				exitNote = "exit: error — " + err.Error()
			}
			if output == "" {
				output = "(no output)"
			}
			fmt.Fprint(r.Out, output)
			obs := fmt.Sprintf("$ %s\n%s\n[%s]", b.Content, strings.TrimRight(output, "\n"), exitNote)
			observations = append(observations, obs)
		}

		if len(observations) == 0 {
			// Model produced no runnable blocks — treat as final answer.
			break
		}

		// Feed observations back for next iteration.
		messages = append(messages, think.ChatMsg{
			Role:    "user",
			Content: "Output:\n" + strings.Join(observations, "\n\n") + "\n\nContinue.",
		})
	}
	return nil
}
