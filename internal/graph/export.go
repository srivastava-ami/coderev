package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// sortedGraph returns copies of the graph's nodes and edges in a canonical order
// (nodes by ID; edges by source, then target, then relation). Exports use this so
// output is byte-for-byte deterministic regardless of the builder's internal map
// iteration order.
func sortedGraph(g *Graph) ([]Node, []Edge) {
	ns := make([]Node, len(g.Nodes))
	copy(ns, g.Nodes)
	sort.Slice(ns, func(i, j int) bool { return ns[i].ID < ns[j].ID })
	es := make([]Edge, len(g.Edges))
	copy(es, g.Edges)
	sort.Slice(es, func(i, j int) bool {
		if es[i].Source != es[j].Source {
			return es[i].Source < es[j].Source
		}
		if es[i].Target != es[j].Target {
			return es[i].Target < es[j].Target
		}
		return es[i].Relation < es[j].Relation
	})
	return ns, es
}

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
	snodes, sedges := sortedGraph(g)
	gf := jsonGraph{Directed: true}
	for _, n := range snodes {
		ft := string(n.Kind)
		gf.Nodes = append(gf.Nodes, jsonNode{
			ID:         n.ID,
			Label:      n.Label,
			FileType:   ft,
			SourceFile: n.SourceFile,
		})
	}
	for _, e := range sedges {
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

// ExportGraphHTML writes a fully self-contained HTML page into dir/graph.html.
// It has NO external dependencies — no CDN, no third-party library, no web fonts,
// no network at all. Node positions are computed deterministically in Go
// (ComputeLayout) and embedded; the page is a static SVG rendered by a few lines
// of vanilla JS that add pan, zoom and node drag. The same graph always produces
// byte-identical HTML.
func ExportGraphHTML(g *Graph, dir string) error {
	pos := ComputeLayout(g)
	snodes, sedges := sortedGraph(g)

	type jsNode struct {
		ID    string  `json:"id"`
		Label string  `json:"label"`
		Group string  `json:"group"` // file / function / type
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
	}
	type jsEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	nodes := make([]jsNode, 0, len(snodes))
	for _, n := range snodes {
		p := pos[n.ID]
		nodes = append(nodes, jsNode{ID: n.ID, Label: n.Label, Group: string(n.Kind), X: p.X, Y: p.Y})
	}
	edges := make([]jsEdge, 0, len(sedges))
	for _, e := range sedges {
		edges = append(edges, jsEdge{Source: e.Source, Target: e.Target})
	}

	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>coderev · code graph</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
html,body{width:100%%;height:100%%;overflow:hidden;background:#0a0a0f;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}
#svg{width:100%%;height:100%%;cursor:grab}
#svg.panning{cursor:grabbing}
.legend{position:absolute;top:12px;left:12px;background:rgba(255,255,255,.95);border-radius:8px;padding:10px 12px;font-size:12px;line-height:1.7;box-shadow:0 2px 8px rgba(0,0,0,.3)}
.legend i{display:inline-block;width:10px;height:10px;border-radius:50%%;margin-right:6px;vertical-align:middle}
.hint{position:absolute;bottom:10px;left:12px;color:#888;font-size:11px}
text{pointer-events:none;fill:#cfcfd6;font-size:9px}
line{stroke:#555;stroke-opacity:.6;stroke-width:1}
circle{stroke:#0a0a0f;stroke-width:1.5;cursor:pointer}
</style>
</head>
<body>
<div class="legend">
<div><i style="background:#4A90D9"></i>file</div>
<div><i style="background:#7B68EE"></i>function</div>
<div><i style="background:#E6A817"></i>type</div>
</div>
<div class="hint">scroll to zoom · drag background to pan · drag a node to move it</div>
<svg id="svg" viewBox="0 0 %.0f %.0f" preserveAspectRatio="xMidYMid meet">
<defs>
<marker id="arrow" viewBox="0 -5 10 10" refX="16" refY="0" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,-5L10,0L0,5" fill="#555"></path></marker>
</defs>
<g id="view"></g>
</svg>
<script>
const NODES = %s;
const LINKS = %s;
const COLOR = {file:"#4A90D9",function:"#7B68EE",type:"#E6A817"};
const svg = document.getElementById("svg");
const view = document.getElementById("view");
const NS = svg.namespaceURI;
const byId = {};
NODES.forEach(n => { byId[n.id] = n; });

const lineEls = [];
LINKS.forEach(l => {
  const s = byId[l.source], t = byId[l.target];
  if (!s || !t) return;
  const ln = document.createElementNS(NS, "line");
  ln.setAttribute("marker-end", "url(#arrow)");
  ln.s = s; ln.t = t;
  view.appendChild(ln);
  lineEls.push(ln);
});

NODES.forEach(n => {
  const c = document.createElementNS(NS, "circle");
  c.setAttribute("r", n.group === "file" ? 7 : 4.5);
  c.setAttribute("fill", COLOR[n.group] || "#aaa");
  c.node = n;
  view.appendChild(c);
  n.c = c;
  const tx = document.createElementNS(NS, "text");
  tx.textContent = n.label;
  tx.setAttribute("dx", n.group === "file" ? 9 : 6);
  tx.setAttribute("dy", 3);
  view.appendChild(tx);
  n.tx = tx;
});

function render() {
  for (const ln of lineEls) {
    ln.setAttribute("x1", ln.s.x); ln.setAttribute("y1", ln.s.y);
    ln.setAttribute("x2", ln.t.x); ln.setAttribute("y2", ln.t.y);
  }
  for (const n of NODES) {
    n.c.setAttribute("cx", n.x); n.c.setAttribute("cy", n.y);
    n.tx.setAttribute("x", n.x); n.tx.setAttribute("y", n.y);
  }
}
render();

let vb = {x:0, y:0, w:%.0f, h:%.0f};
function applyVB() { svg.setAttribute("viewBox", vb.x+" "+vb.y+" "+vb.w+" "+vb.h); }
function toSvg(e) {
  const r = svg.getBoundingClientRect();
  return { x: vb.x + (e.clientX - r.left)/r.width*vb.w,
           y: vb.y + (e.clientY - r.top)/r.height*vb.h };
}
svg.addEventListener("wheel", e => {
  e.preventDefault();
  const scale = e.deltaY < 0 ? 0.9 : 1.1;
  const p = toSvg(e);
  vb.x = p.x - (p.x - vb.x)*scale;
  vb.y = p.y - (p.y - vb.y)*scale;
  vb.w *= scale; vb.h *= scale;
  applyVB();
}, {passive:false});

let drag = null;
svg.addEventListener("mousedown", e => {
  if (e.target.tagName === "circle") { drag = {node: e.target.node}; }
  else { drag = {pan:true, lx:e.clientX, ly:e.clientY}; svg.classList.add("panning"); }
});
window.addEventListener("mousemove", e => {
  if (!drag) return;
  if (drag.node) {
    const p = toSvg(e);
    drag.node.x = +p.x.toFixed(1); drag.node.y = +p.y.toFixed(1);
    render();
  } else {
    const r = svg.getBoundingClientRect();
    vb.x -= (e.clientX - drag.lx)/r.width*vb.w;
    vb.y -= (e.clientY - drag.ly)/r.height*vb.h;
    drag.lx = e.clientX; drag.ly = e.clientY;
    applyVB();
  }
});
window.addEventListener("mouseup", () => { drag = null; svg.classList.remove("panning"); });
</script>
</body>
</html>`, layoutWidth, layoutHeight, nodesJSON, edgesJSON, layoutWidth, layoutHeight)

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
