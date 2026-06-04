// index.js — filter system, export/import only
// All other functionality is in studio.js

let filters = [];

function restoreFilters(savedFilters) {
    if (!savedFilters?.length) return;
    document.getElementById('filter-rows').innerHTML = '';
    savedFilters.forEach((f, i) => addFilterRow(i===0?'where':f.logic, f.column, f.operator, f.value));
    filters = savedFilters;
    updateFilterCount();
}

function toggleFilters() {
    document.getElementById('filter-panel').classList.toggle('show');
    document.getElementById('filter-btn').classList.toggle('active');
}
window.toggleFilters = toggleFilters;

function getColumnType(colName) {
    const col = currentColumns.find(c => c.name === colName);
    const t = (col?.type||'').toLowerCase();
    if (t.includes('int')||t.includes('serial')||t.includes('decimal')||t.includes('numeric')||t.includes('float')||t.includes('double')||t.includes('real')) return 'number';
    if (t.includes('bool')) return 'boolean';
    if (t.includes('date')||t.includes('time')||t.includes('timestamp')) return 'datetime';
    if (t.includes('uuid')) return 'uuid';
    if (t.includes('json')) return 'json';
    return 'text';
}

function addFilterRow(logic='where', column='', operator='equals', value='') {
    const row = document.createElement('div');
    row.className = 'filter-row';
    const logicHtml = logic==='where'
        ? `<select class="filter-logic" disabled><option>where</option></select>`
        : `<select class="filter-logic"><option value="and" ${logic==='and'?'selected':''}>and</option><option value="or" ${logic==='or'?'selected':''}>or</option></select>`;
    const colOpts = currentColumns.map(c => `<option value="${escapeHtml(c.name)}" ${c.name===column?'selected':''}>${escapeHtml(c.name)} (${escapeHtml(c.type||'')})</option>`).join('');
    row.innerHTML = `${logicHtml}
        <select class="filter-column" onchange="updateFilterOperators(this)">${colOpts}</select>
        <select class="filter-operator">
            ${['equals','not_equals','contains','not_contains','starts_with','ends_with','gt','lt','gte','lte','is_null','is_not_null','is_empty','is_not_empty'].map(o=>`<option value="${o}" ${o===operator?'selected':''}>${o.replace(/_/g,' ')}</option>`).join('')}
        </select>
        <input type="text" class="filter-value" value="${escapeHtmlAttr(value)}" placeholder="Value">
        <button class="filter-remove" onclick="this.parentElement.remove();updateFilterCount()">✕</button>`;
    document.getElementById('filter-rows').appendChild(row);
    updateFilterCount();
    updateFilterOperators(row.querySelector('.filter-column'));
}
window.addFilterRow = addFilterRow;

function updateFilterOperators(sel) {
    const vi = sel.parentElement.querySelector('.filter-value');
    const op = sel.parentElement.querySelector('.filter-operator');
    op.addEventListener('change', function() {
        const noVal = ['is_null','is_not_null','is_empty','is_not_empty'].includes(this.value);
        vi.disabled = noVal; vi.value = noVal ? '' : vi.value; vi.placeholder = noVal ? 'N/A' : 'Value';
    });
}
window.updateFilterOperators = updateFilterOperators;

function updateFilterCount() {
    const n = document.getElementById('filter-rows').children.length;
    const b = document.getElementById('filter-count');
    b.textContent = n; b.style.display = n > 0 ? 'block' : 'none';
}
window.updateFilterCount = updateFilterCount;

function clearFilters() {
    document.getElementById('filter-rows').innerHTML = '';
    updateFilterCount(); filters = [];
    if (typeof state !== 'undefined') { state.filters=[]; state.page=1; state.save?.(); }
    if (typeof loadTableData === 'function') { showLoadingSkeleton(); loadTableData(); }
}
window.clearFilters = clearFilters;

function applyFilters() {
    const rows = document.getElementById('filter-rows').children;
    filters = [];
    for (const row of rows) {
        const logic = row.querySelector('.filter-logic')?.value || 'where';
        const column = row.querySelector('.filter-column').value;
        const operator = row.querySelector('.filter-operator').value;
        const value = row.querySelector('.filter-value').value;
        const noVal = ['is_null','is_not_null','is_empty','is_not_empty'].includes(operator);
        if (column && (noVal || value !== '')) filters.push({ logic, column, operator, value: noVal?'':value });
    }
    toggleFilters();
    if (typeof state !== 'undefined') { state.filters=filters; state.page=1; state.save?.(); }
    if (typeof loadTableData === 'function') { showLoadingSkeleton(); loadTableData(); }
}
window.applyFilters = applyFilters;

// ===== Export =====
async function exportDatabase(exportType) {
    document.querySelectorAll('.dropdown-menu').forEach(d => d.classList.remove('show'));
    const labels = { schema_only:'Schema', data_only:'Data', complete:'Schema + Data' };
    showOpOverlay(`Exporting ${labels[exportType]||exportType}`, 'Connecting…', false);
    try {
        setOpStatus('Fetching data…');
        const resp = await fetch(`/api/export/${exportType}`);
        setOpProgress(80);
        const json = await resp.json();
        if (json.success === false) { showToast(json.message, 'error'); return; }
        const data = json.data || json;
        setOpProgress(95); setOpStatus('Preparing download…');
        const blob = new Blob([JSON.stringify(data, null, 2)], { type:'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a'); a.href=url;
        a.download = `database_export_${exportType}_${new Date().toISOString().replace(/[:.]/g,'-').slice(0,19)}.json`;
        document.body.appendChild(a); a.click(); document.body.removeChild(a); URL.revokeObjectURL(url);
        setOpProgress(100);
        showToast(`Export complete — ${data.tables?.length||0} tables`, 'success');
    } catch(e) { showToast('Export failed: '+e.message, 'error'); }
    finally { setTimeout(hideOpOverlay, 600); }
}
window.exportDatabase = exportDatabase;

function triggerImport() { document.getElementById('import-file-input').click(); }
window.triggerImport = triggerImport;

async function handleImportFile(event) {
    const file = event.target.files[0]; if (!file) return;
    event.target.value = '';
    let data; try { data = JSON.parse(await file.text()); } catch { showToast('Invalid JSON', 'error'); return; }
    if (data.success !== undefined && data.data) data = data.data;
    if (!data.version || !data.tables) { showToast('Invalid export format', 'error'); return; }
    const tc = data.tables.length, ec = data.enum_types?.length||0;
    const totalRows = data.tables.reduce((s,t)=>s+(t.data?.length||0),0);
    const details = `<p><strong>File:</strong> ${escapeHtml(file.name)}</p>
        <p><strong>Provider:</strong> ${escapeHtml(data.database_provider||'?')}</p>
        ${ec?`<p><strong>Enum types:</strong> ${ec}</p>`:''}
        <p><strong>Tables:</strong> ${tc}</p><p><strong>Rows:</strong> ${totalRows}</p>`;
    showConfirm('Import Database', details, () => performImport(data));
}
window.handleImportFile = handleImportFile;

async function performImport(data) {
    const tc = data.tables?.length||0, tr = data.tables?.reduce((s,t)=>s+(t.data?.length||0),0)||0;
    showOpOverlay('Importing', `${tc} tables · ${tr} rows…`, true);
    try {
        setOpStatus('Sending to server…');
        const res = await apiCall('/api/import', { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(data) });
        setOpProgress(100);
        const r = res.result;
        const lines = [];
        if (r.enum_types_created?.length) lines.push(`Enums: ${r.enum_types_created.join(', ')}`);
        if (r.tables_created?.length) lines.push(`Tables created: ${r.tables_created.length}`);
        if (r.rows_inserted) lines.push(`Rows inserted: ${r.rows_inserted}`);
        if (r.errors?.length) lines.push(`<span style="color:#f87171">Errors: ${r.errors.length}</span>`);
        showModal('Import Complete', `<ul style="padding-left:18px;line-height:1.8;">${lines.map(l=>`<li>${l}</li>`).join('')}</ul>`, r.errors?.length?'warning':'success');
        if (typeof refreshData === 'function') refreshData();
        if (typeof loadTables === 'function') loadTables();
    } catch(e) { showToast('Import failed: '+e.message, 'error'); }
    finally { setTimeout(hideOpOverlay, 600); }
}
