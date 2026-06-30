      if(p){
        const lk=GV.links.find(l=>{const a=GV.map[l.s],b=GV.map[l.t];return a&&b&&Math.abs(a.x-(parseFloat(p.getAttribute('d')?.match(/M\s*([\d.-]+)/)?.[1]||0)))<50;});
        if(lk){ const s=GV.map[l.s],t=GV.map[l.t]; if(s&&t&&tip){ tip.textContent=`${s.label||s.id} → ${t.label||t.id} : ${lk.relation||lk.kind}`; tip.className='show'; } }
      }
    });
    gsvg.addEventListener('pointerout',ev=>{ if(ev.target.closest('path.lnk')&&tip) tip.className=''; });
    // Node hover tooltips
    gsvg.addEventListener('pointerover',ev=>{
      const g=ev.target.closest('.gn');
      if(!g) return;
      const n=GV.map[g.dataset.id]; if(!n||!tip) return;
      if(n.type==='file'){
        const role=GV.isEntrypoint(n)?'entrypoint':GV.nodeRole(n);
        const fns=GV.getPkgInfo(n.source_file||'')?.funcs||[];
        tip.textContent=`${n.label}  ·  ${role}  ·  ${n.layer}  ·  ${fns.length} funcs`;
      } else if(n.type==='fn'){
        tip.textContent=`${n.label}()  ·  ${n.parentId}`;
      } else if(n.type==='layer'){
        tip.textContent=`${n.label} layer`;
      } else {
        tip.textContent=n.label;
      }
      tip.className='show';
    });
    gsvg.addEventListener('pointerout',ev=>{ if(ev.target.closest('.gn')&&tip) tip.className=''; });
    // Position tooltip on pointer move
    gsvg.addEventListener('pointermove',ev=>{
      if(tip&&tip.className==='show'){
        const r=gsvg.getBoundingClientRect();
        tip.style.left=(ev.clientX-r.left+12)+'px';
        tip.style.top=(ev.clientY-r.top-8)+'px';
      }
    });
    let drag=null;
    gsvg.addEventListener('pointerdown',ev=>{
      const g=ev.target.closest('.gn');
      const pt=svgPt(ev);
      if(g && g.classList.contains('file')){ const n=GV.map[g.dataset.id]; if(!n)return; drag={node:n,ox:pt.x-n.x,oy:pt.y-n.y}; n.pin=true; gsvg.setPointerCapture(ev.pointerId); }
      else if(g && g.classList.contains('fn')){ /* handled on click */ }
      else { drag={pan:true,sx:ev.clientX,sy:ev.clientY,tx:GV.t.x,ty:GV.t.y}; gsvg.classList.add('grab'); }
    });
    gsvg.addEventListener('pointermove',ev=>{
      if(!drag)return;
      if(drag.pan){ GV.t.x=drag.tx+(ev.clientX-drag.sx); GV.t.y=drag.ty+(ev.clientY-drag.sy); GV.draw(); }
      else if(drag.node){ const pt=svgPt(ev); drag.node.x=pt.x-drag.ox; drag.node.y=pt.y-drag.oy; GV.draw(); }
    });
    gsvg.addEventListener('pointerup',ev=>{ drag=null; gsvg.classList.remove('grab'); });
    gsvg.addEventListener('click',ev=>{
      const g=ev.target.closest('.gn'); if(!g)return;
      const id=g.dataset.id;
      if(g.classList.contains('fn')){ GV.showDetail(g.dataset.parent); return; }
      if(g.classList.contains('file')){ GV.showDetail(id); }
    });
    gsvg.addEventListener('dblclick',ev=>{
      const g=ev.target.closest('.gn'); if(!g||!g.classList.contains('file'))return;
      const id=g.dataset.id; if(GV.expand.has(id))GV.expand.delete(id); else GV.expand.add(id);
      GV.build(); GV.fit();
    });
    gsvg.addEventListener('wheel',ev=>{
      ev.preventDefault();
      const r=gsvg.getBoundingClientRect();
      const px=ev.clientX-r.left,py=ev.clientY-r.top;
      const f=ev.deltaY<0?1.12:0.89;
      const k2=Math.max(0.3,Math.min(2.6,GV.t.k*f));
      const wx=(px-GV.t.x)/GV.t.k, wy=(py-GV.t.y)/GV.t.k;
      GV.t.k=k2; GV.t.x=px-wx*k2; GV.t.y=py-wy*k2; GV.draw();
    },{passive:false});
    // Buttons
    const b_topology=document.getElementById('b_topology');
    const b_hierarchy=document.getElementById('b_hierarchy');
    const b_funcs=document.getElementById('b_funcs');
    const b_reset=document.getElementById('b_reset');
    if(b_topology) b_topology.onclick=()=>{ GV.mode='topology'; b_topology.classList.add('on'); b_hierarchy.classList.remove('on'); GV.build(); GV.fit(); };
    if(b_hierarchy) b_hierarchy.onclick=()=>{ GV.mode='hierarchy'; b_hierarchy.classList.add('on'); b_topology.classList.remove('on'); GV.build(); GV.fit(); };
    if(b_funcs) b_funcs.onclick=()=>{
      const allOn=GV.nodes.filter(n=>n.type==='file').every(n=>GV.expand.has(n.id));
      GV.expand=allOn?new Set():new Set(GV.nodes.filter(n=>n.type==='file').map(n=>n.id));
      b_funcs.classList.toggle('on',!allOn); GV.build(); GV.fit();
    };
    if(b_reset) b_reset.onclick=()=>{ GV.build(); GV.fit(); };
    // Legend
    const legend=document.getElementById('glegend');
    if(legend){
      const layers=[...new Set(GV.nodes.filter(n=>n.type==='file').map(n=>n.layer))];
      renderHTML(legend, layers.map(l=>`<span><b style="background:${GV.nodeColor({layer:l})}"></b>${l}</span>`).join(''));
    }
    window.addEventListener('resize',()=>GV.draw());
  }
};
function svgPt(ev){
  const gsvg=document.getElementById('gsvg'); if(!gsvg) return {x:0,y:0};
  const r=gsvg.getBoundingClientRect(); return {x:(ev.clientX-r.left-GV.t.x)/GV.t.k, y:(ev.clientY-r.top-GV.t.y)/GV.t.k};
}
function cssV(v){ return getComputedStyle(document.documentElement).getPropertyValue(v).trim(); }

// ── Exceptions ────────────────────────────────────────────────────────────────
function renderExceptions() {
  const el = document.getElementById('exceptions-content');
  const excs = R.Exceptions || [];

  if (excs.length === 0) {
    renderHTML(el, '<div style="color:var(--muted);font-size:.85rem;padding:20px">No active exceptions registered.</div>');
    return;
  }

  renderHTML(el, `<table>)
    <thead><tr><th>Rule</th><th>Scope</th><th>Justification</th><th>Approved By</th><th>Expires</th><th>Ticket</th></tr></thead>
    <tbody>${excs.map(e => `<tr>
      <td><span class="rule-id">${e.rule || '—'}</span></td>
      <td><span class="file-link">${e.file_or_module || '—'}</span></td>
      <td style="font-size:.78rem">${escHtml(e.justification || '')}</td>
      <td style="font-size:.78rem;color:var(--muted)">${e.approved_by || '—'}</td>
      <td style="font-size:.78rem;color:${isExpired(e.expires) ? 'var(--fail)' : 'var(--muted)'}">${e.expires || '—'}</td>
      <td>${e.ticket ? `<a href="${escHtml(e.ticket)}" target="_blank">${escHtml(e.ticket)}</a>` : '—'}</td>
    </tr>`).join('')}</tbody>
  </table>`;
}

function isExpired(dateStr) {
  if (!dateStr) return false;
  return new Date(dateStr) < new Date();
}

// ── AI Review ─────────────────────────────────────────────────────────────────
function renderAIReview() {
  const el = document.getElementById('aireview-content');
  const review = R.AIReview || '';
  if (!review.trim()) {
    renderHTML(el, `<div style="color:var(--muted);font-size:.88rem;padding:32px 0;text-align:center">)
      <div style="font-size:1.5rem;margin-bottom:12px">🤖</div>
      <div>No AI review yet.</div>
      <div style="margin-top:8px;font-size:.8rem">Run <code style="background:var(--surface2);padding:2px 6px;border-radius:4px">coderev . --review --diff &lt;base-ref&gt;</code> to generate one.</div>
    </div>`;
    return;
  }
  renderHTML(el, `)
    <div style="background:rgba(96,165,250,.06);border:1px solid rgba(96,165,250,.15);border-radius:8px;padding:8px 14px;font-size:.78rem;margin-bottom:16px;color:var(--muted)">
      🤖 Gap-detector output — deterministic scan covers 90%; this highlights what rules may have missed.
    </div>
    <div class="arch-text">${markdownToHtml(review)}</div>`;
}

// ── Tab navigation ────────────────────────────────────────────────────────────
function showTab(name, btn) {
  document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.getElementById('panel-' + name).classList.add('active');
  btn.classList.add('active');
  if (name === 'architecture' && !archTabShown && R.GraphJSON) {
    archTabShown = true;
    setTimeout(() => GV.init(), 50);
  }
}

// ── Utilities ─────────────────────────────────────────────────────────────────
function escHtml(s) {
  if (!s) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function markdownToHtml(md) {
  if (!md) return '';
  const lines = md.split('\n');
  let html = '', i = 0, inCode = false, inList = false, inBlockquote = false, inTable = false, tableHead = false;
  const closeList = () => { if (inList) { html += '</ul>\n'; inList = false; } };
  const closeBq = () => { if (inBlockquote) { html += '</blockquote>\n'; inBlockquote = false; } };
  const closeTable = () => { if (inTable) { html += '</tbody></table>\n'; inTable = false; tableHead = false; } };
  const flush = () => { closeTable(); closeList(); closeBq(); };
  const esc = (s) => escHtml(s);
  const inline = (s) => {
    // Escape HTML first, then apply markdown inline formatting
    s = esc(s);
    // Images: ![alt](url)
    s = s.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" style="max-width:100%">');
    // Links: [text](url)
    s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');
    // Bold+italic: ***text***
    s = s.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>');
    // Bold: **text** or __text__
    s = s.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    s = s.replace(/__(.+?)__/g, '<strong>$1</strong>');
    // Italic: *text* or _text_
    s = s.replace(/\*(.+?)\*/g, '<em>$1</em>');
    s = s.replace(/_(.+?)_/g, '<em>$1</em>');
    // Strikethrough: ~~text~~
    s = s.replace(/~~(.+?)~~/g, '<del>$1</del>');
    // Inline code: `code`
    s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
    return s;
  };
  while (i < lines.length) {
    let raw = lines[i];
    const trimmed = raw.trimEnd();
    // Fenced code block start/end
    if (/^```/.test(raw)) {
      flush();
      if (inCode) { html += '</code></pre>\n'; inCode = false; i++; continue; }
      const lang = raw.slice(3).trim();
      html += `<pre><code${lang?' class="lang-'+esc(lang)+'"':''}>`;
      inCode = true; i++; continue;
