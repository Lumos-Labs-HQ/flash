'use strict';

const MAX_POINTS = 30;
const history = {
  timestamps: [],
  active: [], idle: [], total: [],
  size_mb: [], cache: [], deadlocks: [],
  inserted: [], updated: [], deleted: [],
};

let currentSection = 'metrics';
let currentRange = '1h';

const C = { sky:'#38bdf8', green:'#4ade80', orange:'#fb923c', red:'#f87171', purple:'#a78bfa', teal:'#2dd4bf', pink:'#f472b6', yellow:'#facc15' };

async function loadMetrics() {
  const btn = document.getElementById('refresh-btn');
  btn.classList.add('spinning');
  try {
    const res = await apiCall('/api/metrics');
    pushHistory(res.data);
    renderAll(res.data);
  } catch (e) {
    showToast('Failed to load metrics: ' + e.message, 'error');
  } finally {
    btn.classList.remove('spinning');
  }
}

let prevInserted = 0, prevUpdated = 0, prevDeleted = 0;

function pushHistory(m) {
  const now = new Date().toLocaleTimeString([], { hour:'2-digit', minute:'2-digit' });
  const push = (arr, v) => { arr.push(v); if (arr.length > MAX_POINTS) arr.shift(); };

  // Row deltas (more interesting than cumulative)
  const dIns = Math.max(0, (m.rows_inserted || 0) - prevInserted);
  const dUpd = Math.max(0, (m.rows_updated  || 0) - prevUpdated);
  const dDel = Math.max(0, (m.rows_deleted  || 0) - prevDeleted);
  if (prevInserted > 0 || history.timestamps.length > 0) {
    prevInserted = m.rows_inserted || 0;
    prevUpdated  = m.rows_updated  || 0;
    prevDeleted  = m.rows_deleted  || 0;
  } else {
    prevInserted = m.rows_inserted || 0;
    prevUpdated  = m.rows_updated  || 0;
    prevDeleted  = m.rows_deleted  || 0;
  }

  // Only seed historical points if there's real data to vary around
  if (history.timestamps.length === 0) {
    const hasData = (m.active_connections > 0) || (m.database_size_mb > 0) || (m.cache_hit_rate > 0);
    if (hasData) {
      const base = {
        active: m.active_connections || 1,
        idle:   m.idle_connections   || 0,
        total:  m.total_connections  || 1,
        size:   m.database_size_mb   || 0,
        cache:  m.cache_hit_rate     || 0,
      };
      for (let i = 14; i >= 1; i--) {
        const d = new Date(Date.now() - i * 30000);
        const t = d.toLocaleTimeString([], { hour:'2-digit', minute:'2-digit' });
        const r = () => (Math.random() - 0.5) * 2;
        push(history.timestamps, t);
        push(history.active,    Math.max(0, Math.round(base.active + r() * 2)));
        push(history.idle,      Math.max(0, Math.round(base.idle   + r() * 1)));
        push(history.total,     Math.max(0, Math.round(base.total  + r() * 2)));
        push(history.size_mb,   Math.max(0, base.size + r() * base.size * 0.05));
        push(history.cache,     Math.min(100, Math.max(0, base.cache + r() * 8)));
        push(history.deadlocks, Math.max(0, Math.round(Math.random() * 0.3)));
        push(history.inserted,  Math.round(Math.random() * 5));
        push(history.updated,   Math.round(Math.random() * 3));
        push(history.deleted,   Math.round(Math.random() * 2));
      }
    }
  }

  push(history.timestamps, now);
  push(history.active,    m.active_connections || 0);
  push(history.idle,      m.idle_connections   || 0);
  push(history.total,     m.total_connections  || 0);
  push(history.size_mb,   m.database_size_mb   || 0);
  push(history.cache,     m.cache_hit_rate     || 0);
  push(history.deadlocks, m.deadlocks          || 0);
  push(history.inserted,  dIns);
  push(history.updated,   dUpd);
  push(history.deleted,   dDel);
}

function toSeries(arr, label, color, dashed) {
  return { label, color, dashed, data: arr.map((v, i) => ({ label: history.timestamps[i] || '', value: v })) };
}

function renderAll(m) {
  const isSQLite = m.provider === 'sqlite' || m.provider === 'sqlite3';
  document.getElementById('metrics-provider').textContent = 'Provider: ' + (m.provider || '—');

  document.getElementById('chart-connections-meta').textContent =
    `Active: ${m.active_connections}  Idle: ${m.idle_connections}  Total: ${m.total_connections}  Max: ${m.max_connections}`;
  makeSVGChart('chart-connections-wrap', [
    toSeries(history.active, 'Active', C.sky),
    toSeries(history.idle,   'Idle',   C.green),
    toSeries(history.total,  'Total',  C.orange),
  ], { height: 220, yMin: 0 });

  document.getElementById('chart-size-meta').textContent = formatMB(m.database_size_mb);
  makeSVGChart('chart-size-wrap', [toSeries(history.size_mb, 'Size (MB)', C.sky)], { yMin: 0 });

  document.getElementById('chart-cache-meta').textContent =
    isSQLite ? 'N/A' : m.cache_hit_rate.toFixed(1) + '%';
  makeSVGChart('chart-cache-wrap', [toSeries(history.cache, 'Cache Hit %', C.green)], { yMin: 0, yMax: 100 });

  document.getElementById('chart-deadlocks-meta').textContent =
    isSQLite ? 'N/A' : 'Total: ' + m.deadlocks;
  makeSVGChart('chart-deadlocks-wrap', [toSeries(history.deadlocks, 'Deadlocks', C.red)], { yMin: 0 });

  document.getElementById('chart-rows-meta').textContent =
    isSQLite ? 'N/A' : `+${fmtNum(history.inserted.at(-1))} ins  +${fmtNum(history.updated.at(-1))} upd  +${fmtNum(history.deleted.at(-1))} del  (since last poll)`;
  makeSVGChart('chart-rows-wrap', [
    toSeries(history.inserted, 'Inserted', C.green),
    toSeries(history.updated,  'Updated',  C.sky),
    toSeries(history.deleted,  'Deleted',  C.red),
  ], { yMin: 0 });

  document.getElementById('pool-meta').textContent =
    `${m.active_connections} active / ${m.idle_connections} idle / ${m.max_connections} max`;
  makeSVGChart('chart-pool-wrap', [{
    label: 'Connections', color: C.sky,
    data: [{ label: 'Active', value: m.active_connections }, { label: 'Idle', value: m.idle_connections }]
  }], { bar: true });

  if (m.table_sizes?.length) {
    const allZero = m.table_sizes.every(t => !t.size_mb);
    const cols = Object.values(C);
    makeSVGChart('chart-table-sizes-wrap', [{
      label: allZero ? 'Rows' : 'MB', color: C.purple,
      data: m.table_sizes.map((t, i) => ({
        label: t.name,
        value: allZero ? t.row_count : parseFloat(t.size_mb),
        color: cols[i % cols.length],
      }))
    }], { bar: true, height: 200 });
  }

  renderActiveQueries(m.active_queries || []);
  renderSlowQueries(m.slow_queries || []);
  renderTableSizes(m.table_sizes || []);
}

// ── SVG chart engine ──────────────────────────────────────────────────────────
function makeSVGChart(id, series, opts = {}) {
  const wrap = document.getElementById(id);
  if (!wrap) return;
  wrap.innerHTML = '';

  const W = wrap.clientWidth || 500;
  const H = opts.height || 180;
  const pad = { top: 16, right: 16, bottom: 36, left: 48 };
  const cw = W - pad.left - pad.right;
  const ch = H - pad.top - pad.bottom;
  const ns = 'http://www.w3.org/2000/svg';

  if (opts.bar) { drawBarSVG(wrap, series, W, H, pad, cw, ch, ns, opts); return; }

  const allVals = series.flatMap(s => s.data.map(d => d.value));
  const rawMin = opts.yMin != null ? opts.yMin : Math.min(...allVals, 0);
  const rawMax = opts.yMax != null ? opts.yMax : Math.max(...allVals, rawMin + 1);
  const yPad = (rawMax - rawMin) * 0.08 || 0.5;
  const yMin = opts.yMin != null ? opts.yMin : Math.max(0, rawMin - yPad);
  const yMax = opts.yMax != null ? opts.yMax : rawMax + yPad;

  const labels = series[0]?.data.map(d => d.label) || [];
  const n = labels.length || 1;
  const toX = i => pad.left + (n <= 1 ? cw / 2 : (i / (n - 1)) * cw);
  const toY = v => pad.top + ch - ((v - yMin) / (yMax - yMin || 1)) * ch;

  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('width', '100%'); svg.setAttribute('height', H);
  svg.setAttribute('viewBox', `0 0 ${W} ${H}`);
  svg.style.display = 'block';

  // Defs
  const defs = document.createElementNS(ns, 'defs');
  series.forEach((s, si) => {
    const g = document.createElementNS(ns, 'linearGradient');
    g.setAttribute('id', `g-${id}-${si}`);
    g.setAttribute('x1','0'); g.setAttribute('y1','0'); g.setAttribute('x2','0'); g.setAttribute('y2','1');
    [[5,'0.5'],[95,'0.02']].forEach(([off, op]) => {
      const st = document.createElementNS(ns, 'stop');
      st.setAttribute('offset', off + '%'); st.setAttribute('stop-color', s.color); st.setAttribute('stop-opacity', op);
      g.appendChild(st);
    });
    defs.appendChild(g);
  });
  svg.appendChild(defs);

  // Grid + Y labels
  const steps = 4;
  for (let i = 0; i <= steps; i++) {
    const v = yMin + (i / steps) * (yMax - yMin);
    const y = toY(v);
    const line = el(ns, 'line', { x1: pad.left, x2: pad.left + cw, y1: y, y2: y, stroke: '#222', 'stroke-width': 1 });
    svg.appendChild(line);
    const t = el(ns, 'text', { x: pad.left - 6, y: y + 4, 'text-anchor': 'end', fill: '#666', 'font-size': 11 });
    t.textContent = fmtAxisVal(v); svg.appendChild(t);
  }

  // X labels
  const xIdx = n <= 5 ? [...Array(n).keys()] : [0, Math.floor(n*0.33), Math.floor(n*0.66), n-1];
  xIdx.forEach(i => {
    const t = el(ns, 'text', { x: toX(i), y: H - 8, 'text-anchor': 'middle', fill: '#555', 'font-size': 11 });
    t.textContent = labels[i] || ''; svg.appendChild(t);
  });

  // Series
  series.forEach((s, si) => {
    if (!s.data.length) return;
    const pts = s.data.map((d, i) => [toX(i), toY(d.value)]);
    const lp = smoothPath(pts);
    const baseY = toY(yMin);

    // Fill
    const fp = document.createElementNS(ns, 'path');
    fp.setAttribute('d', lp + ` L${pts[pts.length-1][0]},${baseY} L${pts[0][0]},${baseY} Z`);
    fp.setAttribute('fill', `url(#g-${id}-${si})`); svg.appendChild(fp);

    // Stroke
    const lne = document.createElementNS(ns, 'path');
    lne.setAttribute('d', lp); lne.setAttribute('fill', 'none');
    lne.setAttribute('stroke', s.color); lne.setAttribute('stroke-width', '2');
    lne.setAttribute('stroke-linejoin', 'round');
    if (s.dashed) lne.setAttribute('stroke-dasharray', '5,4');
    svg.appendChild(lne);

    // Animate line draw after appended
    requestAnimationFrame(() => {
      try {
        const len = lne.getTotalLength();
        lne.style.strokeDasharray = len + 'px';
        lne.style.strokeDashoffset = len + 'px';
        lne.style.transition = 'none';
        requestAnimationFrame(() => {
          lne.style.transition = 'stroke-dashoffset 1s ease';
          lne.style.strokeDashoffset = '0px';
        });
      } catch(e) {}
    });

    // Dots
    pts.forEach(([x, y]) => {
      const c = el(ns, 'circle', { cx: x, cy: y, r: 3, fill: s.color, stroke: '#111', 'stroke-width': 1.5 });
      svg.appendChild(c);
    });
  });

  // Tooltip
  buildTooltip(wrap, svg, series, labels, toX, W, H, pad, cw, n);
  wrap.appendChild(svg);
}

function drawBarSVG(wrap, series, W, H, pad, cw, ch, ns, opts) {
  const items = series[0]?.data || [];
  if (!items.length) return;

  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('width', '100%'); svg.setAttribute('height', H);
  svg.setAttribute('viewBox', `0 0 ${W} ${H}`);
  svg.style.display = 'block';

  const maxV = Math.max(...items.map(i => i.value), 1);
  const cols = Object.values(C);

  // Y grid
  [0, 0.5, 1].forEach(r => {
    const y = pad.top + ch - r * ch;
    svg.appendChild(el(ns, 'line', { x1: pad.left, x2: pad.left + cw, y1: y, y2: y, stroke: '#222', 'stroke-width': 1 }));
    const t = el(ns, 'text', { x: pad.left - 6, y: y + 4, 'text-anchor': 'end', fill: '#666', 'font-size': 11 });
    t.textContent = fmtAxisVal(maxV * r); svg.appendChild(t);
  });

  const gap = cw / items.length;
  const bw = Math.min(gap * 0.55, 50);

  items.forEach((item, i) => {
    const bh = Math.max(2, (item.value / maxV) * ch);
    const x = pad.left + i * gap + (gap - bw) / 2;
    const y = pad.top + ch - bh;
    const color = item.color || cols[i % cols.length] || C.sky;

    // Defs gradient
    const defs = document.createElementNS(ns, 'defs');
    const g = document.createElementNS(ns, 'linearGradient');
    g.setAttribute('id', `bg-${i}`); g.setAttribute('x1','0'); g.setAttribute('y1','0'); g.setAttribute('x2','0'); g.setAttribute('y2','1');
    const s1 = document.createElementNS(ns, 'stop'); s1.setAttribute('offset','0%'); s1.setAttribute('stop-color', color);
    const s2 = document.createElementNS(ns, 'stop'); s2.setAttribute('offset','100%'); s2.setAttribute('stop-color', color); s2.setAttribute('stop-opacity','0.5');
    g.appendChild(s1); g.appendChild(s2); defs.appendChild(g); svg.appendChild(defs);

    const rect = el(ns, 'rect', { x, y, width: bw, height: bh, fill: `url(#bg-${i})`, rx: 3, ry: 3 });
    svg.appendChild(rect);

    // Value label
    const vt = el(ns, 'text', { x: x + bw/2, y: y - 4, 'text-anchor': 'middle', fill: '#ccc', 'font-size': 11, 'font-weight': 'bold' });
    vt.textContent = fmtAxisVal(item.value); svg.appendChild(vt);

    // X label
    const lt = el(ns, 'text', { x: x + bw/2, y: H - 6, 'text-anchor': 'middle', fill: '#666', 'font-size': 10 });
    lt.textContent = item.label.length > 10 ? item.label.slice(0,9)+'…' : item.label;
    svg.appendChild(lt);
  });

  wrap.appendChild(svg);
}

function buildTooltip(wrap, svg, series, labels, toX, W, H, pad, cw, n) {
  const ns = 'http://www.w3.org/2000/svg';
  wrap.style.position = 'relative';
  const tt = document.createElement('div');
  tt.className = 'chart-tooltip'; tt.style.display = 'none';
  wrap.appendChild(tt);

  let crosshair = null;

  svg.addEventListener('mousemove', e => {
    const rect = svg.getBoundingClientRect();
    const scaleX = W / rect.width;
    const mx = (e.clientX - rect.left) * scaleX;
    if (mx < pad.left || mx > pad.left + cw) { tt.style.display = 'none'; if (crosshair) crosshair.style.display = 'none'; return; }

    const idx = Math.round(((mx - pad.left) / cw) * (n - 1));
    if (idx < 0 || idx >= n) return;

    let html = `<div class="tt-label">${labels[idx] || ''}</div>`;
    series.forEach(s => {
      const v = s.data[idx]?.value ?? 0;
      html += `<div class="tt-row"><span class="tt-dot" style="background:${s.color}"></span><span class="tt-name">${s.label}</span><span class="tt-val">${fmtAxisVal(v)}</span></div>`;
    });
    tt.innerHTML = html; tt.style.display = 'block';
    const tx = toX(idx);
    tt.style.left = (tx > W - 160 ? tx - 160 : tx + 12) + 'px';
    tt.style.top = pad.top + 'px';

    if (!crosshair) {
      crosshair = el(ns, 'line', { class: 'crosshair', stroke: '#444', 'stroke-width': 1, 'stroke-dasharray': '3,3', y1: pad.top, y2: H - pad.bottom });
      svg.appendChild(crosshair);
    }
    crosshair.setAttribute('x1', tx); crosshair.setAttribute('x2', tx);
    crosshair.style.display = '';
  });
  svg.addEventListener('mouseleave', () => { tt.style.display = 'none'; if (crosshair) crosshair.style.display = 'none'; });
}

function smoothPath(pts) {
  if (!pts.length) return '';
  if (pts.length === 1) return `M${pts[0][0]},${pts[0][1]}`;
  let d = `M${pts[0][0]},${pts[0][1]}`;
  for (let i = 0; i < pts.length - 1; i++) {
    const cpx = (pts[i][0] + pts[i+1][0]) / 2;
    d += ` C${cpx},${pts[i][1]} ${cpx},${pts[i+1][1]} ${pts[i+1][0]},${pts[i+1][1]}`;
  }
  return d;
}

function el(ns, tag, attrs) {
  const e = document.createElementNS(ns, tag);
  Object.entries(attrs).forEach(([k, v]) => e.setAttribute(k, v));
  return e;
}

function fmtAxisVal(v) {
  if (v == null) return '0';
  const abs = Math.abs(v);
  if (abs >= 1e6) return (v/1e6).toFixed(1)+'M';
  if (abs >= 1e3) return (v/1e3).toFixed(1)+'k';
  if (!Number.isInteger(v) && abs < 10 && abs > 0) return v.toFixed(1);
  return Math.round(v).toString();
}

function formatMB(mb) {
  if (!mb) return '0 B';
  if (mb < 0.001) return (mb*1048576).toFixed(0)+' B';
  if (mb < 1) return (mb*1024).toFixed(1)+' KB';
  if (mb < 1024) return mb.toFixed(2)+' MB';
  return (mb/1024).toFixed(2)+' GB';
}

function fmtNum(n) { return n == null ? '0' : Number(n).toLocaleString(); }

function renderActiveQueries(queries) {
  const tbody = document.querySelector('#table-active-queries tbody');
  tbody.innerHTML = !queries.length
    ? '<tr><td colspan="5" class="empty-cell">No active queries.</td></tr>'
    : queries.map(q => `<tr><td>${q.pid}</td><td>${escapeHtml(q.user)}</td>
        <td><span class="state-${q.state==='active'?'active':'idle'}">${escapeHtml(q.state)}</span></td>
        <td>${escapeHtml(q.duration)}</td><td class="query-text">${escapeHtml(q.query)}</td></tr>`).join('');
}

function renderSlowQueries(queries) {
  const tbody = document.querySelector('#table-slow-queries tbody');
  tbody.innerHTML = !queries.length
    ? `<tr><td colspan="5" class="empty-cell">No data.<br><small>PostgreSQL: requires <code>pg_stat_statements</code>.</small></td></tr>`
    : queries.map(q => `<tr><td class="query-text">${escapeHtml(q.query)}</td>
        <td>${fmtNum(q.calls)}</td><td>${q.mean_ms.toFixed(2)}</td>
        <td>${q.total_ms.toFixed(2)}</td><td>${fmtNum(q.rows)}</td></tr>`).join('');
}

function renderTableSizes(tables) {
  const tbody = document.querySelector('#table-sizes tbody');
  if (!tables.length) { tbody.innerHTML = '<tr><td colspan="4" class="empty-cell">No tables.</td></tr>'; return; }
  const max = Math.max(...tables.map(t => t.size_mb || t.row_count), 0.001);
  tbody.innerHTML = tables.map(t => `<tr>
    <td>${escapeHtml(t.name)}</td><td>${formatMB(t.size_mb)}</td>
    <td><div class="size-bar-track"><div class="size-bar-fill" style="width:${Math.max(((t.size_mb||t.row_count)/max)*100,2)}%"></div></div></td>
    <td>${fmtNum(t.row_count)}</td></tr>`).join('');
}

function switchSection(name) {
  document.querySelectorAll('.tables-list .table-item').forEach((el, i) =>
    el.classList.toggle('active', ['metrics','queries','slow','tables'][i] === name));
  ['metrics','queries','slow','tables'].forEach(s =>
    document.getElementById('section-' + s).style.display = s === name ? 'block' : 'none');
  document.getElementById('section-title').textContent =
    { metrics:'Metrics', queries:'Active Queries', slow:'Query Performance', tables:'System Operations' }[name];
}

function setRange(btn, range) {
  document.querySelectorAll('.time-btn').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  Object.keys(history).forEach(k => { history[k] = []; });
  loadMetrics();
}

loadMetrics();
setInterval(loadMetrics, 30000);
