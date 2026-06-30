package coverage

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// parseGoCover reads a go coverage profile (go test -coverprofile=coverage.out).
// Format: <mode>\n<pkg>/<file>:<startLine>.<startCol>,<endLine>.<endCol> <stmts> <count>
func parseGoCover(path string) (map[string]fileCoverage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]fileCoverage)
	sc := bufio.NewScanner(f)

	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "mode:") || line == "" {
			continue
		}
		// <file>:<range> <stmts> <count>
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		// Extract file from "pkg/path/file.go:10.2,12.10"
		colPos := strings.LastIndex(parts[0], ":")
		if colPos < 0 {
			continue
		}
		file := parts[0][:colPos]

		stmts, err := strconv.Atoi(parts[1])
		if err != nil || stmts == 0 {
			continue
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		cov := result[file]
		cov.total += stmts
		if count > 0 {
			cov.hit += stmts
		}
		result[file] = cov
	}
	return result, sc.Err()
}
