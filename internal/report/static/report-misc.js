}

function filterViolations() { rebuildViolationsTable(R.Findings || []); }

// ── Files ─────────────────────────────────────────────────────────────────────
function renderFiles() {
  rebuildFileList(R.Files || []);
}

function rebuildFileList(files) {
  const query = (document.getElementById('f-search')?.value || '').toLowerCase();
  const filtered = files.filter(f => !query || f.Path.toLowerCase().includes(query));
  const el = document.getElementById('file-list');

  if (filtered.length === 0) {
    renderHTML(el, '<div style="color:var(--muted);font-size:.85rem;padding:20px">No files match.</div>');
    return;
  }

  renderHTML(el, filtered.map((file, i) => {)
    const shortPath = file.Path.split('/').slice(-4).join('/');
    const heatColor = heatToColor(file.HeatScore);
    const count = file.Findings ? file.Findings.length : 0;
    const inner = count > 0 ? file.Findings.map(f =>
      `<div style="display:flex;gap:8px;align-items:flex-start;padding:4px 0;border-bottom:1px solid var(--border)">
        <span class="sev sev-${f.Severity}" style="margin-top:1px">${f.Severity[0].toUpperCase()}</span>
        <div><span class="rule-id">${f.Rule}</span>${f.Line ? ` <span style="color:var(--muted);font-size:.7rem">:${f.Line}</span>` : ''}<br><span style="font-size:.78rem;color:var(--muted)">${escHtml(f.Message)}</span></div>
      </div>`
    ).join('') : '<div style="color:var(--muted);font-size:.8rem">No violations.</div>';

    return `<div class="file-row" id="frow-${i}" onclick="toggleFile(${i})">
      <div class="file-row-header">
        <span class="file-path" title="${file.Path}">${shortPath}</span>
        <span class="file-heat" style="background:${heatColor}"></span>
        <span class="file-count">${count} issue${count !== 1 ? 's' : ''} · ${file.Lines} lines · ${file.Language}</span>
      </div>
      <div class="file-findings">${inner}</div>
    </div>`;
  }).join('');
}

function toggleFile(i) {
  document.getElementById('frow-' + i).classList.toggle('open');
}

function filterFiles() { rebuildFileList(R.Files || []); }

function heatToColor(score) {
  if (score === 0) return 'var(--surface2)';
  if (score < 0.33) return 'rgba(234,179,8,.6)';
  if (score < 0.66) return 'rgba(249,115,22,.7)';
  return 'rgba(239,68,68,.8)';
}

// ── Architecture ──────────────────────────────────────────────────────────────
function renderArchitecture() {
  const arch = R.Architecture;
  renderArchStatus(arch);
  renderArchOverview(arch);
  if (arch && arch.Flows && arch.Flows.length) {
    renderFlows(arch.Flows);
  }
  if (arch && arch.Packages && arch.Packages.length) {
    renderPackageMap(arch.Packages);
  } else {
    renderHTML(document.getElementById('arch-pkg-map'),
      '<div style="color:var(--muted);font-size:.85rem;padding:20px">No package data — scan a Go project.</div>');
  }
}

function renderArchOverview(arch) {
  const el = document.getElementById('arch-overview');
 renderHTML(el, ''; return; })
  // Build entry points list from flows
  const flows = arch.Flows || [];
  const entries = [...new Set(flows.map(f => f.Entry).filter(Boolean))];
  const pkgs = arch.Packages || [];
  // Layer distribution
  const layerCount = {};
  pkgs.forEach(p => { layerCount[p.Layer] = (layerCount[p.Layer] || 0) + 1; });
  const LAYER_ORDER = ['governance','orchestration','execution','persistence','surface'];
  const LAYER_COLORS = {'governance':'var(--info)','orchestration':'var(--advisory)','execution':'var(--pass)','persistence':'#a78bfa','surface':'#f472b6'};
  // Total file count from packages
  const fileSet = new Set();
  pkgs.forEach(p => { Object.keys(p.Files||{}).forEach(f => fileSet.add(f)); });
  const totalFiles = fileSet.size;
  // Entry link map: flow entry node ID → flow name
  const entryFlows = {};
  flows.forEach(f => {
    const e = f.Entry;
    if (!entryFlows[e]) entryFlows[e] = [];
    entryFlows[e].push(f.Name);
  });
  const entryLabels = entries.map(e => {
    const label = e.split(':').pop() || e.split('/').pop() || e;
    return { id: e, label, flows: entryFlows[e] || [] };
  });
  let html = '<div class="arch-overview-row">';
  // Entry points card
  html += `<div class="arch-ocard">
    <div class="olabel">Entry points</div>
    <div class="oval">${entries.length}</div>
    <div class="oitems">${entryLabels.length ? entryLabels.map(e => `<span class="role-badge role-entry" onclick="focusEntry('${escHtml(e.id)}')">${escHtml(e.label)}</span>`).join('') : '<span style="font-size:.72rem;color:var(--muted)">none</span>'}</div>
  </div>`;
  // Layer distribution
  html += `<div class="arch-ocard">
    <div class="olabel">Layers</div>
    <div class="oval">${LAYER_ORDER.filter(l => layerCount[l]).length}</div>
    <div class="oitems">${LAYER_ORDER.filter(l => layerCount[l]).map(l => `<span style="display:inline-flex;align-items:center;gap:3px;font-size:.65rem;padding:1px 6px;border-radius:8px;background:${LAYER_COLORS[l]}22;border:1px solid ${LAYER_COLORS[l]}55;color:${LAYER_COLORS[l]}">${l} ${layerCount[l]}</span>`).join('')}</div>
  </div>`;
  // Flows card
  html += `<div class="arch-ocard">
    <div class="olabel">Execution flows</div>
    <div class="oval">${flows.length}</div>
    <div class="oitems">${flows.slice(0,8).map(f => `<span class="role-badge role-core" onclick="focusFlow('${escHtml(f.Name)}')">${escHtml(f.Name)}</span>`).join('')}${flows.length > 8 ? `<span style="font-size:.65rem;color:var(--muted)">+${flows.length-8}</span>` : ''}</div>
  </div>`;
  // Files card
  html += `<div class="arch-ocard">
    <div class="olabel">Files (in graph)</div>
    <div class="oval">${totalFiles}</div>
    <div class="oitems" style="font-size:.65rem;color:var(--muted)">${pkgs.length} packages across ${Object.keys(layerCount).length} layers</div>
  </div>`;
  html += '</div>';
  renderHTML(el, html);
}

function toggleDocPanel(header) {
  const body = header.nextElementSibling;
  const chev = header.querySelector('.adoc-chev');
  body.classList.toggle('open');
  chev.textContent = body.classList.contains('open') ? '▲' : '▼';
}

// Focus the graph on a specific file/entry point by node ID (file path)
function focusEntry(entryId) {
  // entryId is like "file.go:main" — file part is before last colon
  const fileId = entryId.includes(':') ? entryId.slice(0, entryId.lastIndexOf(':')) : entryId;
  const tab = document.querySelector('.tab[onclick*="architecture"]');
  if (tab) tab.click();
  setTimeout(() => {
    const node = GV.map[fileId];
    if (node) { GV.showDetail(fileId); GV.draw(); }
  }, 200);
}

// Focus the graph on the entry file of a specific flow
function focusFlow(flowName) {
  const arch = R.Architecture;
  const flow = (arch.Flows || []).find(f => f.Name === flowName);
  if (!flow) return;
  focusEntry(flow.Entry);
}

function renderDocs() {
  const arch = R.Architecture;
  const docs = arch && arch.ArchDocFiles ? arch.ArchDocFiles : [];
  const el = document.getElementById('docs-list');
  const tab = document.getElementById('tab-docs');
  if (docs.length === 0) {
    if (tab) tab.style.display = 'none';
 renderHTML(el, '<div style="color:var(--muted);font-size:.85rem;padding:20px">No documentation files found.</div>');
    return;
  }
  if (tab) tab.style.display = '';
  renderHTML(el, docs.map((d, i) => {)
    const isPrimary = i === 0;
    const rendered = d.content ? markdownToHtml(d.content) : (d.html || '');
    return `<div class="adoc-panel" style="margin-bottom:8px">
      <div class="adoc-header" onclick="toggleDocPanel(this)">
        <span style="font-size:.85rem;font-weight:600">${escHtml(d.name || d.path)}</span>
