// State
const state = {
    currentTable: null,
    data: null,
    changes: new Map(),
    page: 1,
    limit: 50,
    tablesCache: null,
    foreignKeys: new Map() 
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadTables();
    setupEventListeners();
});

// Setup
function setupEventListeners() {
    document.getElementById('save-btn').addEventListener('click', saveChanges);
    document.getElementById('add-btn').addEventListener('click', showAddRowDialog);
    document.getElementById('refresh-btn').addEventListener('click', refreshData);
    document.getElementById('delete-selected-btn').addEventListener('click', deleteSelected);
    document.getElementById('prev-btn').addEventListener('click', () => changePage(-1));
    document.getElementById('next-btn').addEventListener('click', () => changePage(1));
    document.getElementById('search-tables').addEventListener('input', debounce(filterTables, 200));
    document.addEventListener('keydown', handleKeyDown);
}

function debounce(func, wait) {
    let timeout;
    return function(...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), wait);
    };
}

// Load tables
async function loadTables() {
    try {
        const res = await fetch('/api/tables');
        const json = await res.json();
        
        if (json.success) {
            state.tablesCache = json.data;
            renderTablesList(json.data);
        }
    } catch (err) {
        console.error('Failed to load tables:', err);
    }
}

// Render tables
function renderTablesList(tables) {
    const container = document.getElementById('tables-list');
    
    if (!tables || tables.length === 0) {
        container.innerHTML = '<div style="padding: 12px; color: #666; font-size: 12px;">No models found</div>';
        return;
    }
    
    container.innerHTML = tables.map(table => `
        <div class="table-item" data-table="${table.name}" onclick="selectTable('${table.name}')" title="${table.name}">
            <span class="table-item-name">${table.name}</span>
            <span class="table-count">${table.row_count}</span>
        </div>
    `).join('');
}

// Filter tables
function filterTables(e) {
    const search = e.target.value.toLowerCase();
    if (!state.tablesCache) return;
    
    if (!search) {
        renderTablesList(state.tablesCache);
        return;
    }
    
    const filtered = state.tablesCache.filter(t => t.name.toLowerCase().includes(search));
    renderTablesList(filtered);
}

// Select table
async function selectTable(tableName) {
    state.currentTable = tableName;
    state.page = 1;
    state.changes.clear();
    
    document.getElementById('current-table').textContent = tableName;
    document.getElementById('save-btn').style.display = 'none';
    
    document.querySelectorAll('.table-item').forEach(item => {
        item.classList.toggle('active', item.dataset.table === tableName);
    });
    
    showLoadingSkeleton();
    await loadTableData();
}

// Loading skeleton
function showLoadingSkeleton() {
    document.getElementById('grid-container').innerHTML = `
        <div style="padding: 16px;">
            <div class="skeleton" style="height: 40px; margin-bottom: 8px;"></div>
            <div class="skeleton" style="height: 300px;"></div>
        </div>
    `;
}

// Load data
async function loadTableData() {
    if (!state.currentTable) return;
    
    try {
        const res = await fetch(`/api/tables/${state.currentTable}?page=${state.page}&limit=${state.limit}`);
        const json = await res.json();
        
        if (json.success) {
            state.data = json.data;
            const rowCount = json.data.rows ? json.data.rows.length : 0;
            document.getElementById('row-count').textContent = `${rowCount} of ${json.data.total || 0}`;
            currentColumns = json.data.columns || [];
            renderDataGrid(json.data);
            updatePagination(json.data);
        }
    } catch (err) {
        console.error('Failed to load data:', err);
    }
}

// Render grid
function renderDataGrid(data) {
    const container = document.getElementById('grid-container');
    
    if (!data.rows || data.rows.length === 0) {
        if (data.columns && data.columns.length > 0) {
            const schemaInfo = `
                <div class="table-schema-info">
                    <div class="schema-title">
                        <span class="iconify" data-icon="mdi:table" style="font-size: 20px;"></span>
                        Table Structure: ${state.currentTable}
                    </div>
                    <div class="schema-columns-grid">
                        ${data.columns.map(col => {
                            let badges = [];
                            if (col.primary_key) badges.push('<span class="badge badge-primary">PK</span>');
                            if (col.foreign_key_table) badges.push('<span class="badge badge-purple">FK → ' + col.foreign_key_table + '.' + col.foreign_key_column + '</span>');
                            if (col.isUnique) badges.push('<span class="badge badge-success">Unique</span>');
                            if (col.isAutoIncrement) badges.push('<span class="badge badge-warning">Auto Inc</span>');
                            if (!col.nullable) badges.push('<span class="badge badge-info">NOT NULL</span>');
                            if (col.default !== null && col.default !== undefined && col.default !== '') badges.push('<span class="badge badge-secondary">Default: ' + col.default + '</span>');
                            
                            return `
                                <div class="schema-column-card">
                                    <div class="schema-column-header">
                                        <div class="schema-column-main">
                                            <div class="schema-column-name">${col.name}</div>
                                            <div class="schema-column-type">${col.type}</div>
                                        </div>
                                    </div>
                                    ${badges.length > 0 ? '<div class="schema-column-badges">' + badges.join('') + '</div>' : ''}
                                </div>
                            `;
                        }).join('')}
                    </div>
                </div>
            `;
            
            container.innerHTML = schemaInfo + `
                <div class="empty-state">
                    <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"></path>
                    </svg>
                    <div>No data in this table</div>
                    <div style="font-size: 12px; color: #666; margin-top: 8px;">Click "Add Record" to insert data</div>
                </div>
            `;
        }
        return;
    }
    
    // Store foreign key info
    if (data.columns) {
        const seen = new Set();
        const orderedCols = [];
        data.columns.forEach(col => {
            if (!seen.has(col.name)) {
                seen.add(col.name);
                orderedCols.push(col);
            }

            if (col.foreign_key_table) {
                state.foreignKeys.set(col.name, {
                    table: col.foreign_key_table,
                    column: col.foreign_key_column
                });
            }
        });

        const html = `
            <table class="data-table">
                <thead>
                    <tr>
                        <th style="width: 50px;"><input type="checkbox" id="select-all" onchange="toggleSelectAll(this)"></th>
                        ${orderedCols.map(col => `
                            <th title="${col.name}">
                                ${col.name}
                                <span class="type-badge">${col.type}</span>
                            </th>
                        `).join('')}
                    </tr>
                </thead>
                <tbody>
                    ${data.rows.map((row, idx) => renderRow(row, idx, orderedCols)).join('')}
                </tbody>
            </table>
        `;

        currentColumns = orderedCols;
        container.innerHTML = html;
    }
}

// Render row
function renderRow(row, idx, columns) {
    const rowId = row.id || idx;
    
    return `
        <tr>
            <td>
                <input type="checkbox" class="row-checkbox" data-row="${rowId}" style="cursor: pointer;" onchange="toggleRowSelection(this)">
            </td>
            ${columns.map(col => {
                const fk = state.foreignKeys.get(col.name);
                const value = row[col.name];
                const valueStr = String(value || '');
                
                // FK cells have special click handler, others are editable
                const cellClass = fk && value ? 'cell value-fk' : 'cell';
                const onClick = fk && value ? 
                    `onclick="event.stopPropagation(); navigateToForeignKey('${fk.table}', '${fk.column}', '${value}'); return false;"` : 
                    `onclick="editCell(this)"`;
                
                const titleText = fk ? `Click to view ${fk.table}.${fk.column} = ${value}` : valueStr;
                
                return `
                    <td class="${cellClass}" data-row="${rowId}" data-column="${col.name}" 
                        ${onClick} 
                        title="${titleText.replace(/"/g, '&quot;')}">
                        ${formatValue(value, fk)}
                    </td>
                `;
            }).join('')}
        </tr>
    `;
}

// foreign key reference - Show popup
async function navigateToForeignKey(tableName, columnName, value) {
    console.log(`Showing FK reference: ${tableName}.${columnName} = ${value}`);
    
    const loadingHtml = `
        <div style="text-align: center; padding: 40px; color: #888;">
            <div class="spinner" style="margin: 0 auto 16px;"></div>
            <div>Loading reference data...</div>
        </div>
    `;
    showModal('Foreign Key Reference', loadingHtml, 'info', false); 
    
    try {
        const response = await fetch(`/api/tables/${tableName}?page=1&limit=1000`);
        const json = await response.json();
        
        if (!json.success || !json.data.rows || json.data.rows.length === 0) {
            showModal('Foreign Key Reference', `No data found in table ${tableName}`, 'warning', false);
            return;
        }
        
        const row = json.data.rows.find(r => r[columnName] == value);
        if (!row) {
            showModal('Foreign Key Reference', `No row found in ${tableName} where ${columnName} = ${value}`, 'warning', false);
            return;
        }
        
        const columns = json.data.columns;
        
        const tableHtml = `
            <div style="margin-bottom: 12px; color: #888; font-size: 12px;">
                Reference: ${tableName}.${columnName} = ${value}
            </div>
            <div style="overflow-x: auto; max-height: 400px; overflow-y: auto;">
                <table class="data-table" style="background: #1e1e1e; width: max-content; min-width: 100%;">
                    <thead>
                        <tr>
                            ${columns.map(col => `
                                <th>${col.name} <span class="type-badge">${col.type}</span></th>
                            `).join('')}
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            ${columns.map(col => `
                                <td style="white-space: nowrap;">${formatValue(row[col.name])}</td>
                            `).join('')}
                        </tr>
                    </tbody>
                </table>
            </div>
            <div style="margin-top: 16px; display: flex; gap: 8px;">
                <button class="btn btn-primary" onclick="goToTable('${tableName}', '${columnName}', '${value}')">
                    Go to ${tableName}
                </button>
            </div>
        `;
        
        showModal('Foreign Key Reference', tableHtml, 'info', false);
    } catch (err) {
        console.error('Failed to fetch FK reference:', err);
        showModal('Error', 'Failed to fetch foreign key reference', 'error', false);
    }
}

// Helper function to navigate to table with filter
async function goToTable(tableName, columnName, value) {
    document.querySelectorAll('.custom-modal').forEach(m => m.classList.remove('show'));
    
    await selectTable(tableName);
    setTimeout(() => {
        if (!state.data || !state.data.columns) return;
        currentColumns = state.data.columns;
        document.getElementById('filter-rows').innerHTML = '';
        addFilterRow('where', columnName, 'equals', value);
        applyFilters();
    }, 500);
}

// Toggle select all
function toggleSelectAll(checkbox) {
    document.querySelectorAll('.row-checkbox').forEach(cb => {
        cb.checked = checkbox.checked;
        toggleRowSelection(cb);
    });
}

// Toggle row selection
function toggleRowSelection(checkbox) {
    const row = checkbox.closest('tr');
    if (checkbox.checked) {
        row.style.background = '#2a3a4a';
    } else {
        row.style.background = '';
    }
    
    const anyChecked = document.querySelectorAll('.row-checkbox:checked').length > 0;
    document.getElementById('delete-selected-btn').style.display = anyChecked ? 'block' : 'none';
}

// Delete selected rows
async function deleteSelected() {
    const checked = document.querySelectorAll('.row-checkbox:checked');
    if (checked.length === 0) return;
    
    const rowIds = Array.from(checked).map(cb => cb.dataset.row);
    
    showConfirm(
        'Confirm Deletion',
        `Are you sure you want to delete ${checked.length} record(s)? This action cannot be undone.`,
        async () => {
            try {
                const res = await fetch(`/api/tables/${state.currentTable}/delete`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ row_ids: rowIds })
                });
                
                const json = await res.json();
                
                if (json.success) {
                    showModal('Success', json.message, 'success');
                    refreshData();
                } else {
                    showModal('Error', json.message, 'error');
                }
            } catch (err) {
                showModal('Error', 'Failed to delete: ' + err.message, 'error');
            }
        }
    );
}

// Format value
function formatValue(value, fk = null) {
    if (value === null || value === undefined) {
        return '<span class="value-null">NULL</span>';
    }
    if (typeof value === 'boolean') {
        return `<span class="value-bool">${value}</span>`;
    }
    if (typeof value === 'number') {
        return `<span class="value-number">${value}</span>`;
    }
    if (typeof value === 'object') {
        return `<span class="value-string">${JSON.stringify(value)}</span>`;
    }
    if (fk) {
        return `<span class="value-fk">${value}</span>`;
    }
    return `<span class="value-string">${value}</span>`;
}

// Keyboard handler for collapsing/expanding sidebar
function handleKeyDown(e) {
    if (e.key === 'ArrowLeft') collapseSidebar();
    if (e.key === 'ArrowRight') expandSidebar();
}

function collapseSidebar() {
    const sb = document.querySelector('.sidebar');
    if (!sb) return;
    sb.classList.add('collapsed');
}

function expandSidebar() {
    const sb = document.querySelector('.sidebar');
    if (!sb) return;
    sb.classList.remove('collapsed');
}

// Edit cell
function editCell(cell) {
    if (cell.querySelector('textarea') || cell.classList.contains('value-fk')) return;
    
    const rowId = cell.dataset.row;
    const column = cell.dataset.column;
    const currentValue = cell.textContent.trim();
    
    const textarea = document.createElement('textarea');
    textarea.value = currentValue === 'NULL' ? '' : currentValue;
    
    textarea.addEventListener('blur', () => saveCell(cell, textarea, rowId, column));
    textarea.addEventListener('keydown', (e) => {
        // Ctrl+Enter or Cmd+Enter to save
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
            textarea.blur();
        }
        // Escape to cancel
        if (e.key === 'Escape') {
            cell.innerHTML = formatValue(currentValue);
            cell.classList.remove('cell-editing');
        }
    });
    
    cell.innerHTML = '';
    cell.appendChild(textarea);
    cell.classList.add('cell-editing');
    textarea.focus();
    textarea.select();
}

// Save cell
function saveCell(cell, textarea, rowId, column) {
    const newValue = textarea.value;
    const oldValue = cell.textContent;
    
    if (newValue !== oldValue) {
        if (!state.changes.has(rowId)) {
            state.changes.set(rowId, {});
        }
        state.changes.get(rowId)[column] = newValue;
        
        cell.classList.add('cell-dirty');
        document.getElementById('save-btn').style.display = 'block';
    }
    
    cell.innerHTML = formatValue(newValue);
    cell.classList.remove('cell-editing');
}

// Save changes
async function saveChanges() {
    if (state.changes.size === 0) return;
    
    const saveBtn = document.getElementById('save-btn');
    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving...';
    
    const changes = [];
    state.changes.forEach((cols, rowId) => {
        Object.entries(cols).forEach(([colName, value]) => {
            changes.push({
                row_id: rowId,
                column: colName,
                value: value,
                action: 'update'
            });
        });
    });
    
    try {
        const res = await fetch(`/api/tables/${state.currentTable}/save`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ changes })
        });
        
        const json = await res.json();
        
        if (json.success) {
            state.changes.clear();
            showModal('Success', json.message, 'success');
            refreshData();
        } else {
            showModal('Error', json.message, 'error');
        }
    } catch (err) {
        showModal('Error', 'Failed to save: ' + err.message, 'error');
    } finally {
        saveBtn.disabled = false;
        saveBtn.textContent = '💾 Save';
    }
}

// Add row - Show modal with form
function showAddRowDialog() {
    if (!state.currentTable || !state.data) return;
    
    showAddRowModal(state.data.columns, async (data) => {
        try {
            const res = await fetch(`/api/tables/${state.currentTable}/add`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ data })
            });
            
            const json = await res.json();
            
            if (json.success) {
                showModal('Success', 'Record added successfully', 'success');
                refreshData();
            } else {
                showModal('Error', json.message, 'error');
            }
        } catch (err) {
            showModal('Error', 'Failed to add record: ' + err.message, 'error');
        }
    });
}

function addRow() {
    showAddRowDialog();
}
async function saveChanges() {
    if (state.changes.size === 0) return;
    
    const saveBtn = document.getElementById('save-btn');
    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving...';
    
    const changes = [];
    state.changes.forEach((cols, rowId) => {
        Object.entries(cols).forEach(([column, value]) => {
            changes.push({ 
                row_id: String(rowId), 
                column, 
                value: String(value), 
                action: 'update' 
            });
        });
    });
    
    try {
        const res = await fetch(`/api/tables/${state.currentTable}/save`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ changes })
        });
        
        const json = await res.json();
        
        if (json.success) {
            state.changes.clear();
            saveBtn.style.display = 'none';
            document.querySelectorAll('.cell-dirty').forEach(c => c.classList.remove('cell-dirty'));
            showModal('Success', 'Changes saved successfully', 'success');
            refreshData();
        } else {
            showModal('Error', json.message || 'Failed to save', 'error');
        }
    } catch (err) {
        showModal('Error', err.message, 'error');
    } finally {
        saveBtn.disabled = false;
        saveBtn.textContent = 'Save changes';
    }
}

// Add row
function addRow() {
    if (!state.currentTable || !state.data) return;

    showAddRowModal(state.data.columns, async (data) => {
        if (!data || Object.keys(data).length === 0) return;
        try {
            const res = await fetch(`/api/tables/${state.currentTable}/add`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ data })
            });
            const json = await res.json();
            if (json.success) {
                showModal('Success', 'Row added', 'success');
                refreshData();
            } else {
                showModal('Error', json.message || 'Failed to add row', 'error');
            }
        } catch (err) {
            showModal('Error', err.message, 'error');
        }
    });
}

// Delete row
function deleteRow(rowId) {
    showConfirm('Confirm Delete', 'Delete this row?', async () => {
        try {
            const res = await fetch(`/api/tables/${state.currentTable}/rows/${rowId}`, { method: 'DELETE' });
            const json = await res.json();
            if (json.success) {
                showModal('Success', 'Row deleted', 'success');
                refreshData();
            } else {
                showModal('Error', json.message || 'Failed to delete row', 'error');
            }
        } catch (err) {
            showModal('Error', err.message, 'error');
        }
    });
}

// Refresh
function refreshData() {
    state.changes.clear();
    document.getElementById('save-btn').style.display = 'none';
    loadTableData();
}

// Pagination
function changePage(delta) {
    state.page += delta;
    if (state.page < 1) state.page = 1;
    showLoadingSkeleton();
    loadTableData();
}

function updatePagination(data) {
    const pagination = document.getElementById('pagination');
    const pageInfo = document.getElementById('page-info');
    const prevBtn = document.getElementById('prev-btn');
    const nextBtn = document.getElementById('next-btn');
    
    if (data.total === 0) {
        pagination.style.display = 'none';
        return;
    }
    
    pagination.style.display = 'flex';
    
    const start = (data.page - 1) * data.limit + 1;
    const end = Math.min(data.page * data.limit, data.total);
    pageInfo.textContent = `${start}-${end} of ${data.total}`;
    
    prevBtn.disabled = data.page === 1;
    nextBtn.disabled = end >= data.total;
}

// Modal system - Show a custom modal with title, content, and type
function showModal(title, content, type = 'info', blocking = false) {
    document.querySelectorAll('.custom-modal').forEach(m => m.remove());
    
    const iconMap = {
        'info': '<span class="iconify" data-icon="mdi:information" style="color: #4a9eff;"></span>',
        'success': '<span class="iconify" data-icon="mdi:check-circle" style="color: #10b981;"></span>',
        'warning': '<span class="iconify" data-icon="mdi:alert" style="color: #f59e0b;"></span>',
        'error': '<span class="iconify" data-icon="mdi:alert-circle" style="color: #ef4444;"></span>'
    };
    
    const modal = document.createElement('div');
    modal.className = 'custom-modal';
    modal.innerHTML = `
        <div class="custom-modal-content">
            <div class="custom-modal-header">
                <div class="custom-modal-title">
                    ${iconMap[type] || iconMap.info}
                    ${title}
                </div>
                <button class="custom-modal-close" onclick="this.closest('.custom-modal').remove()">
                    <span class="iconify" data-icon="mdi:close"></span>
                </button>
            </div>
            <div class="custom-modal-body">
                ${content}
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    setTimeout(() => modal.classList.add('show'), 10);
    
    if (!blocking) {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });
    }
    
    // Close on Escape key
    const escHandler = (e) => {
        if (e.key === 'Escape') {
            modal.remove();
            document.removeEventListener('keydown', escHandler);
        }
    };
    document.addEventListener('keydown', escHandler);
}
