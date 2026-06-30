    }
    if (inCode) { html += esc(raw) + '\n'; i++; continue; }
    // Horizontal rule
    if (/^[-*_]{3,}\s*$/.test(raw)) { flush(); html += '<hr>\n'; i++; continue; }
    // ATX headers
    const hm = raw.match(/^(#{1,6})\s+(.+)$/);
    if (hm) { flush(); html += `<h${hm[1].length}>${inline(hm[2])}</h${hm[1].length}>\n`; i++; continue; }
    // Blockquote
    const bqm = raw.match(/^>\s*(.*)$/);
    if (bqm) {
      if (!inBlockquote) { flush(); html += '<blockquote>\n'; inBlockquote = true; }
      html += (bqm[1] ? inline(bqm[1]) + '<br>\n' : '');
      i++; continue;
    }
    if (inBlockquote) { closeBq(); }
    // Table
    if (raw.includes('|') && /^\|.+\|$/.test(raw.trim())) {
      const cells = raw.split('|').filter(c => c !== undefined).slice(1, -1);
      // Check if it's a separator row
      if (cells.length > 0 && /^[-:| ]+$/.test(cells.join(''))) { i++; continue; }
      if (!inTable) { flush(); html += '<table><thead>\n'; inTable = true; tableHead = true; }
      const tag = tableHead ? 'th' : 'td';
      html += '<tr>';
      cells.forEach(c => { html += `<${tag}>${inline(c.trim())}</${tag}>`; });
      html += '</tr>\n';
      if (tableHead) { html += '</thead>\n<tbody>\n'; tableHead = false; }
      i++; continue;
    }
    if (inTable && !raw.includes('|')) { closeTable(); }
    // Unordered list
    const ulm = raw.match(/^[-*+]\s+(.+)$/);
    if (ulm) {
      if (!inList) { flush(); html += '<ul>\n'; inList = true; }
      html += `<li>${inline(ulm[1])}</li>\n`;
      i++; continue;
    }
    // Ordered list
    const olm = raw.match(/^\d+\.\s+(.+)$/);
    if (olm) {
      if (inList) { html += '</ul>\n'; inList = false; }
      html += `<li>${inline(olm[1])}</li>\n`;
      i++; continue;
    } else if (inList) { closeList(); }
    // Empty line
    if (raw.trim() === '') { html += '\n'; i++; continue; }
    // Paragraph
    let para = '';
    while (i < lines.length && lines[i].trim() !== '' && !/^```/.test(lines[i]) && !/^#{1,6}\s/.test(lines[i]) && !/^>/.test(lines[i]) && !/^\|/.test(lines[i]) && !/^[-*+]\s/.test(lines[i]) && !/^\d+\.\s/.test(lines[i]) && !/^[-*_]{3,}\s*$/.test(lines[i])) {
      para += (para ? ' ' : '') + lines[i].trim();
      i++;
    }
    if (para) { html += `<p>${inline(para)}</p>\n`; }
    else { i++; }
  }
  if (inCode) html += '</code></pre>\n';
  closeTable(); closeList(); closeBq();
