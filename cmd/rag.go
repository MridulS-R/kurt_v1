package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"kurt_v1/internal/config"
	"kurt_v1/internal/embed"
	"kurt_v1/internal/rag"
	"kurt_v1/internal/think"
)

func ragCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "rag",
		Short: "Retrieval-augmented generation — index docs and query them",
	}
	c.AddCommand(ragIndexCmd())
	c.AddCommand(ragQueryCmd())
	c.AddCommand(ragListCmd())
	c.AddCommand(ragRemoveCmd())
	return c
}

func ragIndexCmd() *cobra.Command {
	var collection string
	var embedProvider, embedModel, embedBaseURL, embedHost string
	var chunkSize int

	c := &cobra.Command{
		Use:   "index <path> [path...]",
		Short: "Index files into a collection",
		Long: `Chunk and embed local files into a named collection for later retrieval.

Supported file types: .txt .md .go .py .js .ts .rs .java .c .cpp .h .rb .sh

Examples:
  kurt rag index ./docs --collection project-docs
  kurt rag index README.md *.go --collection codebase`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, _ := config.Load()
			embedr, err := newEmbedder(embedProvider, embedModel, embedBaseURL, embedHost, cfg)
			if err != nil {
				return err
			}

			err = rag.IndexFiles(collection, args, embedr, chunkSize, func(path string, n int) {
				fmt.Fprintf(os.Stderr, "  indexed %s (%d chunks)\n", path, n)
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Done → collection %q\n", collection)
			return nil
		},
	}

	c.Flags().StringVar(&collection, "collection", "default", "Collection name")
	c.Flags().StringVar(&embedProvider, "embed-provider", "", "Embedding provider (ollama/openai)")
	c.Flags().StringVar(&embedModel, "embed-model", "", "Embedding model override")
	c.Flags().StringVar(&embedBaseURL, "embed-base-url", "", "Embedding API base URL")
	c.Flags().StringVar(&embedHost, "embed-host", "", "Ollama host for embeddings")
	c.Flags().IntVar(&chunkSize, "chunk-size", 512, "Chunk size in tokens (approx)")
	return c
}

func ragQueryCmd() *cobra.Command {
	var collection string
	var embedProvider, embedModel, embedBaseURL, embedHost string
	var topK int
	var answer bool
	var provider, model, baseURL, host string

	c := &cobra.Command{
		Use:   "query <question>",
		Short: "Search the collection and optionally answer with an LLM",
		Long: `Embed the question, find the most similar chunks, and print them.
With --answer, the retrieved chunks are sent to the LLM as context.

Examples:
  kurt rag query "how does authentication work?" --collection project-docs
  kurt rag query "what does init() do?" --collection codebase --answer`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")
			cfg, _, _ := config.Load()

			embedr, err := newEmbedder(embedProvider, embedModel, embedBaseURL, embedHost, cfg)
			if err != nil {
				return err
			}

			queryVec, err := embedr.Embed(question)
			if err != nil {
				return fmt.Errorf("embedding query: %w", err)
			}

			hits, err := rag.Search(collection, queryVec, topK)
			if err != nil {
				return fmt.Errorf("searching collection %q: %w", collection, err)
			}
			if len(hits) == 0 {
				fmt.Println("No results found.")
				return nil
			}

			if !answer {
				for i, h := range hits {
					pct := int(float32(100)*h.Score + 0.5)
					fmt.Printf("── %d  score %d%%  %s (chunk %d) ──\n%s\n\n",
						i+1, pct, h.Source, h.Index, h.Text)
				}
				return nil
			}

			// Build context for LLM
			var ctxBuf strings.Builder
			for _, h := range hits {
				fmt.Fprintf(&ctxBuf, "--- %s ---\n%s\n\n", h.Source, h.Text)
			}
			prompt := fmt.Sprintf("Answer the following question using only the provided context.\n\nContext:\n%s\nQuestion: %s", ctxBuf.String(), question)

			prov, err := think.New(think.ProviderConfig{
				Name:    firstOf(provider, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama"),
				Model:   firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model),
				BaseURL: firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL),
				Host:    firstOf(host, cfg.Think.Host),
			})
			if err != nil {
				return err
			}

			msgs := []think.ChatMsg{{Role: "user", Content: prompt}}
			var buf bytes.Buffer
			if err := prov.ChatStream(msgs, &buf); err != nil {
				return err
			}
			fmt.Println(strings.TrimSpace(buf.String()))
			return nil
		},
	}

	c.Flags().StringVar(&collection, "collection", "default", "Collection to search")
	c.Flags().StringVar(&embedProvider, "embed-provider", "", "Embedding provider")
	c.Flags().StringVar(&embedModel, "embed-model", "", "Embedding model override")
	c.Flags().StringVar(&embedBaseURL, "embed-base-url", "", "Embedding API base URL")
	c.Flags().StringVar(&embedHost, "embed-host", "", "Ollama host for embeddings")
	c.Flags().IntVar(&topK, "top", 5, "Number of chunks to retrieve")
	c.Flags().BoolVar(&answer, "answer", false, "Feed retrieved chunks to LLM and print answer")
	c.Flags().StringVar(&provider, "provider", "", "LLM provider (for --answer)")
	c.Flags().StringVar(&model, "model", "", "LLM model override (for --answer)")
	c.Flags().StringVar(&baseURL, "base-url", "", "LLM base URL (for --answer)")
	c.Flags().StringVar(&host, "host", "", "Ollama host (for --answer)")
	return c
}

func ragListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all indexed collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			cols, err := rag.ListCollections()
			if err != nil {
				return err
			}
			if len(cols) == 0 {
				fmt.Println("No collections. Run: kurt rag index <path>")
				return nil
			}
			fmt.Printf("%-20s  %10s  %s\n", "Collection", "Chunks", "Indexed At")
			fmt.Println(strings.Repeat("─", 50))
			for _, m := range cols {
				fmt.Printf("%-20s  %10d  %s\n", m.Collection, m.ChunkCount, m.IndexedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
}

func ragRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <collection>",
		Short: "Delete a collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rag.RemoveCollection(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Removed collection %q\n", args[0])
			return nil
		},
	}
}

func newEmbedder(providerName, model, baseURL, host string, cfg config.Config) (embed.Embedder, error) {
	name := firstOf(providerName, os.Getenv("KURT_PROVIDER"), cfg.Think.Provider, "ollama")
	m := firstOf(model, os.Getenv("KURT_MODEL"), cfg.Think.Model)
	bu := firstOf(baseURL, os.Getenv("KURT_BASE_URL"), cfg.Think.BaseURL)
	h := firstOf(host, cfg.Think.Host)

	switch name {
	case "openai", "groq", "together", "openrouter", "lmstudio", "openai-compat":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if bu == "" {
			bu = "https://api.openai.com/v1"
		}
		if m == "" {
			m = "text-embedding-3-small"
		}
		return &embed.OpenAIEmbedder{Model: m, BaseURL: bu, APIKey: apiKey}, nil
	default: // ollama
		if h == "" {
			h = envOr("KURT_OLLAMA_HOST", "http://127.0.0.1:11434")
		}
		if m == "" {
			m = "nomic-embed-text"
		}
		return &embed.OllamaEmbedder{Model: m, Host: h}, nil
	}
}
