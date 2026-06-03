'use strict';

const MAX_POINTS = 30;
const history = {
  timestamps: [],
  active: [], idle: [], total: [], max_conn: [],
  size_mb: [],
  cache: [],
  deadlocks: [],
  inserted: [], updated: [], deleted: [],
};

let currentSection = 'metrics';
let currentRange = '1h';
let metricsData = null;

// ── Palette — all bright, readable on dark bg ─────────────────────────────────
const C = {
  sky:    '#38bdf8',
  green:  '#4ade80',
  orange: '#fb923c',
  red:    '#f87171',
  purple: '#a78bfa',
  yellow: '#facc15',
  teal:   '#2dd4bf',
  pink:   '#f472b6',
};

async function loadMetrics() {
  const btn = document.getElementById('refresh-btn');
  btn.classList.add('spinning');
  try {
    const res = await apiCall('/api/metrics');
    metricsData = res.data;
    pushHistory(metricsData);
    renderAll(metricsData);
  } catch (e) {
    showToast('Failed to load metrics: ' + e.message, 'error');
  } finally {
    btn.classList.remove('spinning');
  }
}

function pushHistory(m) {
  const now = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  push(history.timestamps, now);
  push(history.active,    m.active_connections  || 0);
  push(history.idle,      m.idle_connections    || 0);
  push(history.total,     m.total_connections   || 0);
  push(history.max_conn,  m.max_connections     || 0);
  push(history.size_mb,   m.database_size_mb    || 0);
  push(history.cache,     m.cache_hit_rate      || 0);
  push(history.deadlocks, m.deadlocks           || 0);
  push(history.inserted,  m.rows_inserted       || 0);
  push(history.updated,   m.rows_updated        || 0);
  push(history.deleted,   m.rows_deleted        || 0);
}

function push(arr, val) {
  arr.push(val);
  if (arr.length > MAX_POINTS) arr.shift();
}

function renderAll(m) {
  const isSQLite = m.provider === 'sqlite' || m.provider === 'sqlite3';
  document.getElementById('metrics-provider').textContent = 'Provider: ' + (m.provider || 'unknown');

  drawMultiLine('chart-connections', history.timestamps, [
    { label: 'Active', data: history.active,   color: C.sky },
    { label: 'Idle',   data: history.idle,     color: C.green },
    { label: 'Total',  data: history.total,    color: C.orange },
    { label: 'Max',    data: history.max_conn, color: C.red, dashed: true },
  ], { meta: `Active: ${m.active_connections}  Idle: ${m.idle_connections}  Total: ${m.total_connections}  Max: ${m.max_connections}` });

  drawLine('chart-size', history.timestamps, history.size_mb, C.sky,
    { meta: formatMB(m.database_size_mb), yLabel: 'MB' });

  drawLine('chart-cache', history.timestamps, history.cache, C.green,
    { meta: isSQLite ? 'N/A for SQLite' : m.cache_hit_rate.toFixed(1) + '%', yMin: 0, yMax: 100, yLabel: '%' });

  drawLine('chart-deadlocks', history.timestamps, history.deadlocks, C.red,
    { meta: isSQLite ? 'N/A for SQLite' : 'Total: ' + m.deadlocks, yLabel: 'count' });

  drawMultiLine('chart-rows', history.timestamps, [
    { label: 'Inserted', data: history.inserted, color: C.green },
    { label: 'Updated',  data: history.updated,  color: C.sky },
    { label: 'Deleted',  data: history.deleted,  color: C.red },
  ], { meta: isSQLite ? 'N/A for SQLite' :
    `Inserted: ${fmtNum(m.rows_inserted)}  Updated: ${fmtNum(m.rows_updated)}  Deleted: ${fmtNum(m.rows_deleted)}` });

  // Connection pool bar — active vs idle only
  document.getElementById('pool-meta').textContent =
    `${m.active_connections} active / ${m.idle_connections} idle / ${m.max_connections} max`;
  drawBar('chart-pool', [
    { label: 'Active', value: m.active_connections, color: C.sky },
    { label: 'Idle',   value: m.idle_connections,   color: C.green },
  ]);

  // Table sizes bar
  if (m.table_sizes && m.table_sizes.length > 0) {
    const allZero = m.table_sizes.every(t => !t.size_mb);
    drawBar('chart-table-sizes',
      m.table_sizes.map((t, i) => ({
        label: t.name,
        value: allZero ? t.row_count : t.size_mb,
        color: Object.values(C)[i % Object.values(C).length],
      })), { yLabel: allZero ? 'rows' : 'MB' }
    );
  }

  renderActiveQueries(m.active_queries || []);
  renderSlowQueries(m.slow_queries || []);
  renderTableSizes(m.table_sizes || []);
}

// ── Canvas helpers ────────────────────────────────────────────────────────────
function setupCanvas(id) {
  const canvas = document.getElementById(id);
  if (!canvas) return null;
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.parentElement.clientWidth - 32;
  const h = 160;
  canvas.width  = w * dpr;
  canvas.height = h * dpr;
  canvas.style.width  = w + 'px';
  canvas.style.height = h + 'px';
  const ctx = canvas.getContext('2d');
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, w, h);
  return { ctx, w, h };
}

const PAD = { top: 14, right: 16, bottom: 28, left: 48 };

function chartArea(w, h) {
  return {
    x0: PAD.left,
    y0: PAD.top,
    x1: w - PAD.right,
    y1: h - PAD.bottom,
    width:  w - PAD.left - PAD.right,
    height: h - PAD.top  - PAD.bottom,
  };
}

function drawGridAndAxes(ctx, w, h, minV, maxV, labels, steps = 4) {
  const a = chartArea(w, h);

  // Grid lines + Y labels
  ctx.font = '10px Inter, sans-serif';
  ctx.textAlign = 'right';
  for (let i = 0; i <= steps; i++) {
    const ratio = i / steps;
    const y = a.y1 - ratio * a.height;
    const val = minV + ratio * (maxV - minV);

    ctx.strokeStyle = '#252525';
    ctx.lineWidth = 1;
    ctx.beginPath(); ctx.moveTo(a.x0, y); ctx.lineTo(a.x1, y); ctx.stroke();

    ctx.fillStyle = '#666';
    ctx.fillText(fmtAxisVal(val), a.x0 - 4, y + 3.5);
  }

  // X axis line
  ctx.strokeStyle = '#333';
  ctx.lineWidth = 1;
  ctx.beginPath(); ctx.moveTo(a.x0, a.y1); ctx.lineTo(a.x1, a.y1); ctx.stroke();

  // X labels
  if (labels && labels.length > 0) {
    ctx.fillStyle = '#555';
    ctx.textAlign = 'center';
    const pts = labels.length;
    const indices = pts <= 3 ? [...Array(pts).keys()] :
      [0, Math.floor(pts / 2), pts - 1];
    indices.forEach(i => {
      if (!labels[i]) return;
      const x = a.x0 + (i / (pts - 1 || 1)) * a.width;
      ctx.fillText(labels[i], x, h - 6);
    });
  }
}

function toY(v, minV, maxV, a) {
  const ratio = maxV === minV ? 0.5 : (v - minV) / (maxV - minV);
  return a.y1 - ratio * a.height;
}

function drawLine(id, labels, data, color, opts = {}) {
  const c = setupCanvas(id);
  if (!c) return;
  const { ctx, w, h } = c;

  // Update meta text
  const metaEl = document.getElementById(id + '-meta');
  if (metaEl && opts.meta != null) metaEl.textContent = opts.meta;

  const pts = data.length === 0 ? [0] :
              data.length === 1 ? [data[0], data[0]] : data;
  const lbs = labels.length <= 1 ? ['', ''] : labels;

  const minV = opts.yMin != null ? opts.yMin : Math.min(...pts);
  const maxV = opts.yMax != null ? opts.yMax : Math.max(...pts, minV + 1);

  drawGridAndAxes(ctx, w, h, minV, maxV, lbs);

  const a = chartArea(w, h);
  const xStep = a.width / (pts.length - 1 || 1);

  // Fill
  const grad = ctx.createLinearGradient(0, a.y0, 0, a.y1);
  grad.addColorStop(0, color + '44');
  grad.addColorStop(1, color + '05');
  ctx.fillStyle = grad;
  ctx.beginPath();
  pts.forEach((v, i) => {
    const x = a.x0 + i * xStep;
    const y = toY(v, minV, maxV, a);
    i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
  });
  ctx.lineTo(a.x0 + (pts.length - 1) * xStep, a.y1);
  ctx.lineTo(a.x0, a.y1);
  ctx.closePath();
  ctx.fill();

  // Line
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  ctx.lineJoin = 'round';
  ctx.beginPath();
  pts.forEach((v, i) => {
    const x = a.x0 + i * xStep;
    const y = toY(v, minV, maxV, a);
    i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
  });
  ctx.stroke();

  // Dot + annotation on latest value
  const lx = a.x0 + (pts.length - 1) * xStep;
  const lv = pts[pts.length - 1];
  const ly = toY(lv, minV, maxV, a);
  ctx.beginPath(); ctx.arc(lx, ly, 4, 0, Math.PI * 2);
  ctx.fillStyle = color; ctx.fill();
  ctx.strokeStyle = '#0f0f0f'; ctx.lineWidth = 1.5; ctx.stroke();

  // Value label on dot
  ctx.fillStyle = '#fff';
  ctx.font = 'bold 10px Inter, sans-serif';
  ctx.textAlign = 'center';
  ctx.fillText(fmtAxisVal(lv), lx, ly - 8);
}

function drawMultiLine(id, labels, series, opts = {}) {
  const c = setupCanvas(id);
  if (!c) return;
  const { ctx, w, h } = c;

  const metaEl = document.getElementById(id + '-meta');
  if (metaEl && opts.meta) metaEl.textContent = opts.meta;

  const allVals = series.flatMap(s => s.data);
  if (allVals.length === 0) return;

  const minV = Math.min(...allVals);
  const maxV = Math.max(...allVals, minV + 1);
  const lbs = labels.length <= 1 ? ['', ''] : labels;

  drawGridAndAxes(ctx, w, h, minV, maxV, lbs);

  const a = chartArea(w, h);

  series.forEach(s => {
    const pts = s.data.length === 0 ? [0] :
                s.data.length === 1 ? [s.data[0], s.data[0]] : s.data;
    const xStep = a.width / (pts.length - 1 || 1);

    ctx.setLineDash(s.dashed ? [5, 4] : []);
    ctx.strokeStyle = s.color;
    ctx.lineWidth = 2;
    ctx.lineJoin = 'round';
    ctx.beginPath();
    pts.forEach((v, i) => {
      const x = a.x0 + i * xStep;
      const y = toY(v, minV, maxV, a);
      i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
    });
    ctx.stroke();

    // Dot on latest
    ctx.setLineDash([]);
    const lx = a.x0 + (pts.length - 1) * xStep;
    const lv = pts[pts.length - 1];
    const ly = toY(lv, minV, maxV, a);
    ctx.beginPath(); ctx.arc(lx, ly, 3.5, 0, Math.PI * 2);
    ctx.fillStyle = s.color; ctx.fill();

    // Value label
    ctx.fillStyle = s.color;
    ctx.font = 'bold 10px Inter, sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText(fmtAxisVal(lv), lx, ly - 8);
  });
  ctx.setLineDash([]);
}

function drawBar(id, items, opts = {}) {
  const c = setupCanvas(id);
  if (!c || items.length === 0) return;
  const { ctx, w, h } = c;

  const maxV = Math.max(...items.map(i => i.value), 1);
  const a = chartArea(w, h);

  // Y grid
  ctx.font = '10px Inter, sans-serif';
  ctx.textAlign = 'right';
  [0, 0.5, 1].forEach(ratio => {
    const y = a.y1 - ratio * a.height;
    ctx.strokeStyle = '#252525'; ctx.lineWidth = 1;
    ctx.beginPath(); ctx.moveTo(a.x0, y); ctx.lineTo(a.x1, y); ctx.stroke();
    ctx.fillStyle = '#666';
    ctx.fillText(fmtAxisVal(maxV * ratio), a.x0 - 4, y + 3.5);
  });

  const gap = a.width / items.length;
  const barW = Math.max(6, Math.min(gap * 0.55, 50));

  items.forEach((item, i) => {
    const bh = Math.max(2, (item.value / maxV) * a.height);
    const x = a.x0 + i * gap + (gap - barW) / 2;
    const y = a.y1 - bh;

    // Bar with gradient
    const grad = ctx.createLinearGradient(0, y, 0, a.y1);
    grad.addColorStop(0, item.color);
    grad.addColorStop(1, item.color + '66');
    ctx.fillStyle = grad;
    ctx.beginPath();
    if (ctx.roundRect) ctx.roundRect(x, y, barW, bh, [3, 3, 0, 0]);
    else ctx.rect(x, y, barW, bh);
    ctx.fill();

    // Value on top of bar
    ctx.fillStyle = '#fff';
    ctx.font = 'bold 10px Inter, sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText(fmtAxisVal(item.value), x + barW / 2, y - 4);

    // X label
    ctx.fillStyle = '#777';
    ctx.font = '10px Inter, sans-serif';
    const lbl = item.label.length > 10 ? item.label.slice(0, 9) + '…' : item.label;
    ctx.fillText(lbl, x + barW / 2, h - 6);
  });
}

// ── Table renders ─────────────────────────────────────────────────────────────
function renderActiveQueries(queries) {
  const tbody = document.querySelector('#table-active-queries tbody');
  if (!queries.length) {
    tbody.innerHTML = '<tr><td colspan="5" class="empty-cell">No active queries at the moment.</td></tr>';
    return;
  }
  tbody.innerHTML = queries.map(q => `
    <tr>
      <td>${q.pid}</td>
      <td>${escapeHtml(q.user)}</td>
      <td><span class="state-${q.state === 'active' ? 'active' : 'idle'}">${escapeHtml(q.state)}</span></td>
      <td>${escapeHtml(q.duration)}</td>
      <td class="query-text">${escapeHtml(q.query)}</td>
    </tr>`).join('');
}

function renderSlowQueries(queries) {
  const tbody = document.querySelector('#table-slow-queries tbody');
  if (!queries.length) {
    tbody.innerHTML = `<tr><td colspan="5" class="empty-cell">No data.<br>
      <small>PostgreSQL: requires <code>pg_stat_statements</code> extension.<br>
      MySQL: requires <code>performance_schema</code>.</small></td></tr>`;
    return;
  }
  tbody.innerHTML = queries.map(q => `
    <tr>
      <td class="query-text">${escapeHtml(q.query)}</td>
      <td>${fmtNum(q.calls)}</td>
      <td>${q.mean_ms.toFixed(2)}</td>
      <td>${q.total_ms.toFixed(2)}</td>
      <td>${fmtNum(q.rows)}</td>
    </tr>`).join('');
}

function renderTableSizes(tables) {
  const tbody = document.querySelector('#table-sizes tbody');
  if (!tables.length) {
    tbody.innerHTML = '<tr><td colspan="4" class="empty-cell">No tables found.</td></tr>';
    return;
  }
  const maxSize = Math.max(...tables.map(t => t.size_mb || t.row_count), 0.001);
  tbody.innerHTML = tables.map(t => `
    <tr>
      <td>${escapeHtml(t.name)}</td>
      <td style="white-space:nowrap">${formatMB(t.size_mb)}</td>
      <td><div class="size-bar-wrap">
        <div class="size-bar-track">
          <div class="size-bar-fill" style="width:${Math.max(((t.size_mb||t.row_count)/maxSize)*100,2)}%"></div>
        </div></div></td>
      <td>${fmtNum(t.row_count)}</td>
    </tr>`).join('');
}

// ── Sections ──────────────────────────────────────────────────────────────────
function switchSection(name) {
  currentSection = name;
  document.querySelectorAll('.tables-list .table-item').forEach((el, i) => {
    el.classList.toggle('active', ['metrics','queries','slow','tables'][i] === name);
  });
  ['metrics','queries','slow','tables'].forEach(s => {
    document.getElementById('section-' + s).style.display = s === name ? 'block' : 'none';
  });
  const titles = { metrics:'Metrics', queries:'Active Queries', slow:'Query Performance', tables:'System Operations' };
  document.getElementById('section-title').textContent = titles[name];
}

function setRange(btn, range) {
  document.querySelectorAll('.time-btn').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  currentRange = range;
  Object.keys(history).forEach(k => { history[k] = []; });
  loadMetrics();
}

// ── Helpers ───────────────────────────────────────────────────────────────────
function fmtAxisVal(v) {
  if (v == null) return '0';
  if (Math.abs(v) >= 1e6) return (v / 1e6).toFixed(1) + 'M';
  if (Math.abs(v) >= 1e3) return (v / 1e3).toFixed(1) + 'k';
  if (Number.isInteger(v) || Math.abs(v) >= 10) return Math.round(v).toString();
  return v.toFixed(2);
}

function formatMB(mb) {
  if (!mb) return '0 B';
  if (mb < 0.001) return (mb * 1048576).toFixed(0) + ' B';
  if (mb < 1) return (mb * 1024).toFixed(1) + ' KB';
  if (mb < 1024) return mb.toFixed(2) + ' MB';
  return (mb / 1024).toFixed(2) + ' GB';
}

function fmtNum(n) {
  return n == null ? '0' : Number(n).toLocaleString();
}

// ── Boot ──────────────────────────────────────────────────────────────────────
loadMetrics();
setInterval(loadMetrics, 30000);
