package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"github.com/spf13/cobra"
)

func tokensCmd() *cobra.Command {
	var encoding string
	var model string

	c := &cobra.Command{
		Use:   "tokens [text]",
		Short: "Count tokens in text (no API call)",
		Long: `Count how many tokens a piece of text would consume.
Uses the tiktoken BPE tokenizer (same as OpenAI/Claude).

Reads from stdin if no text argument is given.
Vocab files are downloaded on first run and cached in ~/.cache/tiktoken/.

Encodings:
  cl100k_base  GPT-4, GPT-3.5, Claude (default)
  o200k_base   GPT-4o, GPT-4o-mini

Examples:
  kurt tokens "hello world"
  cat prompt.txt | kurt tokens
  git diff | kurt tokens --model gpt-4o
  kurt tokens --encoding o200k_base < large_doc.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			enc := resolveEncoding(encoding, model)

			var text string
			if len(args) > 0 {
				text = strings.Join(args, " ")
			} else {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return fmt.Errorf("provide text as argument or pipe it: echo 'hello' | kurt tokens")
				}
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				text = string(b)
			}

			if strings.TrimSpace(text) == "" {
				return fmt.Errorf("empty input")
			}

			tk, err := tiktoken.GetEncoding(enc)
			if err != nil {
				return fmt.Errorf("loading tokenizer %q: %w\n(hint: requires internet access on first run to download vocab)", enc, err)
			}
			tokens := tk.Encode(text, nil, nil)

			chars := len([]rune(text))
			ratio := 0.0
			if len(tokens) > 0 {
				ratio = float64(chars) / float64(len(tokens))
			}

			fmt.Printf("%d tokens  (%d chars, %.1f chars/token, encoding: %s)\n",
				len(tokens), chars, ratio, enc)
			return nil
		},
	}

	c.Flags().StringVar(&encoding, "encoding", "", "BPE encoding name (cl100k_base, o200k_base)")
	c.Flags().StringVar(&model, "model", "", "Model name — auto-selects encoding (gpt-4o → o200k_base)")
	return c
}

func resolveEncoding(enc, model string) string {
	if enc != "" {
		return enc
	}
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.HasPrefix(m, "gpt-4o") {
		return "o200k_base"
	}
	return "cl100k_base"
}
