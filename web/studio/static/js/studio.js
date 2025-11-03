// State
const state = {
    currentTable: null,
    data: null,
    changes: new Map(),
    page: 1,
    limit: 50,
    tablesCache: null
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadTables();
    setupEventListeners();
});

// Setup
function setupEventListeners() {
    document.getElementById('save-btn').addEventListener('click', saveChanges);
    document.getElementById('add-btn').addEventListener('click', addRow);
    document.getElementById('refresh-btn').addEventListener('click', refreshData);
    document.getElementById('delete-selected-btn').addEventListener('click', deleteSelected);
    document.getElementById('prev-btn').addEventListener('click', () => changePage(-1));
    document.getElementById('next-btn').addEventListener('click', () => changePage(1));
    document.getElementById('search-tables').addEventListener('input', debounce(filterTables, 200));
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
        <div class="table-item" data-table="${table.name}" onclick="selectTable('${table.name}')">
            <span>${table.name}</span>
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
            document.getElementById('row-count').textContent = `${json.data.rows.length} of ${json.data.total}`;
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
        container.innerHTML = `
            <div class="empty-state">
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"></path>
                </svg>
                <div>No data in this table</div>
            </div>
        `;
        return;
    }
    
    const html = `
        <table class="data-table">
            <thead>
                <tr>
                    <th style="width: 60px;"></th>
                    ${data.columns.map(col => `
                        <th>
                            ${col.name}
                            <span class="type-badge">${col.type}</span>
                        </th>
                    `).join('')}
                </tr>
            </thead>
            <tbody>
                ${data.rows.map((row, idx) => renderRow(row, idx, data.columns)).join('')}
            </tbody>
        </table>
    `;
    
    container.innerHTML = html;
}

// Render row
function renderRow(row, idx, columns) {
    const rowId = row.id || idx;
    
    return `
        <tr>
            <td>
                <input type="checkbox" class="row-checkbox" data-row="${rowId}" style="cursor: pointer;" onchange="toggleRowSelection(this)">
            </td>
            ${columns.map(col => `
                <td class="cell" data-row="${rowId}" data-column="${col.name}" ondblclick="editCell(this)">
                    ${formatValue(row[col.name])}
                </td>
            `).join('')}
        </tr>
    `;
}

// Toggle row selection
function toggleRowSelection(checkbox) {
    const row = checkbox.closest('tr');
    if (checkbox.checked) {
        row.style.background = '#2a3a4a';
    } else {
        row.style.background = '';
    }
    
    // Show/hide delete button
    const anyChecked = document.querySelectorAll('.row-checkbox:checked').length > 0;
    document.getElementById('delete-selected-btn').style.display = anyChecked ? 'block' : 'none';
}

// Delete selected rows
async function deleteSelected() {
    const checked = document.querySelectorAll('.row-checkbox:checked');
    if (checked.length === 0) return;
    
    if (!confirm(`Delete ${checked.length} selected row(s)?`)) return;
    
    const rowIds = Array.from(checked).map(cb => cb.dataset.row);
    
    try {
        const res = await fetch(`/api/tables/${state.currentTable}/delete`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ row_ids: rowIds })
        });
        
        const json = await res.json();
        
        if (json.success) {
            alert('✓ ' + json.message);
            refreshData();
        } else {
            alert('Error: ' + json.message);
        }
    } catch (err) {
        alert('Failed to delete: ' + err.message);
    }
}

// Format value
function formatValue(value) {
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
    return `<span class="value-string">${value}</span>`;
}

// Edit cell
function editCell(cell) {
    if (cell.querySelector('input')) return;
    
    const rowId = cell.dataset.row;
    const column = cell.dataset.column;
    const currentValue = cell.textContent.trim();
    
    const input = document.createElement('input');
    input.value = currentValue === 'NULL' ? '' : currentValue;
    
    input.addEventListener('blur', () => saveCell(cell, input, rowId, column));
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') input.blur();
        if (e.key === 'Escape') {
            cell.innerHTML = formatValue(currentValue);
            cell.classList.remove('cell-editing');
        }
    });
    
    cell.innerHTML = '';
    cell.appendChild(input);
    cell.classList.add('cell-editing');
    input.focus();
    input.select();
}

// Save cell
function saveCell(cell, input, rowId, column) {
    const newValue = input.value;
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
            alert('✓ Changes saved successfully');
            refreshData();
        } else {
            alert('Error: ' + json.message);
        }
    } catch (err) {
        alert('Failed to save: ' + err.message);
    } finally {
        saveBtn.disabled = false;
        saveBtn.textContent = 'Save changes';
    }
}

// Add row
function addRow() {
    if (!state.currentTable || !state.data) return;
    
    const data = {};
    state.data.columns.forEach(col => {
        if (!col.primary_key) {
            const value = prompt(`${col.name} (${col.type}):`);
            if (value !== null) data[col.name] = value;
        }
    });
    
    if (Object.keys(data).length === 0) return;
    
    fetch(`/api/tables/${state.currentTable}/add`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ data })
    })
    .then(res => res.json())
    .then(json => {
        if (json.success) {
            alert('✓ Row added');
            refreshData();
        } else {
            alert('Error: ' + json.message);
        }
    });
}

// Delete row
function deleteRow(rowId) {
    if (!confirm('Delete this row?')) return;
    
    fetch(`/api/tables/${state.currentTable}/rows/${rowId}`, { method: 'DELETE' })
    .then(res => res.json())
    .then(json => {
        if (json.success) {
            alert('✓ Row deleted');
            refreshData();
        } else {
            alert('Error: ' + json.message);
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
