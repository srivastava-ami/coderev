
// ── Safe HTML helpers ────────────────────────────────────────────────────────
// renderHTML safely renders template-generated HTML using DOMParser (safe from XSS).
// Content is from internal template strings, not user input.
function renderHTML(el, html) {
  if (!html) { el.replaceChildren(); return; }
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, 'text/html');
  el.replaceChildren(...doc.body.childNodes);
}

// ── Boot ──────────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  renderHeader();
  renderWarnings();
  renderSummary();
  renderViolations();
  renderFiles();
  renderArchitecture();
  renderDocs();
  renderExceptions();
  renderAIReview();
});

// ── Header ───────────────────────────────────────────────────────────────────
function renderHeader() {
  document.getElementById('repo-name').textContent = R.Meta.RepoName + ' — Code Review';
  document.getElementById('report-meta').textContent =
    'Standards v' + R.Meta.StandardsVersion + ' · ' + R.Meta.Generated;
  const badge = document.getElementById('overall-badge');
  badge.textContent = R.Summary.OverallStatus;
  badge.className = 'badge badge-' + (R.Summary.OverallStatus === 'PASS' ? 'pass' : 'fail');
}

// ── Warnings ─────────────────────────────────────────────────────────────────
function renderWarnings() {
  const el = document.getElementById('warnings-banner');
  if (!R.Warnings || R.Warnings.length === 0) return;
  renderHTML(el, '<div class="warnings"><strong>⚠ Adapter warnings:</strong> ' +)
    R.Warnings.map(w => `<code>${w.Adapter}</code>: ${w.Reason}`).join(' | ') + '</div>';
}

// ── Summary ───────────────────────────────────────────────────────────────────
function renderSummary() {
  const s = R.Summary;
  document.getElementById('s-files').textContent = s.TotalFiles;
  document.getElementById('s-blockers').textContent = s.BySeverity.blocker || 0;
  document.getElementById('s-majors').textContent = s.BySeverity.major || 0;
  document.getElementById('s-total').textContent = s.TotalFindings;

  renderDonut();
  renderBar();
  renderPillars();
}

function renderDonut() {
  const s = R.Summary.BySeverity;
  const items = [
    { label: 'Blocker', value: s.blocker || 0, color: '#ef4444' },
    { label: 'Major',   value: s.major   || 0, color: '#f97316' },
    { label: 'Advisory',value: s.advisory|| 0, color: '#eab308' },
    { label: 'Info',    value: s.info    || 0, color: '#60a5fa' },
  ].filter(i => i.value > 0);

  const total = items.reduce((a, b) => a + b.value, 0);
  if (total === 0) { renderHTML(document.getElementById('donut-chart'), '<text x="100" y="85" text-anchor="middle" fill="#8892a4" font-size="12">No findings</text>'); return; }

  const svg = document.getElementById('donut-chart');
  const cx = 80, cy = 80, r = 60, inner = 38;
  let start = -Math.PI / 2;
  let paths = '';

  for (const item of items) {
    const angle = (item.value / total) * 2 * Math.PI;
    const x1 = cx + r * Math.cos(start), y1 = cy + r * Math.sin(start);
    const x2 = cx + r * Math.cos(start + angle), y2 = cy + r * Math.sin(start + angle);
    const ix1 = cx + inner * Math.cos(start + angle), iy1 = cy + inner * Math.sin(start + angle);
    const ix2 = cx + inner * Math.cos(start), iy2 = cy + inner * Math.sin(start);
    const large = angle > Math.PI ? 1 : 0;
    paths += `<path d="M${x1},${y1} A${r},${r} 0 ${large},1 ${x2},${y2} L${ix1},${iy1} A${inner},${inner} 0 ${large},0 ${ix2},${iy2} Z" fill="${item.color}" opacity=".85"/>`;
    start += angle;
  }

  // Centre text
  paths += `<text x="${cx}" y="${cy-6}" text-anchor="middle" fill="#e2e8f0" font-size="20" font-weight="700">${total}</text>`;
  paths += `<text x="${cx}" y="${cy+10}" text-anchor="middle" fill="#8892a4" font-size="10">findings</text>`;

  // Legend
  let ly = 20;
  for (const item of items) {
    paths += `<rect x="155" y="${ly-8}" width="10" height="10" rx="2" fill="${item.color}"/>`;
    paths += `<text x="170" y="${ly}" fill="#e2e8f0" font-size="11">${item.label} (${item.value})</text>`;
    ly += 22;
  }

  renderHTML(svg, paths);
}

function renderBar() {
  const byPillar = R.Summary.ByPillar || {};
  const entries = Object.entries(byPillar).sort((a, b) => b[1] - a[1]).slice(0, 10);
  if (entries.length === 0) return;

  const svg = document.getElementById('bar-chart');
  const maxVal = Math.max(...entries.map(e => e[1]));
  const barH = 12, gap = 6, leftW = 120, rightPad = 40;
  const chartW = 360 - leftW - rightPad;
  let html = '';
  let y = 10;

  for (const [pillar, count] of entries) {
    const barW = (count / maxVal) * chartW;
    const color = barColorFor(pillar);
    html += `<text x="${leftW - 6}" y="${y + 9}" text-anchor="end" fill="#8892a4" font-size="10" dominant-baseline="middle">${pillar}</text>`;
    html += `<rect x="${leftW}" y="${y}" width="${barW}" height="${barH}" rx="3" fill="${color}" opacity=".8"/>`;
    html += `<text x="${leftW + barW + 5}" y="${y + 9}" fill="#e2e8f0" font-size="10" dominant-baseline="middle">${count}</text>`;
    y += barH + gap;
  }

  svg.setAttribute('viewBox', `0 0 360 ${y}`);
  renderHTML(svg, html);
}

function barColorFor(pillar) {
  const m = { security:'#ef4444', stability:'#f97316', complexity:'#eab308',
               file_structure:'#a78bfa', type_safety:'#60a5fa', observability:'#34d399',
               hardcoding:'#fb923c', documentation:'#94a3b8', performance:'#e879f9' };
  return m[pillar] || '#60a5fa';
}

function renderPillars() {
  const grid = document.getElementById('pillar-grid');
  const pillars = R.Pillars || [];
  if (pillars.length === 0) {
    renderHTML(grid, '<div style="color:var(--muted);font-size:.85rem">No violations detected.</div>');
    return;
  }

  renderHTML(grid, pillars.map(p => {)
    const pct = Math.round(p.Score * 100);
    const color = p.Status === 'FAIL' ? '#ef4444' : p.Status === 'WARN' ? '#f97316' : '#22c55e';
    const count = p.Findings ? p.Findings.length : 0;
    return `<div class="pillar-card">
      <div class="pillar-name status-${p.Status}">${p.Name.replace(/_/g,' ')}</div>
      <div class="pillar-bar-bg"><div class="pillar-bar" style="width:${pct}%;background:${color}"></div></div>
      <div class="pillar-count">${count} finding${count !== 1 ? 's' : ''} · score ${pct}%</div>
    </div>`;
  }).join('');
}

// ── Violations ────────────────────────────────────────────────────────────────
let currentSevFilter = 'all';

function renderViolations() {
  rebuildViolationsTable(R.Findings || []);
}

function rebuildViolationsTable(findings) {
  const query = (document.getElementById('v-search')?.value || '').toLowerCase();
  let filtered = findings.filter(f => {
    if (currentSevFilter !== 'all' && f.Severity !== currentSevFilter) return false;
    if (query && !JSON.stringify(f).toLowerCase().includes(query)) return false;
    return true;
  });

  document.getElementById('violations-count').textContent =
    `Showing ${filtered.length} of ${findings.length} findings`;

  const body = document.getElementById('violations-body');
  if (filtered.length === 0) {
    renderHTML(body, '<tr><td colspan="5" style="text-align:center;padding:32px;color:var(--muted)">No violations match the current filter.</td></tr>');
    return;
  }

  renderHTML(body, filtered.map((f, i) => {)
    const shortFile = f.File ? f.File.split('/').slice(-3).join('/') : '—';
    const hasSnippet = f.Snippet && f.Snippet.trim().length > 0;
    const snip = hasSnippet ? `<button class="snippet-toggle" onclick="toggleSnippet(${i})">show code</button><pre class="snippet" id="snip-${i}">${escHtml(f.Snippet)}</pre>` : '';
    return `<tr>
      <td><span class="sev sev-${f.Severity}">${f.Severity.toUpperCase()}</span></td>
      <td><span class="rule-id">${f.Rule}</span></td>
      <td><span class="file-link" title="${f.File}">${shortFile}</span>${f.Line ? `<br><span style="color:var(--muted);font-size:.7rem">line ${f.Line}</span>` : ''}</td>
      <td>${escHtml(f.Message)}${snip}</td>
      <td style="color:var(--muted);font-size:.78rem">${escHtml(f.Remediation || '')}</td>
    </tr>`;
  }).join('');
}

function toggleSnippet(i) {
  const el = document.getElementById('snip-' + i);
  const btn = el.previousElementSibling;
  el.classList.toggle('open');
  btn.textContent = el.classList.contains('open') ? 'hide code' : 'show code';
}

function setSevFilter(sev, btn) {
  currentSevFilter = sev;
  document.querySelectorAll('.filter-btn[data-sev]').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  rebuildViolationsTable(R.Findings || []);
