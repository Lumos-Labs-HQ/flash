'use strict';



function makeSVGChart(containerId, series, opts = {}) {
  const container = document.getElementById(containerId);
  if (!container) return;
  container.innerHTML = '';

  const W = container.clientWidth || 500;
  const H = opts.height || 200;
  const pad = { top: 16, right: 16, bottom: 36, left: 52 };
  const cw = W - pad.left - pad.right;
  const ch = H - pad.top - pad.bottom;

  const allVals = series.flatMap(s => s.data.map(d => d.value));
  const rawMin = opts.yMin != null ? opts.yMin : Math.min(...allVals, 0);
  const rawMax = opts.yMax != null ? opts.yMax : Math.max(...allVals, 1);
  const yPad = (rawMax - rawMin) * 0.1;
  const yMin = rawMin - yPad;
  const yMax = rawMax + yPad;

  const xLabels = series[0]?.data.map(d => d.label) || [];
  const n = xLabels.length;

  const toX = i => pad.left + (n <= 1 ? cw / 2 : (i / (n - 1)) * cw);
  const toY = v => pad.top + ch - ((v - yMin) / (yMax - yMin)) * ch;

  const ns = 'http://www.w3.org/2000/svg';
  const svg = document.createElementNS(ns, 'svg');
  svg.setAttribute('width', W);
  svg.setAttribute('height', H);
  svg.style.overflow = 'visible';

  // ── Defs (gradients) ────────────────────────────────────────────────────────
  const defs = document.createElementNS(ns, 'defs');
  series.forEach((s, si) => {
    const grad = document.createElementNS(ns, 'linearGradient');
    grad.setAttribute('id', `grad-${containerId}-${si}`);
    grad.setAttribute('x1', '0'); grad.setAttribute('y1', '0');
    grad.setAttribute('x2', '0'); grad.setAttribute('y2', '1');
    const stop1 = document.createElementNS(ns, 'stop');
    stop1.setAttribute('offset', '5%');
    stop1.setAttribute('stop-color', s.color);
    stop1.setAttribute('stop-opacity', '0.4');
    const stop2 = document.createElementNS(ns, 'stop');
    stop2.setAttribute('offset', '95%');
    stop2.setAttribute('stop-color', s.color);
    stop2.setAttribute('stop-opacity', '0.05');
    grad.appendChild(stop1); grad.appendChild(stop2);
    defs.appendChild(grad);
  });
  svg.appendChild(defs);

  // ── Grid lines (horizontal) ─────────────────────────────────────────────────
  const ySteps = 4;
  for (let i = 0; i <= ySteps; i++) {
    const v = yMin + (i / ySteps) * (yMax - yMin);
    const y = toY(v);
    const line = document.createElementNS(ns, 'line');
    line.setAttribute('x1', pad.left); line.setAttribute('x2', pad.left + cw);
    line.setAttribute('y1', y); line.setAttribute('y2', y);
    line.setAttribute('stroke', '#232323'); line.setAttribute('stroke-width', '1');
    svg.appendChild(line);

    // Y label
    const txt = document.createElementNS(ns, 'text');
    txt.setAttribute('x', pad.left - 6);
    txt.setAttribute('y', y + 4);
    txt.setAttribute('text-anchor', 'end');
    txt.setAttribute('fill', '#666');
    txt.setAttribute('font-size', '11');
    txt.textContent = fmtAxisVal(v);
    svg.appendChild(txt);
  }

  // ── X axis labels ───────────────────────────────────────────────────────────
  const xIndices = n <= 6 ? [...Array(n).keys()] :
    [0, Math.floor(n * 0.25), Math.floor(n * 0.5), Math.floor(n * 0.75), n - 1];
  xIndices.forEach(i => {
    const x = toX(i);
    const txt = document.createElementNS(ns, 'text');
    txt.setAttribute('x', x);
    txt.setAttribute('y', H - 8);
    txt.setAttribute('text-anchor', 'middle');
    txt.setAttribute('fill', '#666');
    txt.setAttribute('font-size', '11');
    txt.textContent = xLabels[i] || '';
    svg.appendChild(txt);
  });

  // ── Series (fill + line) ────────────────────────────────────────────────────
  series.forEach((s, si) => {
    if (!s.data.length) return;
    const pts = s.data.map((d, i) => [toX(i), toY(d.value)]);

    // Smooth cubic bezier path
    const linePath = smoothPath(pts);
    const baseY = toY(yMin);

    // Fill area
    const fillD = linePath + ` L${pts[pts.length-1][0]},${baseY} L${pts[0][0]},${baseY} Z`;
    const fill = document.createElementNS(ns, 'path');
    fill.setAttribute('d', fillD);
    fill.setAttribute('fill', `url(#grad-${containerId}-${si})`);
    svg.appendChild(fill);

    // Stroke line
    const line = document.createElementNS(ns, 'path');
    line.setAttribute('d', linePath);
    line.setAttribute('fill', 'none');
    line.setAttribute('stroke', s.color);
    line.setAttribute('stroke-width', '2');
    line.setAttribute('stroke-linejoin', 'round');
    if (s.dashed) line.setAttribute('stroke-dasharray', '5,4');

    // Animate stroke
    const len = line.getTotalLength ? 2000 : 0;
    if (len) {
      line.style.strokeDasharray = len;
      line.style.strokeDashoffset = len;
      line.style.transition = 'stroke-dashoffset 0.8s ease';
      svg.appendChild(line);
      requestAnimationFrame(() => { line.style.strokeDashoffset = 0; });
    } else {
      svg.appendChild(line);
    }

    // Dots on each point
    pts.forEach(([x, y], i) => {
      const circle = document.createElementNS(ns, 'circle');
      circle.setAttribute('cx', x); circle.setAttribute('cy', y);
      circle.setAttribute('r', '3.5');
      circle.setAttribute('fill', s.color);
      circle.setAttribute('stroke', '#0f0f0f');
      circle.setAttribute('stroke-width', '1.5');
      circle.style.opacity = '0';
      circle.style.transition = `opacity 0.3s ease ${0.5 + i * 0.02}s`;
      svg.appendChild(circle);
      requestAnimationFrame(() => { circle.style.opacity = '1'; });
    });
  });

  // ── Tooltip overlay ─────────────────────────────────────────────────────────
  const tooltip = document.createElement('div');
  tooltip.className = 'chart-tooltip';
  tooltip.style.display = 'none';
  container.style.position = 'relative';
  container.appendChild(tooltip);

  svg.addEventListener('mousemove', e => {
    const rect = svg.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    if (mx < pad.left || mx > pad.left + cw) { tooltip.style.display = 'none'; return; }

    const idx = Math.round(((mx - pad.left) / cw) * (n - 1));
    if (idx < 0 || idx >= n) { tooltip.style.display = 'none'; return; }

    const xLabel = xLabels[idx] || '';
    let html = `<div class="tt-label">${xLabel}</div>`;
    series.forEach(s => {
      const v = s.data[idx]?.value ?? 0;
      html += `<div class="tt-row"><span class="tt-dot" style="background:${s.color}"></span>
               <span class="tt-name">${s.label}</span>
               <span class="tt-val">${fmtAxisVal(v)}</span></div>`;
    });
    tooltip.innerHTML = html;
    tooltip.style.display = 'block';

    const tx = toX(idx) + 12;
    const ty = pad.top;
    tooltip.style.left = (tx > W - 140 ? tx - 140 : tx) + 'px';
    tooltip.style.top = ty + 'px';

    // Crosshair
    let ch = svg.querySelector('.crosshair');
    if (!ch) {
      ch = document.createElementNS(ns, 'line');
      ch.setAttribute('class', 'crosshair');
      ch.setAttribute('stroke', '#444'); ch.setAttribute('stroke-width', '1');
      ch.setAttribute('stroke-dasharray', '3,3');
      svg.appendChild(ch);
    }
    ch.setAttribute('x1', toX(idx)); ch.setAttribute('x2', toX(idx));
    ch.setAttribute('y1', pad.top); ch.setAttribute('y2', pad.top + cH);
  });

  svg.addEventListener('mouseleave', () => { tooltip.style.display = 'none'; });

  container.appendChild(svg);
}

// Smooth cubic bezier through points
function smoothPath(pts) {
  if (pts.length === 1) return `M${pts[0][0]},${pts[0][1]}`;
  let d = `M${pts[0][0]},${pts[0][1]}`;
  for (let i = 0; i < pts.length - 1; i++) {
    const [x0, y0] = pts[i];
    const [x1, y1] = pts[i + 1];
    const cpx = (x0 + x1) / 2;
    d += ` C${cpx},${y0} ${cpx},${y1} ${x1},${y1}`;
  }
  return d;
}

function fmtAxisVal(v) {
  if (v == null) return '0';
  const abs = Math.abs(v);
  if (abs >= 1e6) return (v / 1e6).toFixed(1) + 'M';
  if (abs >= 1e3) return (v / 1e3).toFixed(1) + 'k';
  if (!Number.isInteger(v) && abs < 10) return v.toFixed(1);
  return Math.round(v).toString();
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
