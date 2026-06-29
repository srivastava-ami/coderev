package llm

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type ReviewChunk struct {
	File string
	Ctx  ReviewContext
}

func ChunkByFile(rc ReviewContext) []ReviewChunk {
	switch {
	case len(rc.Hunks) > 0:
		return chunkByHunkFiles(rc)
	case len(rc.Neighbors) > 0:
		return chunkByNeighborFiles(rc)
	default:
		return []ReviewChunk{{Ctx: rc}}
	}
}

func chunkByHunkFiles(rc ReviewContext) []ReviewChunk {
	files := uniqueFiles(rc.Hunks)
	var chunks []ReviewChunk
	for _, f := range files {
		chunk := ReviewChunk{
			File: f,
			Ctx: ReviewContext{
				BaseRef:   rc.BaseRef,
				Hunks:     filterHunks(rc.Hunks, f),
				Findings:  filterFindings(rc.Findings, f),
				Neighbors: filterNeighbors(rc.Neighbors, f),
			},
		}
		if len(chunk.Ctx.Hunks) > 0 || len(chunk.Ctx.Neighbors) > 0 || len(chunk.Ctx.Findings) > 0 {
			chunks = append(chunks, chunk)
		}
	}
	return chunks
}

func chunkByNeighborFiles(rc ReviewContext) []ReviewChunk {
	files := uniqueNeighborFiles(rc.Neighbors)
	var chunks []ReviewChunk
	for _, f := range files {
		chunk := ReviewChunk{
			File: f,
			Ctx: ReviewContext{
				BaseRef:   rc.BaseRef,
				Findings:  filterFindings(rc.Findings, f),
				Neighbors: filterNeighbors(rc.Neighbors, f),
			},
		}
		if len(chunk.Ctx.Neighbors) > 0 || len(chunk.Ctx.Findings) > 0 {
			chunks = append(chunks, chunk)
		}
	}
	return chunks
}

func uniqueFiles(hunks []DiffHunk) []string {
	seen := make(map[string]bool)
	var res []string
	for _, h := range hunks {
		if !seen[h.File] {
			seen[h.File] = true
			res = append(res, h.File)
		}
	}
	return res
}

func uniqueNeighborFiles(neighbors []GraphNeighbor) []string {
	seen := make(map[string]bool)
	var res []string
	for _, n := range neighbors {
		if !seen[n.File] {
			seen[n.File] = true
			res = append(res, n.File)
		}
	}
	return res
}

func filterHunks(hunks []DiffHunk, file string) []DiffHunk {
	var res []DiffHunk
	for _, h := range hunks {
		if h.File == file {
			res = append(res, h)
		}
	}
	return res
}

func filterFindings(findings []analysis.Finding, file string) []analysis.Finding {
	var res []analysis.Finding
	for _, f := range findings {
		if f.File == file || strings.HasSuffix(f.File, file) {
			res = append(res, f)
		}
	}
	return res
}

func filterNeighbors(neighbors []GraphNeighbor, file string) []GraphNeighbor {
	var res []GraphNeighbor
	for _, n := range neighbors {
		if n.File == file || strings.HasSuffix(n.File, file) {
			res = append(res, n)
		}
	}
	return res
}
