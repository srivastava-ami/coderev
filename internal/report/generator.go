package report

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
)

//go:embed template.html
var htmlTemplate string

// Generate writes the self-contained HTML report to outputPath.
func Generate(r Report, outputPath string) error {
	reportJSON, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshalling report: %w", err)
	}

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	data := struct {
		ReportJSON template.JS
		Meta       Meta
	}{
		ReportJSON: template.JS(reportJSON),
		Meta:       r.Meta,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	return nil
}
