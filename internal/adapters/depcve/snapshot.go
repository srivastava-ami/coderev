package depcve

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	cacheDir  = ".config/coderev/vulndb"
	cacheFile = "osv-snapshot.json.gz"
	shipDir   = "data"
)

type Vuln struct {
	ID       string   `json:"id"`
	Summary  string   `json:"summary"`
	Aliases  []string `json:"aliases"`
	Affected []struct {
		Package struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		} `json:"package"`
		Ranges []struct {
			Type   string `json:"type"`
			Events []struct {
				Introduced string `json:"introduced,omitempty"`
				Fixed      string `json:"fixed,omitempty"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
}

func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, cacheDir, cacheFile), nil
}

func shipPath(target string) string {
	return filepath.Join(target, shipDir, cacheFile)
}

func readGzippedJSON(path string) ([]Vuln, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	var vulns []Vuln
	if err := json.NewDecoder(gr).Decode(&vulns); err != nil {
		return nil, err
	}
	return vulns, nil
}

func writeGzippedJSON(path string, vulns []Vuln) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	return json.NewEncoder(gw).Encode(vulns)
}

func fetchSnapshot(url string) ([]Vuln, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching snapshot: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching snapshot: HTTP %d", resp.StatusCode)
	}
	var reader io.ReadCloser = resp.Body
	if ct := resp.Header.Get("Content-Type"); ct != "" && ct != "application/gzip" && ct != "application/x-gzip" {
		reader = resp.Body
	} else {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gunzip snapshot: %w", err)
		}
		defer gr.Close()
		reader = gr
	}
	var vulns []Vuln
	if err := json.NewDecoder(reader).Decode(&vulns); err != nil {
		return nil, fmt.Errorf("decoding snapshot: %w", err)
	}
	cache, err := cachePath()
	if err == nil {
		writeGzippedJSON(cache, vulns)
	}
	return vulns, nil
}

func loadSnapshot(target, snapshotURL string) []Vuln {
	cache, err := cachePath()
	if err == nil {
		if v, err := readGzippedJSON(cache); err == nil && len(v) > 0 {
			return v
		}
	}
	shipped := shipPath(target)
	if v, err := readGzippedJSON(shipped); err == nil && len(v) > 0 {
		return v
	}
	if snapshotURL != "" {
		if v, err := fetchSnapshot(snapshotURL); err == nil {
			return v
		}
	}
	return nil
}
