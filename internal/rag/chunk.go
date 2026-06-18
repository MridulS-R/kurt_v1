package rag

import (
	"strings"
	"unicode"
)

const (
	defaultChunkSize    = 600 // chars (~150 tokens)
	defaultChunkOverlap = 80
)

// Chunk splits text into overlapping segments.
func Chunk(text string, size, overlap int) []string {
	if size <= 0 {
		size = defaultChunkSize
	}
	if overlap < 0 {
		overlap = defaultChunkOverlap
	}

	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return nil
	}

	// Split into paragraphs first for natural boundaries.
	paragraphs := splitParagraphs(text)

	var chunks []string
	var current strings.Builder

	flush := func() {
		s := strings.TrimSpace(current.String())
		if s != "" {
			chunks = append(chunks, s)
		}
		current.Reset()
	}

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if current.Len()+len(para)+1 > size {
			if current.Len() > 0 {
				flush()
				// Carry overlap from last chunk into next.
				if overlap > 0 && len(chunks) > 0 {
					last := chunks[len(chunks)-1]
					words := strings.Fields(last)
					var tail strings.Builder
					// Take last ~overlap chars worth of words.
					i := len(words) - 1
					for i >= 0 && tail.Len() < overlap {
						if tail.Len() > 0 {
							tail.WriteString(" ")
						}
						tail.WriteString(words[i])
						i--
					}
					tailStr := tail.String()
					// Reverse word order
					tailWords := strings.Fields(tailStr)
					for left, right := 0, len(tailWords)-1; left < right; left, right = left+1, right-1 {
						tailWords[left], tailWords[right] = tailWords[right], tailWords[left]
					}
					current.WriteString(strings.Join(tailWords, " "))
				}
			}
			// If a single paragraph exceeds size, hard-split it.
			if len(para) > size {
				parts := hardSplit(para, size, overlap)
				for _, p := range parts {
					chunks = append(chunks, p)
				}
				continue
			}
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}
	flush()
	return chunks
}

func splitParagraphs(text string) []string {
	var out []string
	for _, block := range strings.Split(text, "\n\n") {
		block = strings.TrimFunc(block, unicode.IsSpace)
		if block != "" {
			out = append(out, block)
		}
	}
	return out
}

func hardSplit(text string, size, overlap int) []string {
	var out []string
	runes := []rune(text)
	step := size - overlap
	if step <= 0 {
		step = size
	}
	for i := 0; i < len(runes); i += step {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[i:end]))
		if end == len(runes) {
			break
		}
	}
	return out
}
