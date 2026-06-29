package llm

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func TestChunkByFile_withHunks(t *testing.T) {
	tests := []struct {
		name    string
		rc      ReviewContext
		want    []string // expected chunk File values in order
		wantLen int
	}{
		{
			name: "group by hunk file",
			rc: ReviewContext{
				BaseRef: "main",
				Hunks: []DiffHunk{
					{File: "a.go", Header: "@@ -1 +1 @@", Content: "+a\n"},
					{File: "b.go", Header: "@@ -1 +1 @@", Content: "+b\n"},
					{File: "a.go", Header: "@@ -2 +2 @@", Content: "+a2\n"},
				},
				Findings: []analysis.Finding{
					{Rule: "r1", File: "a.go", Line: 1, Message: "m1"},
					{Rule: "r2", File: "src/b.go", Line: 2, Message: "m2"},
				},
				Neighbors: []GraphNeighbor{
					{ID: "a1", File: "a.go", Label: "A"},
					{ID: "b1", File: "b.go", Label: "B"},
				},
			},
			want:    []string{"a.go", "b.go"},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkByFile(tt.rc)
			if len(chunks) != tt.wantLen {
				t.Fatalf("got %d chunks, want %d", len(chunks), tt.wantLen)
			}
			for i, f := range tt.want {
				if chunks[i].File != f {
					t.Errorf("chunks[%d].File = %q, want %q", i, chunks[i].File, f)
				}
			}

			if len(tt.want) < 1 {
				return
			}
			if chunks[0].Ctx.BaseRef != "main" {
				t.Errorf("chunks[0].Ctx.BaseRef = %q", chunks[0].Ctx.BaseRef)
			}
			if len(chunks[0].Ctx.Hunks) != 2 {
				t.Errorf("chunks[0] hunks = %d, want 2", len(chunks[0].Ctx.Hunks))
			}
			if len(chunks[0].Ctx.Findings) != 1 {
				t.Errorf("chunks[0] findings = %d, want 1", len(chunks[0].Ctx.Findings))
			}
			if len(chunks[0].Ctx.Neighbors) != 1 {
				t.Errorf("chunks[0] neighbors = %d, want 1", len(chunks[0].Ctx.Neighbors))
			}

			if len(tt.want) < 2 {
				return
			}
			if len(chunks[1].Ctx.Hunks) != 1 {
				t.Errorf("chunks[1] hunks = %d, want 1", len(chunks[1].Ctx.Hunks))
			}
			if len(chunks[1].Ctx.Findings) != 1 {
				t.Errorf("chunks[1] findings = %d, want 1 (HasSuffix match)", len(chunks[1].Ctx.Findings))
			}
			if len(chunks[1].Ctx.Neighbors) != 1 {
				t.Errorf("chunks[1] neighbors = %d, want 1", len(chunks[1].Ctx.Neighbors))
			}
		})
	}
}

func TestChunkByFile_withoutHunks(t *testing.T) {
	tests := []struct {
		name    string
		rc      ReviewContext
		want    []string
		wantLen int
	}{
		{
			name: "group by neighbor file",
			rc: ReviewContext{
				BaseRef: "main",
				Findings: []analysis.Finding{
					{Rule: "r1", File: "a.go", Line: 1, Message: "m1"},
					{Rule: "r2", File: "b.go", Line: 2, Message: "m2"},
				},
				Neighbors: []GraphNeighbor{
					{ID: "a1", File: "a.go", Label: "A"},
					{ID: "b1", File: "b.go", Label: "B"},
				},
			},
			want:    []string{"a.go", "b.go"},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkByFile(tt.rc)
			if len(chunks) != tt.wantLen {
				t.Fatalf("got %d chunks, want %d", len(chunks), tt.wantLen)
			}
			for i, f := range tt.want {
				if chunks[i].File != f {
					t.Errorf("chunks[%d].File = %q, want %q", i, chunks[i].File, f)
				}
			}
			if chunks[0].Ctx.Hunks != nil {
				t.Errorf("chunks[0] hunks = %v, want nil", chunks[0].Ctx.Hunks)
			}
		})
	}
}

func TestChunkByFile_empty(t *testing.T) {
	tests := []struct {
		name string
		rc   ReviewContext
	}{
		{name: "empty context", rc: ReviewContext{BaseRef: "main"}},
		{name: "zero value", rc: ReviewContext{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkByFile(tt.rc)
			if len(chunks) != 1 {
				t.Fatalf("got %d chunks, want 1", len(chunks))
			}
			if chunks[0].File != "" {
				t.Errorf("chunk.File = %q, want empty", chunks[0].File)
			}
			if chunks[0].Ctx.BaseRef != tt.rc.BaseRef {
				t.Errorf("chunk.Ctx.BaseRef = %q, want %q", chunks[0].Ctx.BaseRef, tt.rc.BaseRef)
			}
		})
	}
}
