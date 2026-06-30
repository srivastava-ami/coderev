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

//go:embed static/report.css
var cssContent string

//go:embed static/report-core.js
var jsCore string

//go:embed static/report-misc.js
var jsMisc string

//go:embed static/report-arch.js
var jsArch string

//go:embed static/report-graph.js
var jsGraph string

//go:embed static/report-viz.js
var jsViz string

//go:embed static/report-utils.js
var jsUtils string

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

	jsContent := jsCore + "\n" + jsMisc + "\n" + jsArch + "\n" + jsGraph + "\n" + jsViz + "\n" + jsUtils
	data := struct {
		ReportJSON template.JS
		Meta       Meta
		CSS        template.CSS
		JS         template.JS
	}{
		ReportJSON: template.JS(reportJSON),
		Meta:       r.Meta,
		CSS:        template.CSS(cssContent),
		JS:         template.JS(jsContent),
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	return nil
}
