package coverage

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type fileCoverage struct {
	hit   int
	total int
}

// parseLcov reads an lcov.info file and returns per-source-file coverage.
// Format reference: https://ltp.sourceforge.net/coverage/lcov/geninfo.1.php
// DA:<line>,<hit_count>  — one entry per executable line
// SF:<source_file>       — starts a record
// end_of_record          — ends a record
func parseLcov(path string) (map[string]fileCoverage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]fileCoverage)
	var current string

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "SF:"):
			current = strings.TrimPrefix(line, "SF:")
		case strings.HasPrefix(line, "DA:") && current != "":
			parts := strings.SplitN(strings.TrimPrefix(line, "DA:"), ",", 2)
			if len(parts) != 2 {
				continue
			}
			cov := result[current]
			cov.total++
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				cov.hit++
			}
			result[current] = cov
		case line == "end_of_record":
			current = ""
		}
	}
	return result, sc.Err()
}
