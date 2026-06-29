package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockProvider struct {
	reviews   []string
	usages    []TokenUsage
	errAt     int
	callCount int
}

func (m *mockProvider) Complete(_ context.Context, prompt string) (string, TokenUsage, error) {
	i := m.callCount
	m.callCount++
	if i == m.errAt {
		return "", TokenUsage{}, errors.New("mock error")
	}
	if i < len(m.reviews) {
		return m.reviews[i], m.usages[i], nil
	}
	return "", TokenUsage{}, errors.New("unexpected call")
}

func TestReviewChunked_Empty(t *testing.T) {
	result, usage, err := ReviewChunked(context.Background(), nil, nil, nil)
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
	if usage != (TokenUsage{}) {
		t.Errorf("expected zero usage, got %+v", usage)
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestReviewChunked_SingleChunk(t *testing.T) {
	provider := &mockProvider{
		reviews: []string{"looks good"},
		usages:  []TokenUsage{{InputTokens: 10, OutputTokens: 20}},
		errAt:   -1,
	}
	chunks := []ReviewChunk{{File: "a.go", Ctx: ReviewContext{}}}

	result, usage, err := ReviewChunked(context.Background(), provider, chunks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result, "###") {
		t.Errorf("single chunk should not have ### header, got: %s", result)
	}
	if !strings.Contains(result, "looks good") {
		t.Errorf("expected review content, got: %s", result)
	}
	if usage.InputTokens != 10 || usage.OutputTokens != 20 {
		t.Errorf("expected usage 10/20, got %+v", usage)
	}
}

func TestReviewChunked_MultipleChunks(t *testing.T) {
	provider := &mockProvider{
		reviews: []string{"review a", "review b", "review c"},
		usages:  []TokenUsage{{1, 2}, {3, 4}, {5, 6}},
		errAt:   -1,
	}
	chunks := []ReviewChunk{
		{File: "a.go", Ctx: ReviewContext{}},
		{File: "b.go", Ctx: ReviewContext{}},
		{File: "c.go", Ctx: ReviewContext{}},
	}

	var calls []ChunkProgress
	progress := func(p ChunkProgress) {
		calls = append(calls, p)
	}

	result, usage, err := ReviewChunked(context.Background(), provider, chunks, progress)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "### a.go") {
		t.Errorf("expected ### a.go header, got: %s", result)
	}
	if !strings.Contains(result, "### b.go") {
		t.Errorf("expected ### b.go header, got: %s", result)
	}
	if !strings.Contains(result, "### c.go") {
		t.Errorf("expected ### c.go header, got: %s", result)
	}
	if usage.InputTokens != 9 || usage.OutputTokens != 12 {
		t.Errorf("expected usage 9/12, got %+v", usage)
	}
	if len(calls) != 3 {
		t.Fatalf("expected 3 progress calls, got %d", len(calls))
	}
	if calls[0].N != 1 || calls[0].File != "a.go" {
		t.Errorf("unexpected first progress: %+v", calls[0])
	}
	if calls[1].N != 2 || calls[1].File != "b.go" {
		t.Errorf("unexpected second progress: %+v", calls[1])
	}
	if calls[2].N != 3 || calls[2].File != "c.go" {
		t.Errorf("unexpected third progress: %+v", calls[2])
	}
}

func TestReviewChunked_ErrorStops(t *testing.T) {
	provider := &mockProvider{
		reviews: []string{"first ok", "", ""},
		usages:  []TokenUsage{{1, 2}, {}, {}},
		errAt:   1,
	}
	chunks := []ReviewChunk{
		{File: "a.go", Ctx: ReviewContext{}},
		{File: "b.go", Ctx: ReviewContext{}},
		{File: "c.go", Ctx: ReviewContext{}},
	}

	_, _, err := ReviewChunked(context.Background(), provider, chunks, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "chunk 2 (b.go)") {
		t.Errorf("error should reference chunk 2 / b.go, got: %v", err)
	}
}
