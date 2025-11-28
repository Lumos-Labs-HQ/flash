function $(sel) { return document.querySelector(sel) }
function $$(sel) { return document.querySelectorAll(sel) }

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function extractHostnameFromURL(url) {
    try {
        const match = url.match(/mongodb(?:\+srv)?:\/\/(?:[^:]+:[^@]+@)?([^/?]+)/);
        return match ? match[1].split(':')[0] : 'mongodb';
    } catch (e) {
        return 'mongodb';
    }
}

// State variables
let currentDatabase = '', currentCollection = '', currentFilter = '', databases = [], collections = [], documents = [], selected = new Set(), page = 1, pageSize = 20, total = 0, viewMode = 'json';
let dbConnectionString = extractHostnameFromURL(window.DB_CONNECTION_URL || 'mongodb://localhost');

document.addEventListener('DOMContentLoaded', init);

async function init() {
    setupListeners();
    await loadDatabases();
}

function setupListeners() {
    const refreshBtn = $('#refresh-docs-btn');
    if (refreshBtn) refreshBtn.onclick = () => {
        loadDocs();
    };
    const backBtn = $('#back-btn');
    if (backBtn) backBtn.onclick = () => { currentDatabase = ''; currentCollection = ''; showDatabasesPanel() };
    const selectAllBtn = $('#select-all-btn');
    if (selectAllBtn) selectAllBtn.onclick = toggleSelectAll;
    const filterBtn = $('#filter-btn');
    if (filterBtn) filterBtn.onclick = () => { loadFilterSchema(); openModal('filter-modal'); };
    const insertBtn = $('#insert-btn');
    if (insertBtn) insertBtn.onclick = () => { $('#doc-id').value = ''; $('#doc-json').value = '{}'; openModal('doc-modal') };
    const deleteBtn = $('#delete-btn');
    if (deleteBtn) deleteBtn.onclick = bulkDelete;
    const saveBtn = $('#save-btn');
    if (saveBtn) saveBtn.onclick = saveDoc;
    const prevBtn = $('#prev-page');
    const nextBtn = $('#next-page');
    if (prevBtn) prevBtn.onclick = () => goToPage(page - 1);
    if (nextBtn) nextBtn.onclick = () => goToPage(page + 1);
    const pageSizeEl = $('#page-size');
    if (pageSizeEl) pageSizeEl.onchange = (e) => { pageSize = +e.target.value; page = 1; loadDocs() };
    const viewModeEl = $('#view-mode');
    if (viewModeEl) viewModeEl.onchange = (e) => { viewMode = e.target.value; renderDocs() };

    // Text search - search entire collection via MongoDB
    const textSearch = $('#text-search');
    if (textSearch) {
        textSearch.onkeypress = (e) => {
            if (e.key === 'Enter') {
                const query = e.target.value.trim();
                if (query) {
                    const searchFilter = JSON.stringify({
                        $or: Object.keys(documents.length > 0 ? documents[0] : {}).map(key => ({
                            [key]: { $regex: query, $options: 'i' }
                        }))
                    });
                    page = 1;
                    loadDocs(searchFilter);
                } else {
                    loadDocs(currentFilter);
                }
            }
        };
    }

    $$('.modal-close').forEach(el => el.onclick = () => closeModals());
    $$('.modal-backdrop').forEach(el => el.onclick = () => closeModals());

    // Refresh collections button
    const refreshCollBtn = $('#refresh-collections-btn');
    if (refreshCollBtn) refreshCollBtn.onclick = () => loadCollections();

    // Forms
    const filterForm = $('#filter-form');
    if (filterForm) filterForm.onsubmit = (e) => { e.preventDefault(); applyFilter(); closeModals() };
    const collForm = $('#collection-form');
    if (collForm) collForm.onsubmit = (e) => { e.preventDefault(); createCollection($('#collection-name').value.trim()) };
    const addCollBtn = $('#add-collection-btn');
    if (addCollBtn) addCollBtn.onclick = () => openModal('collection-modal');
    const dbForm = $('#database-form');
    if (dbForm) dbForm.onsubmit = (e) => { e.preventDefault(); createDatabase($('#database-name').value.trim()) };

    $$('.tab').forEach(tab => tab.onclick = function () {
        $$('.tab').forEach(t => t.classList.remove('active'));
        this.classList.add('active');
        const tabName = this.dataset.tab;
        hideAllViews();
        if (tabName === 'documents') {
            $('#toolbar').style.display = 'flex';
            $('#table-view').style.display = viewMode === 'table' ? 'block' : 'none';
            $('#json-view').style.display = viewMode === 'json' ? 'block' : 'none';
        } else if (tabName === 'schema') {
            $('#schema-view').style.display = 'block';
            loadSchema();
        } else if (tabName === 'indexes') {
            $('#indexes-view').style.display = 'block';
            loadIndexes();
        } else if (tabName === 'aggregation') {
            $('#aggregation-view').style.display = 'block';
            loadAggregation();
        }
    });

    $('#create-index-btn').onclick = () => openModal('index-modal');
    $('#index-form').onsubmit = (e) => { e.preventDefault(); createIndex() };

    initAggregation();
}

async function loadFilterSchema() {
    if (!currentCollection) return;
    try {
        const params = new URLSearchParams({ database: currentDatabase, page: 1, limit: 50 });
        const res = await fetch(`/api/collections/${currentCollection}/documents?${params}`);
        const data = await res.json();
        const docs = data.success && data.data ? data.data.documents : [];

        if (docs.length > 0) {
            const fields = new Set();
            docs.forEach(doc => Object.keys(doc).forEach(k => fields.add(k)));
            const fieldList = Array.from(fields).sort();

            const container = $('#filter-schema-fields');
            if (container) {
                container.innerHTML = '<small style="color: var(--text-tertiary); display: block; margin-bottom: 6px;">Click field to add to filter:</small>' +
                    '<div class="field-chips">' +
                    fieldList.map(f => `<button type="button" class="chip" onclick="addFieldToFilter('${f}')">${f}</button>`).join('') +
                    '</div>';
            }
        }
    } catch (err) {
        console.error('Failed to load schema:', err);
    }
}

function hideAllViews() {
    $('#toolbar').style.display = 'none';
    $('#table-view').style.display = 'none';
    $('#json-view').style.display = 'none';
    $('#schema-view').style.display = 'none';
    $('#indexes-view').style.display = 'none';
    $('#aggregation-view').style.display = 'none';
}

function showLoading(container) {
    if (typeof container === 'string') container = $(container);
    if (container) {
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    }
}

function showContentLoading() {
    const jsonView = $('#json-view');
    const tableBody = $('#docs-table tbody');
    if (jsonView) jsonView.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    if (tableBody) tableBody.innerHTML = '<tr><td colspan="100%" style="text-align:center;padding:40px;"><div class="loading"><div class="spinner"></div></div></td></tr>';
}

async function loadDatabases() {
    showLoading('#databases-list');
    try {
        const res = await fetch('/api/databases');
        const data = await res.json();
        console.log('Databases response:', data);
        databases = data.success ? data.data : data.data || data || [];
        renderDatabases();
    } catch (err) {
        console.error('Database load error:', err);
        showError('Failed to load databases: ' + err.message);
        $('#databases-list').innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-tertiary)">Failed to load</div>';
    }
}

function showDatabasesPanel() {
    $('#databases-panel').style.display = 'flex';
    $('#collections-panel').style.display = 'none';
    $('#breadcrumb').textContent = 'Select database';
    $('#empty-title').textContent = 'Select a Database';
    $('#empty-text').textContent = 'Choose a database to begin';
    $('#empty-state').style.display = 'flex';
    hideAllViews();
    $('#tabs-bar').style.display = 'none';
}

function showCollectionsPanel() {
    $('#databases-panel').style.display = 'none';
    $('#collections-panel').style.display = 'flex';
    $('#db-title').textContent = currentDatabase;
}

async function selectDatabase(name) {
    currentDatabase = name;
    currentCollection = '';
    showCollectionsPanel();
    await loadCollections();
    $('#breadcrumb').textContent = `${dbConnectionString} > ${name}`;
    $('#empty-title').textContent = 'Select a Collection';
    $('#empty-text').textContent = 'Choose a collection to view documents';
    $('#empty-state').style.display = 'flex';
    hideAllViews();
}

async function loadCollections() {
    if (!currentDatabase) {
        collections = [];
        renderCollections();
        return;
    }
    showLoading('#collections-list');
    try {
        const res = await fetch('/api/collections?database=' + encodeURIComponent(currentDatabase));
        const data = await res.json();
        console.log('Collections response:', data);
        collections = data.success ? data.data : data.data || data || [];
        renderCollections();
    } catch (err) {
        console.error('Collections load error:', err);
        showError('Failed to load collections: ' + err.message);
        $('#collections-list').innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-tertiary)">Failed to load</div>';
    }
}

async function selectCollection(name) {
    currentCollection = name;
    currentFilter = ''; 
    page = 1;
    selected.clear();
    const textSearch = $('#text-search');
    if (textSearch) textSearch.value = '';
    $$('.collection-item').forEach(el => el.classList.remove('active'));
    event.currentTarget.classList.add('active');
    $('#breadcrumb').textContent = `${dbConnectionString} > ${currentDatabase} > ${name}`;
    $('#tabs-bar').style.display = 'flex';
    $('#toolbar').style.display = 'flex';
    $$('.tab').forEach(t => t.classList.remove('active'));
    $$('.tab[data-tab="documents"]')[0].classList.add('active');
    await loadDocs();
}

async function loadDocs(filter) {
    if (!currentCollection) return;
    const actualFilter = filter !== undefined ? filter : currentFilter;
    
    showContentLoading();
    $('#empty-state').style.display = 'none';
    $('#toolbar').style.display = 'flex';
    if (viewMode === 'table') {
        $('#table-view').style.display = 'block';
        $('#json-view').style.display = 'none';
    } else {
        $('#table-view').style.display = 'none';
        $('#json-view').style.display = 'block';
    }
    
    try {
        const params = new URLSearchParams({ page: page, limit: pageSize });
        if (actualFilter) params.append('filter', actualFilter);
        if (currentDatabase) params.append('database', currentDatabase);
        console.log('Fetching documents with params:', { page, pageSize, database: currentDatabase, collection: currentCollection, filter: actualFilter });
        const res = await fetch(`/api/collections/${currentCollection}/documents?${params}`);
        const data = await res.json();
        console.log('Raw API response:', JSON.stringify(data, null, 2));

        let result = data;
        if (data.success && data.data) {
            result = data.data;
        } else if (data.data) {
            result = data.data;
        }

        console.log('Extracted result:', result);
        documents = result.documents || result.Documents || [];
        total = result.total_count || result.TotalCount || result.total || 0;
        console.log('Final documents:', documents.length, 'Total:', total);
        renderDocs();
        updatePagination();
    } catch (err) {
        console.error('Documents load error:', err);
        showError('Failed to load documents: ' + err.message);
        $('#json-view').innerHTML = '<div style="padding:40px;text-align:center;color:var(--text-tertiary)">Failed to load documents</div>';
    }
}

function applyFilter() {
    const filterQuery = $('#filter-query').value.trim();
    currentFilter = filterQuery;
    page = 1;
    const textSearch = $('#text-search');
    if (textSearch) textSearch.value = '';
    loadDocs(currentFilter);
}

function filterDisplayedDocuments(query) {
    if (!query) {
        $$('.json-card').forEach(card => card.style.display = '');
        $$('.data-table tbody tr').forEach(row => row.style.display = '');
        return;
    }
    $$('.json-card').forEach(card => {
        const text = card.textContent.toLowerCase();
        card.style.display = text.includes(query) ? '' : 'none';
    });
    $$('.data-table tbody tr').forEach(row => {
        const text = row.textContent.toLowerCase();
        row.style.display = text.includes(query) ? '' : 'none';
    });
}

function setFilterExample(example) {
    $('#filter-query').value = example;
}

function addFieldToFilter(fieldName) {
    const textarea = $('#filter-query');
    const current = textarea.value.trim();
    let newValue;

    if (!current || current === '{}') {
        newValue = `{"${fieldName}": "value"}`;
    } else {
        try {
            const obj = JSON.parse(current);
            obj[fieldName] = "value";
            newValue = JSON.stringify(obj, null, 2);
        } catch (e) {
            newValue = current;
        }
    }

    textarea.value = newValue;
    textarea.focus();
}

function renderDocs() {
    const empty = $('#empty-state');
    const toolbar = $('#toolbar');
    const tableView = $('#table-view');
    const jsonView = $('#json-view');

    if (!documents.length) {
        empty.style.display = 'flex';
        toolbar.style.display = 'none';
        tableView.style.display = 'none';
        jsonView.style.display = 'none';
        $('#empty-title').textContent = 'No Documents';
        $('#empty-text').innerHTML = 'This collection is empty<br><button class="btn btn-primary" style="margin-top: 16px;" id="empty-add-btn"><ion-icon name="add-outline"></ion-icon> Add Document</button>';
        const emptyAddBtn = $('#empty-add-btn');
        if (emptyAddBtn) {
            emptyAddBtn.onclick = () => {
                $('#doc-id').value = '';
                $('#doc-json').value = '{}';
                openModal('doc-modal');
            };
        }
        return;
    }

    empty.style.display = 'none';
    toolbar.style.display = 'flex';

    if (viewMode === 'table') {
        tableView.style.display = 'block';
        jsonView.style.display = 'none';
        renderTableView();
    } else {
        tableView.style.display = 'none';
        jsonView.style.display = 'block';
        renderJSONView();
    }
}

function renderTableView() {
    const keys = new Set();
    documents.forEach(doc => Object.keys(doc).forEach(k => keys.add(k)));
    const cols = Array.from(keys);

    const thead = $('#docs-table thead tr');
    thead.innerHTML = `<th width="40"><input type="checkbox" id="select-all-table"></th>${cols.map(k => `<th>${k}</th>`).join('')}<th width="100">Actions</th>`;
    $('#select-all-table').onchange = toggleSelectAll;

    const tbody = $('#docs-table tbody');
    tbody.innerHTML = documents.map(doc => {
        const id = doc._id || doc.id || '';
        const sel = selected.has(id);
        return `<tr class="${sel ? 'selected' : ''}">
      <td><input type="checkbox" class="row-check" data-id="${id}" ${sel ? 'checked' : ''}></td>
      ${cols.map(k => `<td title="${escapeHtml(JSON.stringify(doc[k]))}">${formatValue(doc[k])}</td>`).join('')}
      <td class="row-actions">
        <button class="action-btn edit" onclick="editDoc('${id}')" title="Edit"><svg width="14" height="14" fill="currentColor"><path d="M11 1l3 3L4 14H1v-3z"/></svg></button>
        <button class="action-btn delete" onclick="deleteDoc('${id}')" title="Delete"><svg width="14" height="14" fill="currentColor"><path d="M5 1h4v1H5zm-2 2h8v10H3zm2 1v8h1V4zm2 0v8h1V4zm2 0v8h1V4z"/></svg></button>
      </td>
    </tr>`;
    }).join('');

    $$('.row-check').forEach(cb => cb.onchange = toggleRow);
    updateBulkActions();
}

function renderJSONView() {
    const container = $('#json-view');
    container.innerHTML = documents.map(doc => {
        const id = doc._id || doc.id || '';
        const sel = selected.has(id);
        return `<div class="json-card ${sel ? 'selected' : ''}">
      <div class="json-card-header">
        <div style="display:flex;align-items:center;gap:6px;">
          <input type="checkbox" class="row-check" data-id="${id}" ${sel ? 'checked' : ''}>
          <span class="json-card-id">${id}</span>
        </div>
        <div class="json-card-actions">
          <button class="copy-btn" onclick="copyDocToClipboard(this, '${id}')" title="Copy as JSON"><svg width="14" height="14" fill="currentColor" viewBox="0 0 16 16"><path d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"/><path d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"/></svg></button>
          <button class="action-btn edit" onclick="editDoc('${id}')" title="Edit"><svg width="14" height="14" fill="currentColor"><path d="M11 1l3 3L4 14H1v-3z"/></svg></button>
          <button class="action-btn delete" onclick="deleteDoc('${id}')" title="Delete"><svg width="14" height="14" fill="currentColor"><path d="M5 1h4v1H5zm-2 2h8v10H3zm2 1v8h1V4zm2 0v8h1V4zm2 0v8h1V4z"/></svg></button>
        </div>
      </div>
      <div class="json-card-body">
        ${renderJSONTree(doc, 0)}
      </div>
    </div>`;
    }).join('');

    $$('.row-check').forEach(cb => cb.onchange = toggleRow);
    updateBulkActions();
    
    // Setup toggle buttons - json-expand starts hidden via CSS
    $$('.json-toggle').forEach(btn => {
        btn.onclick = function (e) {
            e.stopPropagation();
            const expandEl = this.parentElement.nextElementSibling;
            if (expandEl && expandEl.classList.contains('json-expand')) {
                const isExpanded = expandEl.style.display === 'block';
                expandEl.style.display = isExpanded ? 'none' : 'block';
                this.textContent = isExpanded ? '▶' : '▼';
            }
        };
    });
}

function copyDocToClipboard(btn, id) {
    const doc = documents.find(d => (d._id || d.id) === id);
    if (!doc) return;
    
    const jsonStr = JSON.stringify(doc, null, 2);
    navigator.clipboard.writeText(jsonStr).then(() => {
        btn.classList.add('copied');
        showSuccess('Copied to clipboard');
        setTimeout(() => btn.classList.remove('copied'), 2000);
    }).catch(err => {
        showError('Failed to copy: ' + err.message);
    });
}

function renderJSONTree(obj, depth) {
    if (obj === null || obj === undefined) return `<span class="json-null">null</span>`;

    const type = typeof obj;
    if (type === 'string') return `<span class="json-string">"${escapeHtml(obj)}"</span>`;
    if (type === 'number') return `<span class="json-number">${obj}</span>`;
    if (type === 'boolean') return `<span class="json-boolean">${obj}</span>`;

    const isArray = Array.isArray(obj);
    const entries = isArray ? obj.map((v, i) => [i, v]) : Object.entries(obj);

    if (!entries.length) {
        return isArray ? '<span class="json-empty">[ ]</span>' : '<span class="json-empty">{ }</span>';
    }

    let html = `<div class="json-tree-container">`;

    entries.forEach(([key, val], idx) => {
        const hasChildren = (val !== null && typeof val === 'object' && (Array.isArray(val) ? val.length > 0 : Object.keys(val).length > 0));

        html += `<div class="json-tree-line">`;

        if (hasChildren) {
            html += `<button class="json-toggle">▶</button>`;
            if (!isArray) {
                html += `<span class="json-key">${escapeHtml(String(key))}</span><span class="json-colon">:</span>`;
            } else {
                html += `<span class="json-key">[${key}]</span><span class="json-colon">:</span>`;
            }
            const preview = Array.isArray(val) ? `Array(${val.length})` : `Object(${Object.keys(val).length})`;
            html += `<span class="json-preview">${preview}</span>`;
            html += `</div>`;
            html += `<div class="json-expand">${renderJSONTree(val, depth + 1)}</div>`;
        } else {
            html += `<span class="json-spacer"></span>`;
            if (!isArray) {
                html += `<span class="json-key">${escapeHtml(String(key))}</span><span class="json-colon">:</span>`;
            } else {
                html += `<span class="json-key">[${key}]</span><span class="json-colon">:</span>`;
            }
            html += renderJSONTree(val, depth + 1);
            html += `</div>`;
        }
    });

    html += `</div>`;
    return html;
}

function formatValue(val) {
    if (val === null) return '<span style="color:var(--text-tertiary)">null</span>';
    if (typeof val === 'boolean') return `<span style="color:var(--orange)">${val}</span>`;
    if (typeof val === 'number') return `<span style="color:var(--blue)">${val}</span>`;
    if (typeof val === 'string') return escapeHtml(val.length > 50 ? val.substring(0, 50) + '...' : val);
    if (typeof val === 'object') return `<span style="color:var(--text-tertiary)">${Array.isArray(val) ? `Array(${val.length})` : 'Object'}</span>`;
    return escapeHtml(String(val));
}

function toggleSelectAll(e) {
    const allChecked = selected.size === documents.length && documents.length > 0;
    if (allChecked) {
        selected.clear();
    } else {
        documents.forEach(doc => selected.add(doc._id || doc.id || ''));
    }
    renderDocs();
}

function toggleRow(e) {
    const id = e.target.dataset.id;
    e.target.checked ? selected.add(id) : selected.delete(id);
    updateBulkActions();
    const card = e.target.closest('.json-card');
    if (card) card.classList.toggle('selected', e.target.checked);
    const row = e.target.closest('tr');
    if (row) row.classList.toggle('selected', e.target.checked);
}

function updateBulkActions() {
    const count = selected.size;
    const selInfo = $('#selection-info');
    const delBtn = $('#delete-btn');
    if (selInfo) selInfo.textContent = count ? `${count} selected` : '';
    if (delBtn) {
        delBtn.style.display = count ? 'inline-flex' : 'none';
        delBtn.disabled = !count;
    }
}

async function editDoc(id) {
    const doc = documents.find(d => (d._id || d.id) === id);
    if (!doc) return;
    $('#doc-id').value = id;
    $('#doc-json').value = JSON.stringify(doc, null, 2);
    $('#modal-title').textContent = 'Edit Document';
    openModal('doc-modal');
}

async function saveDoc() {
    const id = $('#doc-id').value.trim();
    const json = $('#doc-json').value.trim();
    if (!json) return showError('Document cannot be empty');

    try {
        const doc = JSON.parse(json);
        const params = new URLSearchParams();
        if (currentDatabase) params.append('database', currentDatabase);
        const url = id
            ? `/api/collections/${currentCollection}/documents/${id}?${params}`
            : `/api/collections/${currentCollection}/documents?${params}`;
        const res = await fetch(url, {
            method: id ? 'PUT' : 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(doc)
        });
        const data = await res.json();
        if (!data.success) throw new Error(data.error || 'Save failed');
        showSuccess(id ? 'Document updated' : 'Document inserted');
        closeModals();
        loadDocs();
    } catch (err) {
        showError('Failed to save: ' + err.message);
    }
}

async function deleteDoc(id) {
    if (!confirm('Delete this document?')) return;
    try {
        const params = new URLSearchParams({ database: currentDatabase });
        const res = await fetch(`/api/collections/${currentCollection}/documents/${id}?${params}`, { method: 'DELETE' });
        if (!res.ok) throw new Error(await res.text());
        showSuccess('Document deleted');
        loadDocs();
    } catch (err) {
        showError('Failed to delete: ' + err.message);
    }
}

async function bulkDelete() {
    if (!selected.size || !confirm(`Delete ${selected.size} document(s)?`)) return;
    try {
        const ids = Array.from(selected);
        const params = new URLSearchParams({ database: currentDatabase });
        const res = await fetch(`/api/collections/${currentCollection}/documents/bulk-delete?${params}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ids })
        });
        if (!res.ok) throw new Error(await res.text());
        showSuccess(`${selected.size} document(s) deleted`);
        selected.clear();
        loadDocs();
    } catch (err) {
        showError('Failed to delete: ' + err.message);
    }
}

async function createCollection(name) {
    if (!name) return;
    try {
        const params = new URLSearchParams({ database: currentDatabase });
        const res = await fetch(`/api/collections?${params}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        if (!res.ok) throw new Error(await res.text());
        showSuccess('Collection created');
        closeModals();
        $('#collection-name').value = '';
        await loadCollections();
        selectCollection(name);
    } catch (err) {
        showError('Failed to create: ' + err.message);
    }
}

function showCreateDatabaseModal() {
    openModal('database-modal');
    $('#database-name').value = '';
    $('#database-name').focus();
}

async function createDatabase(name) {
    if (!name) return;
    try {
        const res = await fetch('/api/databases', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        const data = await res.json();
        if (!data.success) throw new Error(data.error || 'Failed to create database');
        showSuccess(`Database "${name}" created successfully`);
        closeModals();
        $('#database-name').value = '';
        await loadDatabases();
        selectDatabase(name);
    } catch (err) {
        showError('Failed to create database: ' + err.message);
    }
}

function goToPage(p) {
    const max = Math.ceil(total / pageSize);
    if (p < 1 || p > max) return;
    page = p;
    loadDocs();
}

function updatePagination() {
    const max = Math.ceil(total / pageSize) || 1;
    const pageInfo = $('#page-info');
    const prevBtn = $('#prev-page');
    const nextBtn = $('#next-page');
    if (pageInfo) pageInfo.textContent = total === 0 ? '0/0' : `${page}/${max}`;
    if (prevBtn) prevBtn.disabled = page === 1;
    if (nextBtn) nextBtn.disabled = page >= max;
}

// Schema functionality
async function loadSchema() {
    if (!currentCollection) return;
    const tbody = $('#schema-table tbody');
    tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;padding:30px;"><div class="loading"><div class="spinner"></div></div></td></tr>';
    try {
        const params = new URLSearchParams({ database: currentDatabase, page: 1, limit: 100 });
        const res = await fetch(`/api/collections/${currentCollection}/documents?${params}`);
        const data = await res.json();
        const docs = data.success && data.data ? data.data.documents : [];

        const schema = inferSchema(docs);
        renderSchema(schema);
    } catch (err) {
        showError('Failed to load schema: ' + err.message);
        tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;color:var(--text-tertiary)">Failed to load schema</td></tr>';
    }
}

function inferSchema(docs) {
    const fields = {};
    const totalDocs = docs.length;

    docs.forEach(doc => {
        Object.entries(doc).forEach(([key, value]) => {
            if (!fields[key]) {
                fields[key] = { types: new Set(), count: 0, nullable: false };
            }
            fields[key].count++;
            fields[key].types.add(getType(value));
            if (value === null || value === undefined) {
                fields[key].nullable = true;
            }
        });
    });

    return Object.entries(fields).map(([name, info]) => ({
        name,
        type: Array.from(info.types).join(' | '),
        nullable: info.nullable || info.count < totalDocs,
        frequency: totalDocs > 0 ? Math.round((info.count / totalDocs) * 100) : 0
    }));
}

function getType(value) {
    if (value === null) return 'null';
    if (Array.isArray(value)) return 'array';
    if (value instanceof Date) return 'date';
    if (typeof value === 'object') return 'object';
    return typeof value;
}

function renderSchema(schema) {
    const tbody = $('#schema-table tbody');
    if (!schema.length) {
        tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;color:var(--text-tertiary)">No schema data</td></tr>';
        return;
    }
    tbody.innerHTML = schema.map(field => `
        <tr>
            <td><strong>${escapeHtml(field.name)}</strong></td>
            <td><code>${escapeHtml(field.type)}</code></td>
            <td>${field.nullable ? 'Yes' : 'No'}</td>
            <td>${field.frequency}%</td>
        </tr>
    `).join('');
}

// Indexes functionality
async function loadIndexes() {
    if (!currentCollection) return;
    const tbody = $('#indexes-table tbody');
    tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;padding:30px;"><div class="loading"><div class="spinner"></div></div></td></tr>';
    try {
        console.log('Loading indexes for:', currentDatabase, currentCollection);
        const params = new URLSearchParams();
        if (currentDatabase) params.append('database', currentDatabase);
        const res = await fetch(`/api/collections/${currentCollection}/indexes?${params}`);
        const data = await res.json();
        console.log('Indexes response:', data);
        const indexes = data.success ? data.data : [];
        console.log('Parsed indexes:', indexes);
        renderIndexes(indexes);
    } catch (err) {
        console.error('Indexes load error:', err);
        showError('Failed to load indexes: ' + err.message);
    }
}

function renderIndexes(indexes) {
    const tbody = $('#indexes-table tbody');
    if (!indexes.length) {
        tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;color:var(--text-tertiary)">No indexes</td></tr>';
        return;
    }
    tbody.innerHTML = indexes.map(idx => `
        <tr>
            <td><strong>${escapeHtml(idx.name)}</strong></td>
            <td><code>${JSON.stringify(idx.keys)}</code></td>
            <td>${idx.unique ? 'Yes' : 'No'}</td>
            <td class="row-actions">
                ${idx.name !== '_id_' ? `<button class="action-btn delete" onclick="dropIndex('${escapeHtml(idx.name)}')" title="Drop">
                    <svg width="14" height="14" fill="currentColor"><path d="M5 1h4v1H5zm-2 2h8v10H3zm2 1v8h1V4zm2 0v8h1V4zm2 0v8h1V4z"/></svg>
                </button>` : '<span style="color:var(--text-tertiary)">Default</span>'}
            </td>
        </tr>
    `).join('');
}

async function createIndex() {
    const keysStr = $('#index-keys').value.trim();
    const unique = $('#index-unique').checked;

    if (!keysStr) return showError('Keys are required');

    try {
        const keys = JSON.parse(keysStr);
        const res = await fetch(`/api/collections/${currentCollection}/indexes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ keys, unique })
        });

        if (!res.ok) throw new Error(await res.text());

        showSuccess('Index created');
        closeModals();
        $('#index-keys').value = '';
        $('#index-unique').checked = false;
        loadIndexes();
    } catch (err) {
        showError('Failed to create index: ' + err.message);
    }
}

async function dropIndex(name) {
    if (!confirm(`Drop index "${name}"?`)) return;

    try {
        const res = await fetch(`/api/collections/${currentCollection}/indexes/${name}`, {
            method: 'DELETE'
        });

        if (!res.ok) throw new Error(await res.text());

        showSuccess('Index dropped');
        loadIndexes();
    } catch (err) {
        showError('Failed to drop index: ' + err.message);
    }
}

function openModal(id) {
    $('#' + id).classList.add('active');
}

function closeModals() {
    $$('.modal').forEach(m => m.classList.remove('active'));
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `
        <ion-icon name="${type === 'success' ? 'checkmark-circle' : type === 'error' ? 'alert-circle' : 'information-circle'}"></ion-icon>
        <span>${message}</span>
    `;
    document.body.appendChild(toast);
    setTimeout(() => toast.classList.add('show'), 10);
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

function showSuccess(msg) {
    showToast(msg, 'success');
}

function showError(msg) {
    showToast(msg, 'error');
}

function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024, sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

let pipelineStages = [];

function initAggregation() {
    $('#add-stage-btn').onclick = addPipelineStage;
    $('#run-pipeline-btn').onclick = runAggregation;
}

function addPipelineStage() {
    const stageId = Date.now();
    const stage = {
        id: stageId,
        operator: '$match',
        code: '{}'
    };
    pipelineStages.push(stage);
    renderPipeline();
}

function renderPipeline() {
    const container = $('#pipeline-stages');

    if (pipelineStages.length === 0) {
        container.innerHTML = '<div class="empty-pipeline"><p>No stages added. Click "Add Stage" to begin.</p></div>';
        return;
    }

    container.innerHTML = pipelineStages.map((stage, index) => `
        <div class="pipeline-stage" data-stage-id="${stage.id}">
            <div class="stage-header">
                <div class="stage-title">
                    <span class="stage-number">${index + 1}</span>
                    <select class="stage-select" onchange="updateStageOperator(${stage.id}, this.value)">
                        ${getStageOperators().map(op =>
        `<option value="${op}" ${stage.operator === op ? 'selected' : ''}>${op}</option>`
    ).join('')}
                    </select>
                </div>
                <div class="stage-actions">
                    <button class="action-btn" onclick="moveStage(${index}, -1)" ${index === 0 ? 'disabled' : ''} title="Move Up">
                        <ion-icon name="arrow-up-outline"></ion-icon>
                    </button>
                    <button class="action-btn" onclick="moveStage(${index}, 1)" ${index === pipelineStages.length - 1 ? 'disabled' : ''} title="Move Down">
                        <ion-icon name="arrow-down-outline"></ion-icon>
                    </button>
                    <button class="action-btn delete" onclick="removeStage(${stage.id})" title="Remove">
                        <ion-icon name="trash-outline"></ion-icon>
                    </button>
                </div>
            </div>
            <div class="stage-body">
                <textarea onchange="updateStageCode(${stage.id}, this.value)" placeholder='${getPlaceholder(stage.operator)}'>${escapeHtml(stage.code)}</textarea>
            </div>
        </div>
    `).join('');
}

function getStageOperators() {
    return ['$match', '$project', '$group', '$sort', '$limit', '$skip', '$unwind', '$lookup', '$addFields', '$count', '$sample'];
}

function getPlaceholder(operator) {
    const placeholders = {
        '$match': '{"field": "value"}',
        '$project': '{"field": 1}',
        '$group': '{"_id": "$field", "count": {"$sum": 1}}',
        '$sort': '{"field": 1}',
        '$limit': '10',
        '$skip': '0',
        '$unwind': '"$arrayField"',
        '$lookup': '{"from": "collection", "localField": "field", "foreignField": "field", "as": "result"}',
        '$addFields': '{"newField": "$existingField"}',
        '$count': '"total"',
        '$sample': '{"size": 10}'
    };
    return placeholders[operator] || '{}';
}

function updateStageOperator(stageId, operator) {
    const stage = pipelineStages.find(s => s.id === stageId);
    if (stage) {
        stage.operator = operator;
        stage.code = getPlaceholder(operator);
        renderPipeline();
    }
}

function updateStageCode(stageId, code) {
    const stage = pipelineStages.find(s => s.id === stageId);
    if (stage) stage.code = code;
}

function removeStage(stageId) {
    pipelineStages = pipelineStages.filter(s => s.id !== stageId);
    renderPipeline();
}

function moveStage(index, direction) {
    const newIndex = index + direction;
    if (newIndex < 0 || newIndex >= pipelineStages.length) return;

    [pipelineStages[index], pipelineStages[newIndex]] = [pipelineStages[newIndex], pipelineStages[index]];
    renderPipeline();
}

async function runAggregation() {
    if (!currentCollection || pipelineStages.length === 0) {
        showError('Add at least one stage to run the pipeline');
        return;
    }

    try {
        const pipeline = pipelineStages.map(stage => {
            let value;
            try {
                value = JSON.parse(stage.code);
            } catch (e) {
                // Handle non-JSON values like strings for $unwind, $count
                value = stage.code.trim().startsWith('"') ? JSON.parse(stage.code) : stage.code;
            }
            return { [stage.operator]: value };
        });

        console.log('Running pipeline:', pipeline);

        const res = await fetch(`/api/collections/${currentCollection}/aggregate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(pipeline)
        });

        const data = await res.json();

        if (!data.success) {
            showError(data.error || 'Aggregation failed');
            return;
        }

        const results = data.data || [];
        $('#results-count').textContent = `${results.length} document${results.length !== 1 ? 's' : ''}`;

        const output = $('#aggregation-output');
        if (results.length === 0) {
            output.innerHTML = '<div style="text-align:center;color:var(--text-tertiary);padding:20px;">No results</div>';
        } else {
            output.innerHTML = results.map(doc =>
                `<div class="json-card"><div class="json-card-body">${renderJSONTree(doc, 0)}</div></div>`
            ).join('');
        }

        showSuccess('Aggregation completed');
    } catch (err) {
        console.error('Aggregation error:', err);
        showError('Failed to run aggregation: ' + err.message);
    }
}

function loadAggregation() {
    pipelineStages = [];
    renderPipeline();
    $('#aggregation-output').innerHTML = '';
    $('#results-count').textContent = '';
}


function renderDatabases() {
    const list = $('#databases-list');
    if (!databases.length) {
        list.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-tertiary)">No databases</div>';
        return;
    }
    list.innerHTML = databases.map(db => {
        const safeName = escapeHtml(db.name);
        const dataName = db.name.toLowerCase();
        const jsonName = JSON.stringify(db.name);
        return `
    <div class="database-item" onclick='selectDatabase(${jsonName})' oncontextmenu='showDbContextMenu(event, ${jsonName})' data-name="${dataName}">
      <div class="item-name">
        <ion-icon name="server-outline"></ion-icon>
        ${safeName}
      </div>
      <span class="item-count">${formatSize(db.sizeOnDisk || 0)}</span>
    </div>`;
    }).join('');
}

function renderCollections() {
    const list = $('#collections-list');
    if (!collections.length) {
        list.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-tertiary)">No collections</div>';
        return;
    }
    list.innerHTML = collections.map(col => {
        const safeName = escapeHtml(col.name);
        const dataName = col.name.toLowerCase();
        const isActive = col.name === currentCollection;
        const jsonName = JSON.stringify(col.name);
        return `
    <div class="collection-item ${isActive ? 'active' : ''}" onclick='selectCollection(${jsonName})' oncontextmenu='showCollContextMenu(event, ${jsonName})' data-name="${dataName}">
      <div class="item-name">
        <ion-icon name="folder-outline"></ion-icon>
        ${safeName}
      </div>
      <span class="item-count">${col.document_count || col.count || 0}</span>
    </div>`;
    }).join('');
}

function filterDatabases(query) {
    const items = document.querySelectorAll('.database-item');
    const lowerQuery = query.toLowerCase();
    items.forEach(item => {
        const name = item.dataset.name || '';
        item.style.display = name.includes(lowerQuery) ? '' : 'none';
    });
}

function filterCollections(query) {
    const items = document.querySelectorAll('.collection-item');
    const lowerQuery = query.toLowerCase();
    items.forEach(item => {
        const name = item.dataset.name || '';
        item.style.display = name.includes(lowerQuery) ? '' : 'none';
    });
}

function showDbContextMenu(e, dbName) {
    e.preventDefault();
    e.stopPropagation();
    closeContextMenu();
    const menu = document.createElement('div');
    menu.className = 'context-menu';
    menu.style.left = e.pageX + 'px';
    menu.style.top = e.pageY + 'px';
    const jsonName = JSON.stringify(dbName);
    menu.innerHTML = `
        <div class="context-item danger" onclick='event.stopPropagation(); deleteDatabase(${jsonName})'>
            <ion-icon name="trash-outline"></ion-icon> Delete Database
        </div>
    `;
    document.body.appendChild(menu);
    setTimeout(() => menu.classList.add('show'), 10);
    setTimeout(() => document.addEventListener('click', closeContextMenu, { once: true }), 100);
}

function showCollContextMenu(e, collName) {
    e.preventDefault();
    e.stopPropagation();
    closeContextMenu();
    const menu = document.createElement('div');
    menu.className = 'context-menu';
    menu.style.left = e.pageX + 'px';
    menu.style.top = e.pageY + 'px';
    const jsonName = JSON.stringify(collName);
    menu.innerHTML = `
        <div class="context-item danger" onclick='event.stopPropagation(); deleteCollection(${jsonName})'>
            <ion-icon name="trash-outline"></ion-icon> Delete Collection
        </div>
    `;
    document.body.appendChild(menu);
    setTimeout(() => menu.classList.add('show'), 10);
    setTimeout(() => document.addEventListener('click', closeContextMenu, { once: true }), 100);
}

function closeContextMenu() {
    $$('.context-menu').forEach(m => m.remove());
}

async function deleteDatabase(dbName) {
    closeContextMenu();
    if (!confirm(`Delete database "${dbName}"? This cannot be undone!`)) return;

    try {
        const response = await fetch(`/api/databases/${encodeURIComponent(dbName)}`, {
            method: 'DELETE'
        });

        const data = await response.json();

        if (data.success) {
            showSuccess(`Database "${dbName}" deleted successfully`);
            await loadDatabases();
            // If the deleted database was the current one, clear current selections
            if (currentDatabase === dbName) {
                currentDatabase = null;
                currentCollection = null;
                $$('#collections-list').innerHTML = '<div class="empty-state">Select a database</div>';
                $$('#main-content').innerHTML = '<div class="empty-state">Select a database and collection</div>';
            }
        } else {
            showError(data.error || 'Failed to delete database');
        }
    } catch (error) {
        console.error('Delete database error:', error);
        showError('Failed to delete database: ' + error.message);
    }
}

async function deleteCollection(collName) {
    closeContextMenu();
    if (!confirm(`Delete collection "${collName}"? This cannot be undone!`)) return;
    try {
        const params = new URLSearchParams();
        if (currentDatabase) params.append('database', currentDatabase);
        const res = await fetch(`/api/collections/${encodeURIComponent(collName)}?${params}`, { method: 'DELETE' });
        const data = await res.json();
        if (!data.success) throw new Error(data.error || 'Delete failed');
        showSuccess('Collection deleted');
        await loadCollections();
    } catch (err) {
        showError('Failed to delete: ' + err.message);
    }
}
