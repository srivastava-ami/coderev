package analysis

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestStreamSourceFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"main.go":   "package main\nfunc main() {}\n",
		"util.go":   "package util\nfunc Help() {}\n",
		"app.ts":    "const x: number = 1;\n",
		"ignored.txt": "not a recognised source\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	batches, err := collectBatches(dir, 2)
	if err != nil {
		t.Fatal(err)
	}

	// We wrote 3 recognised files; batch size 2 => [2, 1]
	if len(batches) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(batches))
	}
	if len(batches[0]) != 2 {
		t.Fatalf("expected first batch to have 2 files, got %d", len(batches[0]))
	}
	if len(batches[1]) != 1 {
		t.Fatalf("expected second batch to have 1 file, got %d", len(batches[1]))
	}

	// Flatten and compare with CollectSourceFiles
	var streamed []FileInfo
	for _, b := range batches {
		streamed = append(streamed, b...)
	}
	collected, err := CollectSourceFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(streamed) != len(collected) {
		t.Fatalf("StreamSourceFiles yielded %d files, CollectSourceFiles returned %d", len(streamed), len(collected))
	}

	sort.Slice(streamed, func(i, j int) bool { return streamed[i].Path < streamed[j].Path })
	sort.Slice(collected, func(i, j int) bool { return collected[i].Path < collected[j].Path })
	for i := range streamed {
		if streamed[i].Path != collected[i].Path ||
			streamed[i].Language != collected[i].Language ||
			streamed[i].Lines != collected[i].Lines {
			t.Errorf("file %d mismatch: streamed=%+v collected=%+v", i, streamed[i], collected[i])
		}
	}
}

func collectBatches(target string, batchSize int) ([][]FileInfo, error) {
	var batches [][]FileInfo
	err := StreamSourceFiles(target, batchSize, func(batch []FileInfo) error {
		// Copy the batch slice so the caller can't mutate it
		b := make([]FileInfo, len(batch))
		copy(b, batch)
		batches = append(batches, b)
		return nil
	})
	return batches, err
}
