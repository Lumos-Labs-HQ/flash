// State management
const state = {
    currentTable: null,
    data: null,
    changes: new Map(),
    page: 1,
    limit: 50
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadTables();
    setupEventListeners();
});

// Setup event listeners
function setupEventListeners() {
    document.getElementById('save-btn').addEventListener('click', saveChanges);
    document.getElementById('add-btn').addEventListener('click', addRow);
    document.getElementById('refresh-btn').addEventListener('click', refreshData);
    document.getElementById('prev-btn').addEventListener('click', () => changePage(-1));
    document.getElementById('next-btn').addEventListener('click', () => changePage(1));
    document.getElementById('search-tables').addEventListener('input', filterTables);
}

// Load tables list
async function loadTables() {
    try {
        const res = await fetch('/api/tables');
        const json = await res.json();
        
        if (json.success) {
            renderTablesList(json.data);
        }
    } catch (err) {
        console.error('Failed to load tables:', err);
    }
}

// Render tables in sidebar
function renderTablesList(tables) {
    const container = document.getElementById('tables-list');
    container.innerHTML = tables.map(table => `
        <div 
            class="table-item px-3 py-2 rounded cursor-pointer hover:bg-gray-800 transition"
            data-table="${table.name}"
            onclick="selectTable('${table.name}')"
        >
            <div class="font-medium">${table.name}</div>
            <div class="text-xs text-gray-400">${table.row_count} rows</div>
        </div>
    `).join('');
}

// Filter tables
function filterTables(e) {
    const search = e.target.value.toLowerCase();
    const items = document.querySelectorAll('.table-item');
    
    items.forEach(item => {
        const name = item.dataset.table.toLowerCase();
        item.style.display = name.includes(search) ? 'block' : 'none';
    });
}

// Select table
async function selectTable(tableName) {
    state.currentTable = tableName;
    state.page = 1;
    state.changes.clear();
    
    // Update UI
    document.getElementById('current-table').textContent = tableName;
    document.querySelectorAll('.table-item').forEach(item => {
        item.classList.toggle('bg-gray-800', item.dataset.table === tableName);
    });
    
    await loadTableData();
}

// Load table data
async function loadTableData() {
    if (!state.currentTable) return;
    
    try {
        const res = await fetch(`/api/tables/${state.currentTable}?page=${state.page}&limit=${state.limit}`);
        const json = await res.json();
        
        if (json.success) {
            state.data = json.data;
            renderDataGrid(json.data);
            updatePagination(json.data);
        }
    } catch (err) {
        console.error('Failed to load data:', err);
    }
}

// Render data grid
function renderDataGrid(data) {
    const container = document.getElementById('grid-container');
    
    if (!data.rows || data.rows.length === 0) {
        container.innerHTML = '<div class="text-center text-gray-500 py-20">No data found</div>';
        return;
    }
    
    const html = `
        <div class="overflow-x-auto">
            <table class="min-w-full border border-gray-200">
                <thead class="bg-gray-100">
                    <tr>
                        <th class="px-4 py-2 border-b text-left text-xs font-semibold text-gray-600 uppercase">Actions</th>
                        ${data.columns.map(col => `
                            <th class="px-4 py-2 border-b text-left text-xs font-semibold text-gray-600 uppercase">
                                ${col.name}
                                <span class="text-gray-400 font-normal">${col.type}</span>
                            </th>
                        `).join('')}
                    </tr>
                </thead>
                <tbody>
                    ${data.rows.map((row, idx) => renderRow(row, idx, data.columns)).join('')}
                </tbody>
            </table>
        </div>
    `;
    
    container.innerHTML = html;
}

// Render single row
function renderRow(row, idx, columns) {
    const rowId = row.id || idx;
    
    return `
        <tr class="table-row border-b">
            <td class="px-4 py-2">
                <button 
                    onclick="deleteRow('${rowId}')" 
                    class="text-red-600 hover:text-red-800 text-sm"
                >
                    🗑️
                </button>
            </td>
            ${columns.map(col => `
                <td 
                    class="px-4 py-2 cell" 
                    data-row="${rowId}" 
                    data-column="${col.name}"
                    ondblclick="editCell(this)"
                >
                    ${formatValue(row[col.name])}
                </td>
            `).join('')}
        </tr>
    `;
}

// Format cell value
function formatValue(value) {
    if (value === null || value === undefined) return '<span class="text-gray-400">NULL</span>';
    if (typeof value === 'boolean') return value ? 'true' : 'false';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
}

// Edit cell
function editCell(cell) {
    if (cell.querySelector('input')) return;
    
    const rowId = cell.dataset.row;
    const column = cell.dataset.column;
    const currentValue = cell.textContent.trim();
    
    const input = document.createElement('input');
    input.type = 'text';
    input.value = currentValue === 'NULL' ? '' : currentValue;
    input.className = 'w-full px-2 py-1 border border-blue-500 rounded focus:outline-none';
    
    input.addEventListener('blur', () => saveCell(cell, input, rowId, column));
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') input.blur();
        if (e.key === 'Escape') {
            cell.textContent = currentValue;
        }
    });
    
    cell.textContent = '';
    cell.appendChild(input);
    cell.classList.add('cell-editing');
    input.focus();
}

// Save cell
function saveCell(cell, input, rowId, column) {
    const newValue = input.value;
    const oldValue = cell.textContent;
    
    if (newValue !== oldValue) {
        // Track change
        if (!state.changes.has(rowId)) {
            state.changes.set(rowId, {});
        }
        state.changes.get(rowId)[column] = newValue;
        
        cell.classList.add('cell-dirty');
        document.getElementById('save-btn').classList.remove('hidden');
    }
    
    cell.textContent = formatValue(newValue);
    cell.classList.remove('cell-editing');
}

// Save all changes
async function saveChanges() {
    if (state.changes.size === 0) return;
    
    const changes = [];
    state.changes.forEach((cols, rowId) => {
        Object.entries(cols).forEach(([column, value]) => {
            changes.push({
                row_id: rowId,
                column: column,
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
            document.getElementById('save-btn').classList.add('hidden');
            document.querySelectorAll('.cell-dirty').forEach(cell => {
                cell.classList.remove('cell-dirty');
            });
            alert('Changes saved successfully!');
            refreshData();
        } else {
            alert('Error: ' + json.message);
        }
    } catch (err) {
        alert('Failed to save changes: ' + err.message);
    }
}

// Add row
function addRow() {
    if (!state.currentTable) return;
    
    const data = {};
    state.data.columns.forEach(col => {
        if (!col.primary_key) {
            const value = prompt(`Enter value for ${col.name}:`);
            if (value !== null) {
                data[col.name] = value;
            }
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
            alert('Row added successfully!');
            refreshData();
        } else {
            alert('Error: ' + json.message);
        }
    })
    .catch(err => alert('Failed to add row: ' + err.message));
}

// Delete row
function deleteRow(rowId) {
    if (!confirm('Are you sure you want to delete this row?')) return;
    
    fetch(`/api/tables/${state.currentTable}/rows/${rowId}`, {
        method: 'DELETE'
    })
    .then(res => res.json())
    .then(json => {
        if (json.success) {
            alert('Row deleted successfully!');
            refreshData();
        } else {
            alert('Error: ' + json.message);
        }
    })
    .catch(err => alert('Failed to delete row: ' + err.message));
}

// Refresh data
function refreshData() {
    state.changes.clear();
    document.getElementById('save-btn').classList.add('hidden');
    loadTableData();
}

// Change page
function changePage(delta) {
    state.page += delta;
    if (state.page < 1) state.page = 1;
    loadTableData();
}

// Update pagination
function updatePagination(data) {
    const pagination = document.getElementById('pagination');
    const pageInfo = document.getElementById('page-info');
    const prevBtn = document.getElementById('prev-btn');
    const nextBtn = document.getElementById('next-btn');
    
    if (data.total === 0) {
        pagination.classList.add('hidden');
        return;
    }
    
    pagination.classList.remove('hidden');
    
    const start = (data.page - 1) * data.limit + 1;
    const end = Math.min(data.page * data.limit, data.total);
    pageInfo.textContent = `${start}-${end} of ${data.total}`;
    
    prevBtn.disabled = data.page === 1;
    nextBtn.disabled = end >= data.total;
}
