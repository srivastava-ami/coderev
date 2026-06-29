package llm

import (
	"context"
	"fmt"
	"strings"
)

type ChunkProgress struct {
	N     int
	Total int
	File  string
	Est   int
}

func ReviewChunked(ctx context.Context, provider Provider, chunks []ReviewChunk, progress func(ChunkProgress)) (string, TokenUsage, error) {
	if len(chunks) == 0 {
		return "", TokenUsage{}, nil
	}

	var result strings.Builder
	var totalUsage TokenUsage

	for i, chunk := range chunks {
		prompt := AssemblePrompt(chunk.Ctx)

		if progress != nil {
			progress(ChunkProgress{
				N:     i + 1,
				Total: len(chunks),
				File:  chunk.File,
				Est:   len(prompt) / 4,
			})
		}

		review, usage, err := provider.Complete(ctx, prompt)
		if err != nil {
			return "", TokenUsage{}, fmt.Errorf("chunk %d (%s): %w", i+1, chunk.File, err)
		}

		totalUsage.InputTokens += usage.InputTokens
		totalUsage.OutputTokens += usage.OutputTokens

		if len(chunks) > 1 {
			result.WriteString("### ")
			result.WriteString(chunk.File)
			result.WriteString("\n\n")
		}
		result.WriteString(review)
		result.WriteString("\n\n")
	}

	return strings.TrimSpace(result.String()), totalUsage, nil
}
