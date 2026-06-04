// ===== State =====
const STORAGE_KEY = 'flashorm_studio_state';
const state = {
    currentTable: null, data: null, changes: new Map(),
    page: 1, limit: 50, tablesCache: null, foreignKeys: new Map(),
    filters: [], scrollPosition: 0,
    save() {
        try { sessionStorage.setItem(STORAGE_KEY, JSON.stringify({ currentTable: this.currentTable, page: this.page, limit: this.limit, filters: this.filters, scrollPosition: window.scrollY||0, changes: Array.from(this.changes.entries()) })); } catch(e) {}
    },
    restore() {
        try {
            const s = sessionStorage.getItem(STORAGE_KEY);
            if (!s) return false;
            const p = JSON.parse(s);
            this.currentTable = p.currentTable||null; this.page = p.page||1; this.limit = p.limit||50;
            this.filters = p.filters||[]; this.scrollPosition = p.scrollPosition||0;
            if (p.changes) this.changes = new Map(p.changes);
            return true;
        } catch(e) { return false; }
    },
    clear() { this.changes.clear(); this.filters=[]; sessionStorage.removeItem(STORAGE_KEY); }
};

let currentColumns = [];
let colWidths = {}; // tableName -> { colName -> px }

// ===== Init =====
document.addEventListener('DOMContentLoaded', async () => {
    setupEventListeners();
    await loadTables();
    if (state.restore() && state.currentTable) {
        await selectTable(state.currentTable);
        if (state.filters.length > 0) setTimeout(() => restoreFilters(state.filters), 200);
    }
    window.addEventListener('beforeunload', () => { state.scrollPosition = window.scrollY; state.save(); });
    document.querySelectorAll('a[href]').forEach(l => l.addEventListener('click', () => { state.scrollPosition = window.scrollY; state.save(); }));
});

function setupEventListeners() {
    document.getElementById('delete-selected-btn')?.addEventListener('click', deleteSelected);
    document.getElementById('search-tables')?.addEventListener('input', debounce(filterTables, 200));
    document.addEventListener('keydown', handleGlobalKey);
}

function debounce(fn, ms) { let t; return (...a) => { clearTimeout(t); t = setTimeout(() => fn(...a), ms); }; }

// ===== Tables =====
async function loadTables() {
    const res = await apiCall('/api/tables').catch(() => null);
    if (res?.success) { state.tablesCache = res.data; renderTablesList(res.data); }
}

function renderTablesList(tables) {
    const el = document.getElementById('tables-list');
    if (!tables?.length) { el.innerHTML = '<div style="padding:12px;color:#444;font-size:12px;">No models found</div>'; return; }
    el.innerHTML = tables.map(t => `
        <div class="table-item" data-table="${escapeHtml(t.name)}" onclick="selectTable('${escapeHtml(t.name)}')" title="${escapeHtml(t.name)}">
            <span class="iconify table-item-icon" data-icon="mdi:table"></span>
            <span class="table-item-name">${escapeHtml(t.name)}</span>
            <span class="table-count">${t.row_count}</span>
        </div>`).join('');
}

function filterTables(e) {
    const q = (e?.target?.value || document.getElementById('search-tables')?.value || '').toLowerCase();
    if (!state.tablesCache) return;
    renderTablesList(q ? state.tablesCache.filter(t => t.name.toLowerCase().includes(q)) : state.tablesCache);
}
window.filterTables = filterTables;

async function selectTable(name) {
    closeMobileSidebar();
    state.currentTable = name; state.page = 1; state.changes.clear();
    document.getElementById('current-table').textContent = name;
    document.getElementById('save-btn').style.display = 'none';
    document.querySelectorAll('.table-item').forEach(i => i.classList.toggle('active', i.dataset.table === name));
    showLoadingSkeleton();
    await loadTableData();
    if (state.data?.columns) currentColumns = state.data.columns;
}
window.selectTable = selectTable;

function showLoadingSkeleton() {
    document.getElementById('grid-container').innerHTML = `<div style="padding:0;"><div class="skeleton" style="height:36px;border-radius:0;"></div><div class="skeleton" style="height:calc(100vh - 200px);border-radius:0;margin-top:1px;"></div></div>`;
}

// ===== Data =====
async function loadTableData() {
    if (!state.currentTable) return;
    let url = `/api/tables/${state.currentTable}?page=${state.page}&limit=${state.limit}`;
    if (state.filters?.length) url += `&filters=${encodeURIComponent(JSON.stringify(state.filters))}`;
    const res = await apiCall(url).catch(() => null);
    if (!res?.success) return;
    state.data = res.data;
    const rows = res.data.rows?.length || 0;
    document.getElementById('row-count').textContent = `${rows} of ${res.data.total||0}`;
    if (res.data.columns) {
        const seen = new Set(); currentColumns = [];
        res.data.columns.forEach(c => { if (!seen.has(c.name)) { seen.add(c.name); currentColumns.push(c); } });
    }
    renderDataGrid(res.data);
    updatePagination(res.data);
}

// ===== Grid Render =====
function renderDataGrid(data) {
    const container = document.getElementById('grid-container');
    container.style.display = ''; container.style.alignItems = ''; container.style.justifyContent = '';

    if (!data.rows?.length) {
        if (data.columns?.length) {
            // Show schema as a proper table
            const rows = data.columns.map(col => {
                const badges = [];
                if (col.primary_key) badges.push('<span class="badge badge-primary">PK</span>');
                if (col.foreign_key_table) badges.push(`<span class="badge badge-purple">FK → ${escapeHtml(col.foreign_key_table)}.${escapeHtml(col.foreign_key_column)}</span>`);
                if (!col.nullable && !col.primary_key) badges.push('<span class="badge badge-info">NOT NULL</span>');
                if (col.auto_increment) badges.push('<span class="badge badge-primary">AUTO</span>');
                if (col.default) badges.push(`<span style="font-size:10px;color:#555;">default: ${escapeHtml(col.default)}</span>`);
                return `<tr>
                    <td><div class="td-inner"><span class="v-string">${escapeHtml(col.name)}</span></div></td>
                    <td><div class="td-inner"><span class="v-uuid">${escapeHtml(col.type||'')}</span></div></td>
                    <td><div class="td-inner"><span class="${col.nullable?'v-bool':'v-null'}">${col.nullable?'yes':'no'}</span></div></td>
                    <td><div class="td-inner" style="gap:4px;flex-wrap:wrap;">${badges.join('')}</div></td>
                </tr>`;
            }).join('');
            container.innerHTML = `<div class="grid-scroll-container"><table class="data-table">
                <thead><tr>
                    <th><div class="th-inner"><span class="col-name">Column</span></div></th>
                    <th><div class="th-inner"><span class="col-name">Type</span></div></th>
                    <th><div class="th-inner"><span class="col-name">Nullable</span></div></th>
                    <th><div class="th-inner"><span class="col-name">Constraints</span></div></th>
                </tr></thead>
                <tbody>${rows}</tbody>
            </table></div>`;
        } else {
            container.style.display = 'flex'; container.style.alignItems = 'center'; container.style.justifyContent = 'center';
            container.innerHTML = `<div class="empty-state"><svg fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"></path></svg><div>No rows</div></div>`;
        }
        return;
    }

    const cols = currentColumns;
    cols.forEach(c => { if (c.foreign_key_table) state.foreignKeys.set(c.name, { table: c.foreign_key_table, column: c.foreign_key_column }); });

    const tableKey = state.currentTable;
    if (!colWidths[tableKey]) colWidths[tableKey] = {};
    const cw = colWidths[tableKey];

    const thCells = cols.map(col => {
        const w = cw[col.name] ? `width:${cw[col.name]}px;min-width:${cw[col.name]}px;` : 'min-width:100px;';
        return `<th style="${w}" data-col="${escapeHtml(col.name)}">
            <div class="th-inner">
                <span class="col-name">${escapeHtml(col.name)}</span>
                <span class="col-type">${escapeHtml(col.type||'')}</span>
            </div>
            <div class="col-resize-handle" data-col="${escapeHtml(col.name)}"></div>
        </th>`;
    }).join('');

    const rows = data.rows.map((row, idx) => {
        const pk = getPKValue(row, cols);
        const cells = cols.map(col => {
            const val = row[col.name];
            const fk = state.foreignKeys.get(col.name);
            const formatted = formatValue(val, fk);
            const clickAttr = fk && val != null
                ? `onclick="navigateToForeignKey('${escapeHtml(fk.table)}','${escapeHtml(fk.column)}','${escapeHtml(String(val))}');event.stopPropagation()"`
                : `onclick="openCellEditor(this,'${escapeHtml(String(pk))}','${escapeHtml(col.name)}',event)"`;
            return `<td data-row="${escapeHtml(String(pk))}" data-col="${escapeHtml(col.name)}">
                <div class="td-inner" ${clickAttr}><span class="cell-text">${formatted}</span></div>
            </td>`;
        }).join('');
        return `<tr data-pk="${escapeHtml(String(pk))}">
            <td class="td-checkbox"><div class="td-inner"><input type="checkbox" class="row-checkbox" data-row="${escapeHtml(String(pk))}" onchange="toggleRowSelection(this)"></div></td>
            ${cells}
        </tr>`;
    }).join('');

    container.innerHTML = `<div class="grid-scroll-container" id="grid-scroll">
        <table class="data-table" id="data-table">
            <thead><tr>
                <th class="th-checkbox"><div class="th-inner"><input type="checkbox" id="select-all" onchange="toggleSelectAll(this)"></div></th>
                ${thCells}
            </tr></thead>
            <tbody>${rows}</tbody>
        </table>
    </div>`;

    setupColumnResize();
}

function getPKValue(row, cols) {
    const pkCol = cols.find(c => c.primary_key);
    return pkCol ? row[pkCol.name] : (row.id ?? Object.values(row)[0]);
}

// ===== Column Resize =====
function setupColumnResize() {
    document.querySelectorAll('.col-resize-handle').forEach(handle => {
        let startX, startW, th;
        handle.addEventListener('mousedown', e => {
            e.preventDefault(); e.stopPropagation();
            th = handle.closest('th');
            startX = e.clientX; startW = th.offsetWidth;
            handle.classList.add('resizing');
            const onMove = ev => {
                const w = Math.max(60, startW + (ev.clientX - startX));
                th.style.width = w + 'px'; th.style.minWidth = w + 'px';
                const col = handle.dataset.col;
                if (!colWidths[state.currentTable]) colWidths[state.currentTable] = {};
                colWidths[state.currentTable][col] = w;
            };
            const onUp = () => { handle.classList.remove('resizing'); document.removeEventListener('mousemove', onMove); document.removeEventListener('mouseup', onUp); };
            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup', onUp);
        });
    });
}

// ===== Cell Editor Popup =====
let editorState = { rowId: null, col: null, originalValue: null };

function openCellEditor(tdInner, rowId, col, event) {
    const td = tdInner.closest('td');
    if (!td) return;

    const rawVal = getCellRawValue(rowId, col);
    const colMeta = currentColumns.find(c => c.name === col);
    const isNullable = colMeta ? colMeta.nullable : true;

    editorState = { rowId, col, originalValue: rawVal, td, isNullable };

    const popup = document.getElementById('cell-editor');
    const ta = document.getElementById('cell-editor-ta');
    ta.value = rawVal === null ? '' : String(rawVal);

    // Show/hide Set NULL based on nullable
    const nullBtn = document.getElementById('ced-null');
    nullBtn.style.display = isNullable ? '' : 'none';

    const rect = td.getBoundingClientRect();
    popup.style.display = 'flex';

    let top = rect.bottom + 2;
    let left = rect.left;
    if (top + 200 > window.innerHeight) top = rect.top - 180;
    if (left + 340 > window.innerWidth) left = window.innerWidth - 350;
    if (left < 8) left = 8;
    popup.style.top = top + 'px';
    popup.style.left = left + 'px';

    ta.focus(); ta.select();
    document.querySelectorAll('td.editing').forEach(c => c.classList.remove('editing'));
    td.classList.add('editing');
}
window.openCellEditor = openCellEditor;

function getCellRawValue(rowId, col) {
    if (!state.data?.rows) return null;
    const cols = currentColumns;
    const pkCol = cols.find(c => c.primary_key);
    const row = state.data.rows.find(r => {
        const pk = pkCol ? r[pkCol.name] : (r.id ?? Object.values(r)[0]);
        return String(pk) === String(rowId);
    });
    return row ? (row[col] ?? null) : null;
}

function closeCellEditor() {
    document.getElementById('cell-editor').style.display = 'none';
    document.querySelectorAll('td.editing').forEach(c => c.classList.remove('editing'));
    editorState = { rowId: null, col: null, originalValue: null };
}

function saveCellEditor() {
    const { rowId, col, originalValue, td } = editorState;
    if (!rowId || !col) return;
    const newVal = document.getElementById('cell-editor-ta').value;
    if (newVal !== String(originalValue ?? '')) {
        if (!state.changes.has(rowId)) state.changes.set(rowId, {});
        state.changes.get(rowId)[col] = newVal;
        if (td) td.classList.add('cell-dirty');
        document.getElementById('save-btn').style.display = 'inline-flex';
        // Update display
        if (td) td.querySelector('.cell-text').innerHTML = formatValue(newVal);
        state.save();
    }
    closeCellEditor();
}

function setCellNull() {
    const { rowId, col, originalValue, td } = editorState;
    if (!rowId || !col) return;
    if (originalValue !== null) {
        if (!state.changes.has(rowId)) state.changes.set(rowId, {});
        state.changes.get(rowId)[col] = null;
        if (td) td.classList.add('cell-dirty');
        document.getElementById('save-btn').style.display = 'inline-flex';
        if (td) td.querySelector('.cell-text').innerHTML = formatValue(null);
        state.save();
    }
    closeCellEditor();
}

// Wire popup buttons
document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('ced-save')?.addEventListener('click', saveCellEditor);
    document.getElementById('ced-cancel')?.addEventListener('click', closeCellEditor);
    document.getElementById('ced-null')?.addEventListener('click', setCellNull);
    document.getElementById('cell-editor-ta')?.addEventListener('keydown', e => {
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); saveCellEditor(); }
        if (e.key === 'Escape') { e.preventDefault(); closeCellEditor(); }
    });
    // Close when clicking outside
    document.addEventListener('mousedown', e => {
        const popup = document.getElementById('cell-editor');
        if (popup.style.display !== 'none' && !popup.contains(e.target) && !e.target.closest('td')) {
            closeCellEditor();
        }
    });
});

// ===== Format Values =====
function formatValue(value, fk) {
    if (value === null || value === undefined) return '<span class="v-null">NULL</span>';
    if (typeof value === 'boolean') return `<span class="v-bool">${value}</span>`;
    if (typeof value === 'number') return `<span class="v-number">${value}</span>`;
    if (typeof value === 'object') {
        if (Array.isArray(value) && value.length === 16 && value.every(b => typeof b === 'number' && b >= 0 && b <= 255)) {
            const uuid = bytesToUuid(value);
            return `<span class="v-uuid">${escapeHtml(uuid)}</span>`;
        }
        try { return `<span class="v-json">${escapeHtml(JSON.stringify(value))}</span>`; } catch { return `<span class="v-json">[Object]</span>`; }
    }
    const s = String(value);
    if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s)) return `<span class="v-uuid">${escapeHtml(s)}</span>`;
    if (/^\d{4}-\d{2}-\d{2}(T|\s)\d{2}:\d{2}:\d{2}/.test(s)) return `<span class="v-date">${escapeHtml(s)}</span>`;
    if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s)) return `<span class="v-email">${escapeHtml(s)}</span>`;
    if (/^https?:\/\//.test(s)) return `<a href="${escapeHtml(s)}" target="_blank" class="v-url" onclick="event.stopPropagation()">${escapeHtml(s)}</a>`;
    if (fk) return `<span class="v-fk">${escapeHtml(s)}</span>`;
    if (s === 'true' || s === 'false') return `<span class="v-bool">${escapeHtml(s)}</span>`;
    const truncated = s.length > 120 ? s.slice(0, 120) + '…' : s;
    return `<span class="v-string">${escapeHtml(truncated)}</span>`;
}

function bytesToUuid(bytes) {
    const h = bytes.map(b => b.toString(16).padStart(2,'0')).join('');
    return `${h.slice(0,8)}-${h.slice(8,12)}-${h.slice(12,16)}-${h.slice(16,20)}-${h.slice(20,32)}`;
}

// ===== Selection =====
function toggleSelectAll(cb) {
    document.querySelectorAll('.row-checkbox').forEach(c => { c.checked = cb.checked; toggleRowSelection(c, true); });
    document.getElementById('delete-selected-btn').style.display = cb.checked ? 'inline-flex' : 'none';
}
window.toggleSelectAll = toggleSelectAll;

function toggleRowSelection(cb, skipBtn) {
    cb.closest('tr').classList.toggle('selected', cb.checked);
    if (!skipBtn) {
        const any = !!document.querySelector('.row-checkbox:checked');
        document.getElementById('delete-selected-btn').style.display = any ? 'inline-flex' : 'none';
    }
}
window.toggleRowSelection = toggleRowSelection;

// ===== CRUD =====
async function saveChanges() {
    if (!state.changes.size) return;
    const btn = document.getElementById('save-btn');
    btn.disabled = true;
    const changes = [];
    state.changes.forEach((cols, rowId) => Object.entries(cols).forEach(([col, val]) => changes.push({ row_id: rowId, column: col, value: val, action: 'update' })));
    const res = await apiCall(`/api/tables/${state.currentTable}/save`, { method:'POST', body: JSON.stringify({ changes }) }).catch(() => null);
    if (res?.success) { state.changes.clear(); btn.style.display = 'none'; showToast('Saved', 'success'); refreshData(); }
    else showToast(res?.message || 'Save failed', 'error');
    btn.disabled = false;
}
window.saveChanges = saveChanges;

function showAddRowDialog() { if (state.data?.columns) showAddRowModal(state.data.columns, async data => {
    const res = await apiCall(`/api/tables/${state.currentTable}/add`, { method:'POST', body:JSON.stringify({ data }) }).catch(() => null);
    if (res?.success) { showToast('Row added', 'success'); refreshData(); } else showToast(res?.message || 'Failed', 'error');
}); }
window.showAddRowDialog = showAddRowDialog;

async function deleteSelected() {
    const ids = Array.from(document.querySelectorAll('.row-checkbox:checked')).map(c => c.dataset.row);
    if (!ids.length) return;
    showConfirm('Delete rows', `Delete ${ids.length} row(s)?`, async () => {
        const res = await apiCall(`/api/tables/${state.currentTable}/delete`, { method:'POST', body:JSON.stringify({ row_ids: ids }) }).catch(() => null);
        if (res?.success) { showToast(`Deleted ${ids.length} row(s)`, 'success'); refreshData(); } else showToast(res?.message || 'Failed', 'error');
    });
}
window.deleteSelected = deleteSelected;

function refreshData() {
    if (!state.currentTable) { loadTables(); return; }
    state.changes.clear();
    document.getElementById('save-btn').style.display = 'none';
    document.querySelectorAll('.cell-dirty').forEach(c => c.classList.remove('cell-dirty'));
    if (typeof filters !== 'undefined') filters.length = 0;
    if (typeof updateFilterCount === 'function') updateFilterCount();
    document.getElementById('filter-panel')?.classList.remove('show');
    document.getElementById('filter-btn')?.classList.remove('active');
    loadTableData(); loadTables();
}
window.refreshData = refreshData;

// ===== FK Navigation =====
async function navigateToForeignKey(table, col, value) {
    const res = await apiCall(`/api/tables/${table}?page=1&limit=1000`).catch(() => null);
    if (!res?.success) return;
    const row = res.data.rows?.find(r => String(r[col]) === String(value));
    if (!row) { showToast(`No matching row in ${table}`, 'error'); return; }
    const cols = res.data.columns;
    const html = `<div style="overflow-x:auto;max-height:400px;overflow-y:auto;">
        <table class="data-table" style="width:max-content;min-width:100%;"><thead><tr>
        ${cols.map(c => `<th><div class="th-inner"><span class="col-name">${escapeHtml(c.name)}</span><span class="col-type">${escapeHtml(c.type||'')}</span></div></th>`).join('')}
        </tr></thead><tbody><tr>${cols.map(c => `<td><div class="td-inner"><span class="cell-text">${formatValue(row[c.name])}</span></div></td>`).join('')}</tr></tbody></table>
    </div><div style="margin-top:12px;"><button class="btn btn-primary" onclick="document.querySelectorAll('.custom-modal').forEach(m=>m.remove());selectTable('${escapeHtml(table)}')">Go to ${escapeHtml(table)}</button></div>`;
    showModal(`${table}.${col} = ${value}`, html, 'info', false);
}
window.navigateToForeignKey = navigateToForeignKey;

// ===== Pagination =====
function changePage(d) { state.page = Math.max(1, state.page + d); showLoadingSkeleton(); loadTableData(); }
window.changePage = changePage;

function updatePagination(data) {
    const pg = document.getElementById('pagination');
    if (!data.total) { pg.style.display = 'none'; return; }
    pg.style.display = 'flex';
    const s = (data.page-1)*data.limit+1, e = Math.min(data.page*data.limit, data.total);
    document.getElementById('page-info').textContent = `${s}–${e} of ${data.total}`;
    document.getElementById('prev-btn').disabled = data.page === 1;
    document.getElementById('next-btn').disabled = e >= data.total;
}

// ===== Save Changes =====
async function saveChanges() {
    if (!state.changes.size) return;
    const btn = document.getElementById('save-btn'); btn.disabled = true;
    const changes = [];
    state.changes.forEach((cols, rowId) => Object.entries(cols).forEach(([col, val]) => changes.push({ row_id: rowId, column: col, value: val, action: 'update' })));
    const res = await apiCall(`/api/tables/${state.currentTable}/save`, { method:'POST', body:JSON.stringify({ changes }) }).catch(() => null);
    if (res?.success) { state.changes.clear(); btn.style.display='none'; showToast('Saved', 'success'); refreshData(); }
    else showToast(res?.message || 'Save failed', 'error');
    btn.disabled = false;
}

// ===== Keyboard =====
function handleGlobalKey(e) {
    if (e.key === 'Escape') closeCellEditor();
}

// ===== Modal / Toast helpers (used by index.js too) =====
function showModal(title, content, type='info', blocking=false) {
    document.querySelectorAll('.custom-modal').forEach(m => m.remove());
    const icons = { info:'mdi:information', success:'mdi:check-circle', warning:'mdi:alert', error:'mdi:alert-circle' };
    const colors = { info:'#4a9eff', success:'#4ade80', warning:'#fb923c', error:'#f87171' };
    const m = document.createElement('div');
    m.className = 'custom-modal';
    m.innerHTML = `<div class="custom-modal-content">
        <div class="custom-modal-header">
            <div class="custom-modal-title"><span class="iconify" data-icon="${icons[type]||icons.info}" style="color:${colors[type]||colors.info}"></span>${escapeHtml(title)}</div>
            <button class="custom-modal-close" onclick="this.closest('.custom-modal').remove()">×</button>
        </div>
        <div class="custom-modal-body">${content}</div>
    </div>`;
    document.body.appendChild(m);
    setTimeout(() => m.classList.add('show'), 10);
    if (!blocking) { m.addEventListener('click', e => { if (e.target===m) m.remove(); }); }
    const esc = e => { if (e.key==='Escape') { m.remove(); document.removeEventListener('keydown',esc); } };
    document.addEventListener('keydown', esc);
}

function showConfirm(title, content, onConfirm) {
    showModal(title, content + `<div class="custom-modal-footer" style="margin-top:16px;">
        <button class="btn btn-secondary" onclick="this.closest('.custom-modal').remove()">Cancel</button>
        <button class="btn btn-primary" id="confirm-ok">Confirm</button>
    </div>`, 'warning', true);
    setTimeout(() => {
        document.getElementById('confirm-ok')?.addEventListener('click', () => {
            document.querySelectorAll('.custom-modal').forEach(m => m.remove());
            onConfirm();
        });
    }, 20);
}

// ===== Sidebar =====
function toggleMobileSidebar() {
    document.getElementById('sidebar').classList.toggle('mobile-open');
    document.getElementById('sidebar-backdrop').classList.toggle('show');
}
function closeMobileSidebar() {
    document.getElementById('sidebar').classList.remove('mobile-open');
    document.getElementById('sidebar-backdrop').classList.remove('show');
}
window.toggleMobileSidebar = toggleMobileSidebar;
window.closeMobileSidebar = closeMobileSidebar;

function showCreateTableForm() { window.location.href = '/schema#create-table'; }
window.showCreateTableForm = showCreateTableForm;

// ===== Branch =====
async function loadBranches() {
    const res = await apiCall('/api/branches').catch(() => null);
    if (!res) return;
    const sel = document.getElementById('branch-selector');
    if (res.branches?.length <= 1) { sel.style.display='none'; return; }
    sel.style.display='inline-block';
    sel.innerHTML = res.branches.map(b => `<option value="${escapeHtml(b.name)}" ${b.name===res.current?'selected':''}>${escapeHtml(b.name)}${b.is_default?' (default)':''}</option>`).join('');
}
async function switchBranch(name) {
    if (!name) return;
    const res = await apiCall('/api/branches/switch', { method:'POST', body:JSON.stringify({ branch: name }) }).catch(() => null);
    if (res?.success) { showToast(`Switched to ${name}`, 'success'); location.reload(); }
    else showToast('Failed to switch branch', 'error');
}
window.switchBranch = switchBranch;
document.addEventListener('DOMContentLoaded', loadBranches);

// ===== Dropdown =====
function toggleDropdown(id) {
    const d = document.getElementById(id);
    document.querySelectorAll('.dropdown-menu').forEach(x => { if (x.id!==id) x.classList.remove('show'); });
    d.classList.toggle('show');
}
document.addEventListener('click', e => { if (!e.target.closest('.dropdown')) document.querySelectorAll('.dropdown-menu').forEach(d => d.classList.remove('show')); });
window.toggleDropdown = toggleDropdown;

// ===== Op overlay =====
function showOpOverlay(title, status, isImport) {
    document.getElementById('op-title').textContent = title;
    document.getElementById('op-icon').textContent = isImport ? '📥' : '📤';
    setOpStatus(status);
    const bar = document.getElementById('op-bar');
    bar.className = 'op-progress-bar indeterminate' + (isImport?' import':'');
    bar.style.width = '';
    document.getElementById('op-overlay').classList.add('show');
}
function setOpStatus(msg) { document.getElementById('op-status').textContent = msg; }
function setOpProgress(pct) { const b=document.getElementById('op-bar'); b.classList.remove('indeterminate'); b.style.width=Math.min(100,Math.round(pct))+'%'; }
function hideOpOverlay() { document.getElementById('op-overlay').classList.remove('show'); }
window.showOpOverlay = showOpOverlay; window.setOpStatus = setOpStatus; window.setOpProgress = setOpProgress; window.hideOpOverlay = hideOpOverlay;

// ===== Context menu =====
(function() {
    let menu = null, ctxTable = null;
    function get() {
        if (!menu) {
            menu = document.createElement('div'); menu.className='context-menu';
            menu.innerHTML=`<button class="context-menu-item" data-a="edit"><span class="iconify" data-icon="mdi:pencil"></span>Edit Schema</button><div class="context-menu-divider"></div><button class="context-menu-item context-menu-item-danger" data-a="delete"><span class="iconify" data-icon="mdi:delete"></span>Drop Table</button>`;
            menu.addEventListener('click', e => { const b=e.target.closest('[data-a]'); if(!b) return; hide(); if(b.dataset.a==='edit') window.location.href='/schema#edit-'+encodeURIComponent(ctxTable); else if(b.dataset.a==='delete') dropTable(ctxTable); });
            document.body.appendChild(menu);
        }
        return menu;
    }
    function show(e, name) { e.preventDefault(); e.stopPropagation(); ctxTable=name; const m=get(); m.style.cssText=`display:block;left:${e.clientX}px;top:${e.clientY}px`; requestAnimationFrame(()=>{ const r=m.getBoundingClientRect(); if(r.right>window.innerWidth) m.style.left=(e.clientX-r.width)+'px'; if(r.bottom>window.innerHeight) m.style.top=(e.clientY-r.height)+'px'; }); }
    function hide() { if(menu) menu.style.display='none'; }
    document.addEventListener('DOMContentLoaded', () => { document.getElementById('tables-list').addEventListener('contextmenu', e => { const ti=e.target.closest('.table-item'); if(ti) show(e, ti.dataset.table); }); });
    document.addEventListener('click', hide); document.addEventListener('scroll', hide, true);
    async function dropTable(name) {
        showConfirm('Drop Table', `<p>Drop <strong>${escapeHtml(name)}</strong>? <span style="color:#f87171">All data will be lost.</span></p>`, async () => {
            const res = await apiCall('/api/schema/apply', { method:'POST', body:JSON.stringify({ type:'drop_table', table:name }) }).catch(()=>null);
            if (res?.success) { showToast(`Table "${name}" dropped`, 'success'); if(state.currentTable===name) { state.currentTable=null; document.getElementById('current-table').textContent='Select a model'; document.getElementById('grid-container').innerHTML=''; } loadTables(); }
            else showToast(res?.message||'Failed', 'error');
        });
    }
})();

// ===== SQL modal =====
function openSQLModal() { document.getElementById('sql-modal').classList.add('show'); }
function closeSQLModal() { document.getElementById('sql-modal').classList.remove('show'); }
window.closeSQLModal = closeSQLModal;
async function executeSQLQuery() {
    const q = document.getElementById('sql-query').value.trim();
    if (!q) return;
    const res = await apiCall('/api/sql', { method:'POST', body:JSON.stringify({ query:q }) }).catch(()=>null);
    if (res?.success) { state.data=res.data; renderDataGrid(res.data); closeSQLModal(); }
    else showToast(res?.message||'Query failed', 'error');
}
window.executeSQLQuery = executeSQLQuery;

// ===== cell-dirty style =====
const style = document.createElement('style');
style.textContent = `
.cell-dirty .td-inner { background: #1a2a1a !important; }
.cell-dirty .td-inner::after { content:''; position:absolute;top:2px;right:2px;width:5px;height:5px;background:#4ade80;border-radius:50%; }
td.editing .td-inner { background: #1a2535 !important; outline: 1px solid #4a9eff; }
`;
document.head.appendChild(style);
