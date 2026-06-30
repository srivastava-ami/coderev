    if(!fileNodes.length) return;
    // Group by layer
    const layerMap = {};
    fileNodes.forEach(n => {
      const l = n.layer || 'other';
      if(!layerMap[l]) layerMap[l] = [];
      layerMap[l].push(n);
    });
    // Compute Y positions (evenly spaced bands)
    const layerKeys = Object.keys(layerMap).sort((a,b) => GV.LAYER_ORDER.indexOf(a) - GV.LAYER_ORDER.indexOf(b));
    GV.layerY = {};
    const padTop = 45, padBottom = 25;
    const availH = GV.H - padTop - padBottom;
    const bandH = Math.min(90, availH / Math.max(layerKeys.length, 1));
    layerKeys.forEach((l,i) => { GV.layerY[l] = padTop + i * bandH + bandH/2; });
    // Assign Y, compute influence score for X ordering
    const influence = {};
    fileNodes.forEach(n => { n.y = GV.layerY[n.layer] || (padTop + 60); n._score = 0; });
    GV.links.forEach(lk => {
      if(lk.kind==='fn'||lk.kind==='tree') return;
      if(!influence[lk.s]) influence[lk.s] = {in:0, out:0};
      if(!influence[lk.t]) influence[lk.t] = {in:0, out:0};
      influence[lk.s].out++;
      influence[lk.t].in++;
    });
    fileNodes.forEach(n => {
      const inf = influence[n.id] || {in:0, out:0};
      n._score = inf.in - inf.out;  // higher score = more called (provider) → left
    });
    // Sort within each layer by influence score, then assign X
    const padX = 30;
    Object.keys(layerMap).forEach(l => {
      const nodes = layerMap[l].sort((a,b) => b._score - a._score);
      const count = nodes.length;
      const gap = Math.min(280, Math.max(150, (GV.W - padX*2) / Math.max(count, 1)));
      const totalW = (count-1)*gap;
      const startX = (GV.W - totalW)/2;
      nodes.forEach((n,i) => { n.x = startX + i*gap; });
    });
    // Position function nodes below their parents
    GV.nodes.filter(n=>n.type==='fn').forEach(n => {
      const p = GV.map[n.parentId];
      if(p){ n.x = p.x; n.y = p.y + 38; }
    });
  },
  fit(){
    const ns=GV.nodes.filter(n=>n.type!=='fn');
    if(!ns.length) return;
    const xs=ns.map(n=>n.x),ys=ns.map(n=>n.y);
    const minx=Math.min(...xs)-60,maxx=Math.max(...xs)+60,miny=Math.min(...ys)-30,maxy=Math.max(...ys)+30;
    const host=document.getElementById('graph-host'),w=host.clientWidth,h=host.clientHeight;
    const k=Math.min(w/(maxx-minx),h/(maxy-miny),1.3);
    GV.t.k=k; GV.t.x=(w-(minx+maxx)*k)/2; GV.t.y=(h-(miny+maxy)*k)/2;
    GV.draw();
  },
  draw(){
    const v=document.getElementById('gview');
    if(!v) return;
    v.setAttribute('transform',`translate(${GV.t.x},${GV.t.y}) scale(${GV.t.k})`);
    const sel=GV.sel, conn=new Set(), fh=GV.flowHighlight;
    if(sel){ conn.add(sel); GV.links.forEach(l=>{ if(l.s===sel)conn.add(l.t); if(l.t===sel)conn.add(l.s); }); }
    let h='';
    // Layer background bands
    const layerKeys = Object.keys(GV.layerY);
    layerKeys.forEach(l => {
      const y = GV.layerY[l], clr = GV.LAYER_COLORS[l] || 'var(--muted)';
      h+=`<g class="lay-band"><rect x="-2000" y="${y-36}" width="4000" height="72" fill="${clr}08" stroke="${clr}15" stroke-width="1"/>
        <text class="lbl" x="12" y="${y-12}">${l}</text></g>`;
    });
    // Function expanded bands
    const fileFns = {};
    GV.nodes.filter(n=>n.type==='fn').forEach(n => { fileFns[n.parentId] = (fileFns[n.parentId]||0) + 1; });
    Object.keys(fileFns).forEach(fid => {
      const fn = GV.map[fid]; if(!fn) return;
      const fns = GV.nodes.filter(n => n.parentId===fid);
      if(!fns.length) return;
      const maxY = Math.max(...fns.map(n => n.y));
      h+=`<rect class="fn-band" x="${fn.x-90}" y="${fn.y+22}" width="180" height="${Math.max(20, maxY-fn.y+22)}" rx="4"/>`;
    });
    // Edges with orthogonal routing and arrowheads
    GV.links.forEach(link => { h += GV.renderEdge(link, sel, fh); });
    // Nodes
    GV.nodes.forEach(node => { h += GV.renderNode(node, sel, conn, fh); });
    renderHTML(v, h);
  },
  renderEdge(link, selectedNode, flowHighlight){
    const from=GV.map[link.s], to=GV.map[link.t];
    if(!from||!to) return '';
    let classes = 'lnk ' + link.kind;
    const isHot = selectedNode && (link.s===selectedNode || link.t===selectedNode);
    const hasFlow = flowHighlight && flowHighlight.nodes.has(link.s) && flowHighlight.nodes.has(link.t);
    const hasFlowLight = flowHighlight && !hasFlow && (flowHighlight.nodes.has(link.s) || flowHighlight.nodes.has(link.t));
    if(isHot) classes += ' hot';
    if(hasFlow) classes += ' flow';
    if(hasFlowLight) classes += ' flow-light';
    if(flowHighlight && !isHot && !hasFlow && !hasFlowLight) classes += ' flow-dim';
    let marker = 'marker-end="url(#arrow-';
    if(hasFlow) marker += 'flow';
    else if(isHot) marker += 'calls-hot';
    else if(link.kind==='import'||link.kind==='tree') marker += 'import';
    else marker += 'calls';
    marker += ')"';
    let path;
    if(GV.mode==='hierarchy'){
      path = `M${from.x} ${from.y+14} C${from.x} ${(from.y+to.y)/2},${to.x} ${(from.y+to.y)/2},${to.x} ${to.y-14}`;
    } else if(from.y === to.y){
      path = `M${from.x} ${from.y} L${to.x} ${to.y}`;
    } else {
      const midX = (from.x + to.x) / 2;
      const endX = to.x > from.x ? to.x - 4 : to.x + 4;
      path = `M${from.x} ${from.y} L${midX} ${from.y} L${midX} ${to.y} L${endX} ${to.y}`;
    }
    return `<path class="${classes}" d="${path}" ${marker}/>`;
  },
  renderNode(node, selectedNode, connectedSet, flowHighlight){
    const color = GV.nodeColor(node);
    const isDim = (selectedNode && !connectedSet.has(node.id)) || (flowHighlight && !flowHighlight.nodes.has(node.id) && !selectedNode) ? ' dim' : '';
    const isSelected = node.id === selectedNode ? ' sel' : '';
    const hasFlowGlow = flowHighlight && flowHighlight.nodes.has(node.id) && !selectedNode;
    if(node.type==='fn') return GV.renderFnNode(node, isDim, isSelected, color);
    if(node.type==='layer') return GV.renderLayerNode(node, isDim, color);
    if(node.type==='root') return GV.renderRootNode(node, isDim, isSelected);
    return GV.renderFileNode(node, isDim, isSelected, color, hasFlowGlow);
  },
  renderFnNode(node, dimClass, selClass, color){
    const w = Math.max(30, node.label.length*5.5+12);
    const parentColor = GV.nodeColor(GV.map[node.parentId])||'var(--muted)';
    return `<g class="gn fn${dimClass}${selClass}" data-id="${node.id}" data-parent="${node.parentId}" transform="translate(${node.x},${node.y})"><rect x="${-w/2}" y="-9" width="${w}" height="18" rx="9" fill="${parentColor}18" stroke="${parentColor}77"/><text class="t" x="0" y="3" text-anchor="middle">${escHtml(node.label)}</text></g>`;
  },
  renderLayerNode(node, dimClass, color){
    const w = Math.max(70, node.label.length*7+20);
    return `<g class="gn layer${dimClass}" data-id="${node.id}" transform="translate(${node.x},${node.y})"><rect x="${-w/2}" y="-12" width="${w}" height="24" rx="12" fill="${color}22" stroke="${color}"/><text class="t" x="0" y="4" text-anchor="middle">${escHtml(node.label)}</text></g>`;
  },
  renderRootNode(node, dimClass, selClass){
    const w = Math.max(60, node.label.length*8+20);
    return `<g class="gn root${dimClass}${selClass}" data-id="${node.id}" transform="translate(${node.x},${node.y})"><rect x="${-w/2}" y="-14" width="${w}" height="28" rx="8" fill="#1d2740" stroke="${cssV('--info')}"/><text class="t" x="0" y="5" text-anchor="middle">${escHtml(node.label)}</text></g>`;
  },
  renderFileNode(node, dimClass, selClass, color, flowGlow){
    const w = 150, hh = 40;
    const isEp = GV.isEntrypoint(node);
    const glowAttr = flowGlow ? ' stroke-width="3" stroke="var(--warn)"' : '';
    const epPolygon = isEp ? `<polygon class="ep-star ep-glow" points="${w/2-12},${-hh/2+5} ${w/2-8},${-hh/2+13} ${w/2-16},${-hh/2+9} ${w/2-4},${-hh/2+9} ${w/2-12},${-hh/2+13}" fill="var(--pass)" stroke="var(--pass)88" stroke-width="1"/>` : '';
    const srcFile = node.source_file ? escHtml(node.source_file.split('/').pop()) : '';
    return `<g class="gn file${dimClass}${selClass}" data-id="${node.id}" transform="translate(${node.x},${node.y})"><rect x="${-w/2}" y="${-hh/2}" width="${w}" height="${hh}" rx="7" fill="${color}22" stroke="${color}"${glowAttr}/><text class="t" x="${-w/2+12}" y="-2">${escHtml(node.label)}</text><text class="s" x="${-w/2+12}" y="14">${srcFile}</text>${epPolygon}</g>`;
  },
  nodeColor(n){
    if(n.type==='fn') return GV.nodeColor(GV.map[n.parentId])||'var(--muted)';
    return GV.LAYER_COLORS[n.layer] || '#8892a4';
  },
  nodeRole(n){
    const layers = {governance:'governance',orchestration:'orchestration',execution:'core',persistence:'adapter',surface:'surface'};
    return layers[n.layer] || 'unknown';
  },
  roleLabel(role){
    return {entrypoint:'Entry point',governance:'Governance',orchestration:'Orchestration',core:'Core logic',adapter:'Adapter',surface:'Output'}[role] || role;
  },
  roleBadge(role){
    const cls = {entrypoint:'role-entry',governance:'role-badge role-unknown',orchestration:'role-badge role-unknown',core:'role-core',adapter:'role-adapter',surface:'role-surface'}[role] || 'role-unknown';
    return `<span class="role-badge ${cls}">${this.roleLabel(role) || role}</span>`;
  },
  isEntrypoint(n){
    const arch = R.Architecture;
    return (arch.Flows || []).some(f => f.Entry && f.Entry.includes(n.source_file||''));
  },
  entryFor(n){
    const arch = R.Architecture;
    return (arch.Flows || []).filter(f => f.Entry && f.Entry.includes(n.source_file||''));
  },
  flowsThrough(file){
    return GV.flowFiles ? (GV.flowFiles[file] || []) : [];
  },
  showDetail(id){
    const n=GV.map[id]; if(!n){ renderHTML(document.getElementById('gdetail'),''); return; }
    const clr=GV.nodeColor(n);
    if(n.type==='fn'){
      GV.sel=n.parentId;
      GV.showDetail(n.parentId);
      GV.draw();
      return;
    }
    GV.sel=id;
    // Build connections
    const calls=[], calledBy=[];
    GV.links.forEach(l=>{
      if(l.s===id && l.kind!=='fn' && l.kind!=='tree') calls.push(l.t);
      if(l.t===id && l.kind!=='fn' && l.kind!=='tree') calledBy.push(l.s);
    });
    const info=GV.getPkgInfo(n.source_file||'');
    const funcs=info?info.funcs:[];
    const pkgName=info?info.pkg:'—';
    const layLabel=n.layer?n.layer:'—';
    const doc=info?info.doc:'';
    const externalLibs=info?null:[]; // external libs only available from Packages directly, not per-file
    const isEntry = GV.isEntrypoint(n);
    const entryFlows = GV.entryFor(n);
    const flowsThrough = GV.flowsThrough(n.source_file||'');
    const role = isEntry ? 'entrypoint' : GV.nodeRole(n);
    // Group calls by layer
    const layerOrder = ['governance','orchestration','execution','persistence','surface'];
    const LAYER_COLORS = {'governance':'var(--info)','orchestration':'var(--advisory)','execution':'var(--pass)','persistence':'#a78bfa','surface':'#f472b6'};
    const groupByLayer = (items) => {
      const groups = {};
      items.forEach(t => {
        const tn = GV.map[t];
        const l = tn ? (tn.layer || 'other') : 'other';
        if (!groups[l]) groups[l] = [];
        groups[l].push(t);
      });
      return groups;
    };
    const callGroups = groupByLayer(calls);
    const calledByGroups = groupByLayer(calledBy);
    // Dependency rule check: core (execution) should not call outward (persistence, surface)
    const ruleViolation = n.layer === 'execution' && (callGroups['persistence'] || callGroups['surface']);
    const ruleWarn = n.layer === 'execution' && callGroups['orchestration'];
    // Get external libs from package data
    const pkgInfo = (R.Architecture.Packages || []).find(p => {
      return Object.keys(p.Files||{}).some(f => {
        const abs = (R.Meta.RepoPath||'') + '/' + f;
        return abs === n.source_file || f.endsWith('/' + (n.source_file||'').split('/').pop());
      });
    });
    const extLibs = pkgInfo ? (pkgInfo.ExternalLibs || []) : [];
    const detail=document.getElementById('gdetail');
    renderHTML(detail, `<div class="dtitle">${escHtml(n.label)}</div>)
      <div class="dsub">${escHtml(n.source_file||n.id)}</div>
      <div class="gd-sublabel" style="margin-top:4px">${GV.roleBadge(role)} <span class="pkg-dep-tag" style="color:${clr};border-color:${clr}55">${escHtml(pkgName)}</span> <span class="pkg-dep-tag" style="color:${clr};border-color:${clr}55">${escHtml(layLabel)}</span></div>
      <div class="gdgrid">
        <div>
          ${doc?`<div class="gfield"><div class="gk">Summary</div><div class="gv">${escHtml(doc)}</div></div>`:''}
          ${funcs.length?`<div class="gfield"><div class="gk">Functions (${funcs.length})</div><div class="gv">${funcs.slice(0,20).map(f=>`<span class="gfn">${escHtml(f)}</span>`).join(' ')}${funcs.length>20?`<span style="color:var(--muted);font-size:10px"> +${funcs.length-20}</span>`:''}</div></div>`:''}
          ${extLibs.length?`<div class="gfield"><div class="gk">External libs (${extLibs.length})</div><div class="gv">${extLibs.slice(0,10).map(l=>`<span class="gfn">${escHtml(l)}</span>`).join(' ')}${extLibs.length>10?`<span style="color:var(--muted);font-size:10px"> +${extLibs.length-10}</span>`:''}</div></div>`:''}
          ${flowsThrough.length?`<div class="gfield"><div class="gk">Part of flows</div><div class="gv">${flowsThrough.map(fn=>`<span class="gfn" onclick="focusFlow('${escHtml(fn)}')">${escHtml(fn)}</span>`).join(' ')}</div></div>`:''}
          ${entryFlows.length?`<div class="gfield"><div class="gk">Entry for flows</div><div class="gv">${entryFlows.map(f=>`<span class="gfn" onclick="focusFlow('${escHtml(f.Name)}')">${escHtml(f.Name)}</span>`).join(' ')}</div></div>`:''}
        </div>
        <div>
          ${ruleViolation||ruleWarn?`<div class="gd-rule ${ruleViolation?'fail':'pass'}">${ruleViolation?'⚠ Dependency rule violation: core (execution) must not depend on adapters (persistence/surface)':'⚠ Suspicious: execution layer calling upward to orchestration'}</div>`:''}
          <div class="gfield"><div class="gk">Calls to (${calls.length})</div>
            <div class="gd-deps">${layerOrder.filter(l=>callGroups[l]).map(l=>`<span class="gd-dep-group"><span class="dd-label" style="color:${LAYER_COLORS[l]||'var(--muted)'}">${l}</span> <span class="dd-count">${callGroups[l].length}</span></span>`).join('')}${Object.keys(callGroups).filter(l=>!layerOrder.includes(l)).length?`<span class="gd-dep-group"><span class="dd-label" style="color:var(--muted)">other</span> <span class="dd-count">${Object.keys(callGroups).filter(l=>!layerOrder.includes(l)).reduce((s,l)=>s+callGroups[l].length,0)}</span></span>`:''}
            </div>
            <div style="margin-top:6px">${calls.slice(0,12).map(t=>{const tn=GV.map[t];return tn?`<span class="gfn">${escHtml(tn.label)}</span>`:`<span class="gfn">${escHtml(t.split('/').pop())}</span>`;}).join(' ')}${calls.length>12?`<span style="color:var(--muted);font-size:10px"> +${calls.length-12}</span>`:''}</div>
          </div>
          <div class="gfield"><div class="gk">Called by (${calledBy.length})</div>
            <div class="gd-deps">${layerOrder.filter(l=>calledByGroups[l]).map(l=>`<span class="gd-dep-group"><span class="dd-label" style="color:${LAYER_COLORS[l]||'var(--muted)'}">${l}</span> <span class="dd-count">${calledByGroups[l].length}</span></span>`).join('')}</div>
            <div style="margin-top:6px">${calledBy.slice(0,12).map(t=>{const tn=GV.map[t];return tn?`<span class="gfn">${escHtml(tn.label)}</span>`:`<span class="gfn">${escHtml(t.split('/').pop())}</span>`;}).join(' ')}${calledBy.length>12?`<span style="color:var(--muted);font-size:10px"> +${calledBy.length-12}</span>`:''}</div>
          </div>
        </div>
      </div>`;
    GV.draw();
  },
  setupUI(){
    const gsvg=document.getElementById('gsvg');
    if(!gsvg) return;
    const tip=document.getElementById('gtip');
    // Edge hover tooltips
    gsvg.addEventListener('pointerover',ev=>{
      const p=ev.target.closest('path.lnk');
