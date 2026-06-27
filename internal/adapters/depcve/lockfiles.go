package depcve

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Dependency struct {
	Name      string
	Version   string
	Ecosystem string
	File      string
}

func parsePackageLockJSON(path string) ([]Dependency, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var doc struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return nil, err
	}
	var deps []Dependency
	for name, pkg := range doc.Dependencies {
		deps = append(deps, Dependency{
			Name:      name,
			Version:   pkg.Version,
			Ecosystem: "npm",
			File:      path,
		})
	}
	return deps, nil
}

func parseGoSum(path string) ([]Dependency, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var deps []Dependency
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		module := parts[0]
		ver := parts[1]
		ver = strings.TrimPrefix(ver, "v")
		deps = append(deps, Dependency{
			Name:      module,
			Version:   ver,
			Ecosystem: "Go",
			File:      path,
		})
	}
	return deps, sc.Err()
}

func parseRequirementsTXT(path string) ([]Dependency, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var deps []Dependency
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if idx := strings.Index(line, "=="); idx > 0 {
			name := strings.TrimSpace(line[:idx])
			ver := strings.TrimSpace(line[idx+2:])
			deps = append(deps, Dependency{
				Name:      name,
				Version:   ver,
				Ecosystem: "PyPI",
				File:      path,
			})
		}
	}
	return deps, sc.Err()
}

func parseLockfiles(target string) ([]Dependency, error) {
	var deps []Dependency
	walkFn := func(path string, d os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "node_modules" || base == ".git" || base == ".cache" || base == "vendor" || base == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		base := d.Name()
		var parsed []Dependency
		switch base {
		case "package-lock.json":
			parsed, err = parsePackageLockJSON(path)
		case "go.sum":
			parsed, err = parseGoSum(path)
		case "requirements.txt":
			parsed, err = parseRequirementsTXT(path)
		}
		if err == nil {
			deps = append(deps, parsed...)
		}
		return nil
	}
	filepath.Walk(target, walkFn)
	return deps, nil
}

func matchVuln(dep Dependency, vuln Vuln) []string {
	var matched bool
	for _, affected := range vuln.Affected {
		if affected.Package.Name != dep.Name {
			continue
		}
		for _, r := range affected.Ranges {
			if r.Type != "ECOSYSTEM" {
				continue
			}
			if versionInRanges(dep.Version, r.Events) {
				matched = true
			}
		}
	}
	if !matched {
		return nil
	}
	ids := []string{vuln.ID}
	for _, a := range vuln.Aliases {
		ids = append(ids, a)
	}
	return ids
}

func versionInRanges(version string, events []struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}) bool {
	vuln := false
	for _, e := range events {
		if e.Introduced != "" {
			if compareVersions(version, e.Introduced) >= 0 {
				vuln = true
			}
		}
		if e.Fixed != "" {
			if compareVersions(version, e.Fixed) >= 0 {
				vuln = false
			}
		}
	}
	return vuln
}

func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	maxLen := len(pa)
	if len(pb) > maxLen {
		maxLen = len(pb)
	}
	for i := 0; i < maxLen; i++ {
		var na, nb int
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d", &na)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d", &nb)
		}
		if na < nb {
			return -1
		}
		if na > nb {
			return 1
		}
	}
	return 0
}
