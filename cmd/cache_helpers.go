package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"kurt_v1/internal/cache"
	"kurt_v1/internal/think"
)

// runWithCache checks the response cache before calling the provider.
// If a cached response exists it is printed immediately; otherwise the
// live response is streamed and stored for future calls.
func runWithCache(p think.Provider, msgs []think.ChatMsg, cacheInput string, ttlHours int) error {
	providerName := providerTypeName(p)
	modelName := providerModelName(p)
	key := cache.Key(providerName, modelName, cacheInput)

	if resp, ok := cache.Get(key, ttlHours); ok {
		fmt.Fprint(os.Stderr, "[cached] ")
		fmt.Println(resp)
		return nil
	}

	var buf bytes.Buffer
	if err := p.ChatStream(msgs, &buf); err != nil {
		return err
	}
	resp := strings.TrimSpace(buf.String())
	fmt.Println(resp)
	_ = cache.Put(providerName, modelName, cacheInput, resp, ttlHours)
	return nil
}

func providerTypeName(p think.Provider) string {
	switch p.(type) {
	case *think.AnthropicProvider:
		return "anthropic"
	case *think.OpenAIProvider:
		v := p.(*think.OpenAIProvider)
		if v.ProviderName != "" {
			return v.ProviderName
		}
		return "openai-compat"
	case *think.OllamaClient:
		return "ollama"
	default:
		return "unknown"
	}
}

func providerModelName(p think.Provider) string {
	switch v := p.(type) {
	case *think.AnthropicProvider:
		return v.Model
	case *think.OpenAIProvider:
		return v.Model
	case *think.OllamaClient:
		return v.Model
	}
	return ""
}
