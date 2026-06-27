package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// jsonGraph is coderev's native code-graph JSON structure.
type jsonGraph struct {
	Directed bool       `json:"directed"`
	Nodes    []jsonNode `json:"nodes"`
	Links    []jsonLink `json:"links"`
}

type jsonNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	FileType   string `json:"file_type"`
	SourceFile string `json:"source_file"`
}

type jsonLink struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// ExportJSON writes coderev's native graph.json to dir.
func ExportJSON(g *Graph, dir string) error {
	gf := jsonGraph{Directed: true}
	for _, n := range g.Nodes {
		ft := string(n.Kind)
		gf.Nodes = append(gf.Nodes, jsonNode{
			ID:         n.ID,
			Label:      n.Label,
			FileType:   ft,
			SourceFile: n.SourceFile,
		})
	}
	for _, e := range g.Edges {
		gf.Links = append(gf.Links, jsonLink{
			Source:   e.Source,
			Target:   e.Target,
			Relation: e.Relation,
		})
	}

	data, err := json.MarshalIndent(gf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal graph.json: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(filepath.Join(dir, "graph.json"), data, 0o644)
}

// ExportGraphHTML writes a self-contained HTML page with a force-directed
// graph visualisation into dir/graph.html.
func ExportGraphHTML(g *Graph, dir string) error {
	// Build a minimal node/edge payload for the JS side.
	type jsNode struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Group string `json:"group"` // file / function / type
	}
	type jsEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Label  string `json:"label"`
	}
	var nodes []jsNode
	for _, n := range g.Nodes {
		nodes = append(nodes, jsNode{ID: n.ID, Label: n.Label, Group: string(n.Kind)})
	}
	var edges []jsEdge
	for _, e := range g.Edges {
		edges = append(edges, jsEdge{Source: e.Source, Target: e.Target, Label: e.Relation})
	}

	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Code Graph</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
html,body{width:100%%;height:100%%;overflow:hidden;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}
#graph{width:100%%;height:100%%}
.controls{position:absolute;top:12px;right:12px;background:#fff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,.15);padding:12px;font-size:13px;z-index:10}
.controls label{margin-right:8px;font-weight:600}
.controls select{padding:4px 8px;border:1px solid #ccc;border-radius:4px}
</style>
</head>
<body>
<div class="controls">
<label>Layout</label>
<select id="layoutSelect" onchange="applyLayout(this.value)">
<option value="force">Force</option>
<option value="radial">Radial</option>
</select>
</div>
<div id="graph"></div>
<script src="https://d3js.org/d3.v7.min.js"></script>
<script>
const nodes = %s;
const links = %s;

const width = window.innerWidth;
const height = window.innerHeight;

const color = d3.scaleOrdinal()
  .domain(["file","function","type"])
  .range(["#4A90D9","#7B68EE","#E6A817"]);

const svg = d3.select("#graph").append("svg")
  .attr("width", width)
  .attr("height", height);

let simulation, link, node, label;

function initForce() {
  svg.selectAll("*").remove();
  const g = svg.append("g");

  // Arrow marker
  svg.append("defs").selectAll("marker")
    .data(["end"])
    .join("marker")
    .attr("id","arrow")
    .attr("viewBox","0 -5 10 10")
    .attr("refX",20)
    .attr("refY",0)
    .attr("markerWidth",6)
    .attr("markerHeight",6)
    .attr("orient","auto")
    .append("path")
    .attr("d","M0,-5L10,0L0,5")
    .attr("fill","#999");

  link = g.append("g")
    .selectAll("line")
    .data(links)
    .join("line")
    .attr("stroke","#999")
    .attr("stroke-opacity",0.6)
    .attr("stroke-width",1.5)
    .attr("marker-end","url(#arrow)");

  node = g.append("g")
    .selectAll("circle")
    .data(nodes)
    .join("circle")
    .attr("r", d => d.group === "file" ? 8 : 5)
    .attr("fill", d => color(d.group))
    .attr("stroke","#fff")
    .attr("stroke-width",1.5)
    .call(drag(simulation));

  label = g.append("g")
    .selectAll("text")
    .data(nodes)
    .join("text")
    .text(d => d.label)
    .attr("font-size", d => d.group === "file" ? 11 : 9)
    .attr("dx", d => d.group === "file" ? 12 : 8)
    .attr("dy", 4)
    .attr("fill","#333");

  simulation = d3.forceSimulation(nodes)
    .force("link", d3.forceLink(links).id(d => d.id).distance(80))
    .force("charge", d3.forceManyBody().strength(-200))
    .force("center", d3.forceCenter(width/2, height/2))
    .force("collision", d3.forceCollide().radius(30))
    .on("tick", () => {
      link.attr("x1", d => d.source.x).attr("y1", d => d.source.y)
          .attr("x2", d => d.target.x).attr("y2", d => d.target.y);
      node.attr("cx", d => d.x).attr("cy", d => d.y);
      label.attr("x", d => d.x).attr("y", d => d.y);
    });
}

function initRadial() {
  svg.selectAll("*").remove();
  const g = svg.append("g").attr("transform","translate("+width/2+","+height/2+")");

  const root = d3.hierarchy({children: nodes}, d => {
    if (!d.children) {
      const kids = links.filter(l => l.source.id === d.id || l.source === d.id).map(l => {
        const t = nodes.find(n => n.id === (l.target.id || l.target));
        return t ? {id: t.id, label: t.label, group: t.group} : null;
      }).filter(Boolean);
      return kids.length ? kids : undefined;
    }
  });
  const layout = d3.cluster().size([2*Math.PI, Math.min(width,height)/2-50]);
  layout(root);

  link = g.append("g")
    .selectAll("line")
    .data(root.links())
    .join("line")
    .attr("stroke","#999")
    .attr("stroke-opacity",0.6)
    .attr("stroke-width",1.5);

  node = g.append("g")
    .selectAll("circle")
    .data(root.descendants())
    .join("circle")
    .attr("transform", d => "rotate("+(d.x*180/Math.PI-90)+") translate("+d.y+",0)")
    .attr("r", d => d.data.group === "file" ? 8 : 5)
    .attr("fill", d => color(d.data.group))
    .attr("stroke","#fff")
    .attr("stroke-width",1.5);
}

function applyLayout(name) {
  if (name === "radial") initRadial();
  else initForce();
}

function drag(sim) {
  function dragstarted(event,d) {
    if (!event.active) sim.alphaTarget(0.3).restart();
    d.fx = d.x; d.fy = d.y;
  }
  function dragged(event,d) {
    d.fx = event.x; d.fy = event.y;
  }
  function dragended(event,d) {
    if (!event.active) sim.alphaTarget(0);
    d.fx = null; d.fy = null;
  }
  return d3.drag().on("start",dragstarted).on("drag",dragged).on("end",dragended);
}

initForce();
</script>
</body>
</html>`, nodesJSON, edgesJSON)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(filepath.Join(dir, "graph.html"), []byte(html), 0o644)
}

// ExportAll writes both graph.json and graph.html into dir.
func ExportAll(g *Graph, dir string) error {
	if err := ExportJSON(g, dir); err != nil {
		return err
	}
	return ExportGraphHTML(g, dir)
}
