        <span style="font-size:.72rem;color:var(--muted)">${escHtml(d.path)}</span>
        <span class="adoc-chev" style="margin-left:auto">${isPrimary ? '▲' : '▼'}</span>
      </div>
      <div class="adoc-body arch-text ${isPrimary ? 'open' : ''}">${rendered}</div>
    </div>`;
  }).join(''));
}

function renderArchStatus(arch) {
  const el = document.getElementById('arch-status');
  if (arch.Source === 'doc') {
    renderHTML(el, `<div style="background:rgba(34,197,94,.08);border:1px solid rgba(34,197,94,.2);border-radius:6px;padding:8px 14px;font-size:.78rem;color:var(--pass)">✓ Architecture document: <code>${escHtml(arch.DocFile)}</code></div>`);
  } else {
    renderHTML(el, `<div style="background:rgba(234,179,8,.08);border:1px solid rgba(234,179,8,.2);border-radius:6px;padding:8px 14px;font-size:.78rem;color:var(--advisory)">⚠ No <code>docs/architecture.md</code> found — add one for a curated overview. Package map below is derived from source.</div>`);
  }
}

function flowClickStep(s) {
  const repoPath = (R.Meta.RepoPath || '').replace(/\/+$/, '');
  const absPath = s.File ? (s.File.startsWith('/') ? s.File : repoPath + '/' + s.File) : '';
  const node = GV.map[absPath];
  if (node) {
    const tab = document.querySelector('.tab[onclick*="architecture"]');
    if (tab) tab.click();
    setTimeout(() => { GV.showDetail(absPath); GV.draw(); }, 200);
  }
}

function renderFlows(flows) {
  const el = document.getElementById('arch-flows');
  const LAYER_COLORS = {governance:'var(--info)',orchestration:'var(--advisory)',execution:'var(--pass)',persistence:'#a78bfa',surface:'#f472b6'};
  let html = `<div style="font-size:.72rem;color:var(--muted);margin-bottom:6px">Use-case flows traced from code graph call edges — click a step to highlight in graph.</div>`;
  flows.forEach(f => {
    html += `<div style="border:1px solid var(--surface2);border-radius:6px;margin-bottom:8px;overflow:hidden">
      <div style="background:var(--surface2);padding:6px 12px;font-size:.8rem;font-weight:600">${escHtml(f.Name)}</div>
      <div style="padding:6px 12px;font-size:.72rem;color:var(--muted)">${escHtml(f.Description)}</div>
      <div style="padding:0 12px 8px 12px">`;
    var lastLayer = '';
    f.Steps.forEach((s,i) => {
      const color = LAYER_COLORS[s.Layer] || 'var(--muted)';
      const arrow = i > 0 ? `<span style="color:var(--muted);margin:0 4px;font-size:.7rem">→</span>` : '';
      const badge = s.Layer !== lastLayer ? `<span style="display:inline-block;background:${color};color:#fff;border-radius:3px;padding:0 5px;font-size:.63rem;font-weight:600;margin-right:4px">${s.Layer}</span>` : '';
      lastLayer = s.Layer;
      html += `${arrow}${badge}<span class="fstep" onclick="flowClickStep({File:'${escHtml(s.File)}',Func:'${escHtml(s.Func)}'})"><span class="fstep-fn">${escHtml(s.Func)}</span><span style="font-size:.62rem;color:var(--muted);margin-left:2px">${escHtml(s.File||'').split('/').pop()}</span></span>`;
    });
    html += `</div></div>`;
  });
  renderHTML(el, html);
}

const NXS_LAYERS = [
  {id:'governance',   label:'Governance',   role:'Config, standards, quality gates.'},
  {id:'orchestration',label:'Orchestration', role:'CLI entrypoints and run wiring.'},
  {id:'execution',    label:'Execution',     role:'Core domain logic — pure, no I/O.'},
  {id:'persistence',  label:'Persistence',   role:'Driven adapters: file I/O, external tools.'},
  {id:'surface',      label:'Surface',       role:'Output: reports, annotations, rendered artifacts.'},
];

function renderPackageMap(pkgs) {
  const container = document.getElementById('arch-pkg-map');
  const byLayer = {};
  pkgs.forEach(p => { (byLayer[p.Layer] = byLayer[p.Layer] || []).push(p); });
  let html = `<div style="font-size:.72rem;color:var(--muted);margin-bottom:10px">
    Architecture manifest generated at <code>.coderev/architecture.toml</code> — schema: nexovia/2 · NXS pattern.
    Click a package to see dependencies. Edit the TOML to add context.
  </div>`;
  NXS_LAYERS.forEach(l => {
    const items = byLayer[l.id];
    if (!items || !items.length) return;
    html += buildLayerSection(l, items);
  });
  // catch-all for any layer not in NXS_LAYERS
  const known = new Set(NXS_LAYERS.map(l=>l.id));
  Object.keys(byLayer).filter(k=>!known.has(k)).forEach(k => {
    html += buildLayerSection({id:k, label:k, role:''}, byLayer[k]);
  });
  renderHTML(container, html || '<div style="color:var(--muted);font-size:.85rem;padding:20px">No packages found.</div>');
}

function buildLayerSection(l, items) {
  const roleHtml = l.role ? `<span style="font-size:.67rem;color:var(--muted);margin-left:8px;font-weight:400">${escHtml(l.role)}</span>` : '';
  const cards = items.map(p => buildPkgCard(p)).join('');
  return `<div class="pkg-layer" style="margin-bottom:8px">
    <div class="pkg-layer-hdr" onclick="toggleLayer(this)">
      <span class="pkg-layer-title">${escHtml(l.label)}</span>${roleHtml}
      <span class="pkg-layer-count">${items.length} pkg${items.length > 1 ? 's' : ''}</span>
      <span class="pkg-layer-chev">▼</span>
    </div>
    <div class="pkg-grid">${cards}</div>
  </div>`;
}

function buildPkgCard(p) {
  const deps = p.Deps || [];
  const syms = p.ExportedSymbols || [];
  const files = p.Files || {};
  const fileKeys = Object.keys(files).sort();
  let fileHtml = '';
  if (fileKeys.length) {
    fileHtml = fileKeys.map(f => {
      const syms = files[f] || [];
      if (!syms.length) return '';
      return `<div style="margin-top:4px;font-size:.68rem;padding:2px 0"><span style="color:var(--muted)">${escHtml(f.split('/').pop())}</span> ${syms.slice(0,8).map(s=>`<span class="pkg-dep-tag" style="color:var(--info)">${escHtml(s)}</span>`).join('')}${syms.length>8?`<span style="color:var(--muted);font-size:.65rem"> +${syms.length-8}</span>`:''}</div>`;
    }).join('');
  }
  const depHtml = deps.length
    ? `<div class="pkg-dep-panel">
        <div class="pkg-dep-lbl">Depends on (${deps.length})</div>
        ${deps.map(d=>`<span class="pkg-dep-tag">${escHtml(d.split('/').pop())}</span>`).join('')}
        ${syms.length ? `<div class="pkg-dep-lbl" style="margin-top:6px">Exports (${syms.length})</div>${fileHtml}` : ''}
       </div>`
    : (syms.length ? `<div class="pkg-dep-panel"><div class="pkg-dep-lbl">Exports (${syms.length})</div>${fileHtml}</div>` : '');
  const docHtml = p.DocSummary ? `<div class="pkg-doc">${escHtml(p.DocSummary)}</div>` : '';
  return `<div class="pkg-card" onclick="this.classList.toggle('open')">
    <div class="pkg-name">${escHtml(p.Name)}</div>
    <div class="pkg-path">${escHtml(p.ImportPath)}</div>
    ${docHtml}${depHtml}
  </div>`;
}

function toggleLayer(header) {
  const grid = header.nextElementSibling;
  const chev = header.querySelector('.pkg-layer-chev');
  grid.classList.toggle('closed');
  chev.textContent = grid.classList.contains('closed') ? '▶' : '▼';
}

/* ====================== INTERACTIVE ARCHITECTURE GRAPH ====================== */
const GV = {
  mode:'topology', expand:new Set(), sel:null, flowHighlight:null,
  t:{x:0,y:0,k:1}, nodes:[], links:[], map:{}, layerY:{},
  W:0, H:480, graphData:null, pkgMap:{},
  LAYER_COLORS:{governance:'var(--info)',orchestration:'var(--advisory)',execution:'var(--pass)',persistence:'#a78bfa',surface:'#f472b6'},
  LAYER_ORDER:['governance','orchestration','execution','persistence','surface'],
  init(){
    if(!R.GraphJSON) return;
    try { GV.graphData = JSON.parse(R.GraphJSON); } catch(e){ return; }
    const pm = {};
    const repoPath = (R.Meta.RepoPath || '').replace(/\/+$/, '');
    (R.Architecture.Packages||[]).forEach(p => {
      Object.keys(p.Files||{}).forEach(f => {
        const abs = repoPath + '/' + f;
        pm[abs] = {layer:p.Layer, pkg:p.Name, doc:p.DocSummary, file:abs, funcs:p.Files[f]||[]};
        pm[f] = pm[abs];
        const short = f.split('/').pop();
        pm[short] = pm[abs];
      });
    });
    GV.pkgMap = pm;
    GV.flowFiles = {};
    (R.Architecture.Flows || []).forEach(f => {
      (f.Steps || []).forEach(s => {
        const absPath = s.File ? (s.File.startsWith('/') ? s.File : repoPath + '/' + s.File) : '';
        if (!GV.flowFiles[absPath]) GV.flowFiles[absPath] = [];
        if (!GV.flowFiles[absPath].includes(f.Name)) GV.flowFiles[absPath].push(f.Name);
      });
    });
    GV.W = document.getElementById('graph-host').clientWidth;
    GV.build();
    GV.fit();
    const firstFile = GV.nodes.find(n => n.type === 'file');
    if (firstFile) GV.showDetail(firstFile.id);
    GV.draw();
    GV.setupUI();
  },
  getLayer(file){
    const m = GV.pkgMap[file] || GV.pkgMap[file?.replace(/.*\//,'')];
    return m ? m.layer : null;
  },
  getPkgInfo(file){
    return GV.pkgMap[file] || GV.pkgMap[file?.replace(/.*\//,'')] || null;
  },
  build(){
    const ns=[], ls=[], map={};
    const add=n=>{ ns.push(n); map[n.id]=n; return n; };
    const g=GV.graphData;
    if(!g) return;
    if(GV.mode==='hierarchy'){
      const layers = {};
      g.nodes.forEach(n => { if(n.kind!=='file') return; const layer = GV.getLayer(n.source_file) || 'other'; (layers[layer] = layers[layer] || []).push(n); });
      const layerIds = Object.keys(layers).sort();
      if(!layerIds.length){ GV.nodes=[]; GV.links=[]; GV.map={}; return; }
      const root=add({id:'__root',type:'root',label:R.Meta.RepoName,order:0});
      layerIds.forEach((l,i) => {
        const ln=add({id:'__L_'+l,type:'layer',layer:l,label:l,order:i+1});
        ls.push({s:'__root',t:ln.id,kind:'tree'});
        layers[l].forEach(fn => {
          const n=add({id:fn.id,type:'file',label:fn.label||fn.id,source_file:fn.source_file,layer:l});
          ls.push({s:ln.id,t:n.id,kind:'tree'});
          if(GV.expand.has(n.id)){
            const info = GV.getPkgInfo(fn.source_file);
            const fns = info ? info.funcs : [];
            fns.forEach(fname => { const fid=n.id+'#'+fname; add({id:fid,type:'fn',label:fname,parentId:n.id,layer:l}); ls.push({s:n.id,t:fid,kind:'fn'}); });
          }
        });
      });
    } else {
      const fileNodes = {};
      g.nodes.forEach(n => {
        if(n.kind==='file'){ const layer=GV.getLayer(n.source_file)||'other'; fileNodes[n.id]=1; add({id:n.id,type:'file',label:n.label||n.id,source_file:n.source_file,layer:layer}); }
      });
      g.links.forEach(lk => {
        if(!fileNodes[lk.source] || !fileNodes[lk.target]) return;
        if(lk.relation==='contains' || lk.relation==='imports' || lk.relation==='calls') ls.push({s:lk.source,t:lk.target,kind:(lk.relation==='calls'?'calls':'import'),relation:lk.relation});
      });
      Object.keys(fileNodes).forEach(fid => {
        if(GV.expand.has(fid)){
          const src = g.nodes.find(n=>n.id===fid)?.source_file||'';
          const info = GV.getPkgInfo(src);
          const fns = info ? info.funcs : [];
          fns.forEach(fname => { const fnid=fid+'#'+fname; add({id:fnid,type:'fn',label:fname,parentId:fid,layer:GV.map[fid]?.layer||'other'}); ls.push({s:fid,t:fnid,kind:'fn'}); });
        }
      });
    }
    GV.nodes=ns; GV.links=ls; GV.map=map;
    if(GV.mode==='hierarchy'){ GV.layoutTree(); } else { GV.layoutLayered(); }
  },
  layoutTree(){
    let leaf=0; const vGap=110, hGap=180;
    const place=(id,depth)=>{ const n=GV.map[id]; if(!n)return; n.depth=depth; const kids=GV.links.filter(l=>l.s===id).map(l=>l.t); if(kids.length){ kids.forEach(k=>place(k,depth+1)); n.x=(GV.map[kids[0]]?.x||0)+(kids.length>1?((GV.map[kids[kids.length-1]]?.x||0)-(GV.map[kids[0]]?.x||0))/2:0); } else { n.x=leaf*hGap; leaf++; } n.y=depth*vGap+40; };
    place('__root',0);
  },
  layoutLayered(){
    // Deterministic layout: fixed Y per layer, X ordered by call influence
    const fileNodes = GV.nodes.filter(n => n.type==='file');
