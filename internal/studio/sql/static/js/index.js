let filters = [];
let currentColumns = [];

// This function just rebuilds the UI to show the active filters
function restoreFilters(savedFilters) {
    if (!savedFilters || savedFilters.length === 0) return;

    // Clear existing filter rows
    const filterRows = document.getElementById('filter-rows');
    if (!filterRows) return;
    filterRows.innerHTML = '';

    // Rebuild filter rows from saved state (UI only)
    savedFilters.forEach((filter, index) => {
        const logic = index === 0 ? 'where' : filter.logic;
        addFilterRow(logic, filter.column, filter.operator, filter.value);
    });

    filters = savedFilters;

    updateFilterCount();
}

function toggleFilters() {
    const panel = document.getElementById('filter-panel');
    const btn = document.getElementById('filter-btn');
    panel.classList.toggle('show');
    btn.classList.toggle('active');
}

// Get column type from currentColumns
function getColumnType(columnName) {
    const col = currentColumns.find(c => c.name === columnName);
    if (!col) return 'text';

    const type = (col.type || '').toLowerCase();

    if (type.includes('uuid')) return 'uuid';
    
    if (type.includes('int') || type.includes('serial') || type.includes('decimal') ||
        type.includes('numeric') || type.includes('float') || type.includes('double') ||
        type.includes('real') || type.includes('money')) return 'number';

    if (type.includes('bool')) return 'boolean';

    if (type.includes('date') || type.includes('time') || type.includes('timestamp')) return 'datetime';
    
    if (type.includes('json')) return 'json';
    return 'text';
}

function addFilterRow(logic = 'where', column = '', operator = 'equals', value = '') {
    const row = document.createElement('div');
    row.className = 'filter-row';

    const logicSelect = logic === 'where' ?
        `<select class="filter-logic" disabled><option>where</option></select>` :
        `<select class="filter-logic"><option value="and" ${logic === 'and' ? 'selected' : ''}>and</option><option value="or" ${logic === 'or' ? 'selected' : ''}>or</option></select>`;

    const columnOptions = currentColumns.map(col =>
        `<option value="${col.name}" ${col.name === column ? 'selected' : ''}>${col.name} (${col.type || 'text'})</option>`
    ).join('');

    row.innerHTML = `
        ${logicSelect}
        <select class="filter-column" onchange="updateFilterOperators(this)">${columnOptions}</select>
        <select class="filter-operator">
            <option value="equals" ${operator === 'equals' ? 'selected' : ''}>equals</option>
            <option value="not_equals" ${operator === 'not_equals' ? 'selected' : ''}>not equals</option>
            <option value="contains" ${operator === 'contains' ? 'selected' : ''}>contains</option>
            <option value="not_contains" ${operator === 'not_contains' ? 'selected' : ''}>not contains</option>
            <option value="starts_with" ${operator === 'starts_with' ? 'selected' : ''}>starts with</option>
            <option value="ends_with" ${operator === 'ends_with' ? 'selected' : ''}>ends with</option>
            <option value="gt" ${operator === 'gt' ? 'selected' : ''}>greater than</option>
            <option value="lt" ${operator === 'lt' ? 'selected' : ''}>less than</option>
            <option value="gte" ${operator === 'gte' ? 'selected' : ''}>≥</option>
            <option value="lte" ${operator === 'lte' ? 'selected' : ''}>≤</option>
            <option value="is_null" ${operator === 'is_null' ? 'selected' : ''}>is null</option>
            <option value="is_not_null" ${operator === 'is_not_null' ? 'selected' : ''}>is not null</option>
            <option value="is_empty" ${operator === 'is_empty' ? 'selected' : ''}>is empty</option>
            <option value="is_not_empty" ${operator === 'is_not_empty' ? 'selected' : ''}>is not empty</option>
        </select>
        <input type="text" class="filter-value" value="${escapeHtmlAttr(value)}" placeholder="Value">
        <button class="filter-remove" onclick="this.parentElement.remove(); updateFilterCount();">✕</button>
    `;

    document.getElementById('filter-rows').appendChild(row);
    updateFilterCount();

    const columnSelect = row.querySelector('.filter-column');
    updateFilterOperators(columnSelect);
}

// Escape HTML attribute values
function escapeHtmlAttr(str) {
    return String(str).replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

// Update filter operators based on column type
function updateFilterOperators(selectElement) {
    const columnName = selectElement.value;
    const colType = getColumnType(columnName);
    const operatorSelect = selectElement.parentElement.querySelector('.filter-operator');
    const valueInput = selectElement.parentElement.querySelector('.filter-value');

    // Enable/disable value input based on operator
    operatorSelect.addEventListener('change', function () {
        const op = this.value;
        if (op === 'is_null' || op === 'is_not_null' || op === 'is_empty' || op === 'is_not_empty') {
            valueInput.disabled = true;
            valueInput.value = '';
            valueInput.placeholder = 'N/A';
        } else {
            valueInput.disabled = false;
            valueInput.placeholder = 'Value';
        }
    });
}

function updateFilterCount() {
    const count = document.getElementById('filter-rows').children.length;
    const badge = document.getElementById('filter-count');
    if (count > 0) {
        badge.textContent = count;
        badge.style.display = 'block';
    } else {
        badge.style.display = 'none';
    }
}

function clearFilters() {
    document.getElementById('filter-rows').innerHTML = '';
    updateFilterCount();
    filters = [];

    // Clear filters from state
    if (typeof state !== 'undefined') {
        state.filters = [];
        state.page = 1; // Reset to first page
        if (typeof state.save === 'function') {
            state.save();
        }
    }

    // Reload data without filters
    if (state.currentTable && typeof loadTableData === 'function') {
        showLoadingSkeleton();
        loadTableData();
    }
}

function applyFilters() {
    const rows = document.getElementById('filter-rows').children;
    filters = [];

    for (let row of rows) {
        const logicSelect = row.querySelector('.filter-logic');
        const logic = logicSelect ? logicSelect.value : 'where';
        const column = row.querySelector('.filter-column').value;
        const operator = row.querySelector('.filter-operator').value;
        const value = row.querySelector('.filter-value').value;

        // For null/empty checks, we don't need a value
        if (operator === 'is_null' || operator === 'is_not_null' ||
            operator === 'is_empty' || operator === 'is_not_empty') {
            if (column) {
                filters.push({ logic, column, operator, value: '' });
            }
        } else if (column && value !== '') {
            filters.push({ logic, column, operator, value });
        }
    }

    toggleFilters();

    // Store in state for persistence and server-side filtering
    if (typeof state !== 'undefined') {
        state.filters = filters;
        state.page = 1; // Reset to first page when applying filters
        if (typeof state.save === 'function') {
            state.save();
        }
    }
    // Reload data with new filters
    if (typeof loadTableData === 'function') {
        showLoadingSkeleton();
        loadTableData();
    }
}

function openSQLModal() {
    document.getElementById('sql-modal').classList.add('show');
}

function closeSQLModal() {
    document.getElementById('sql-modal').classList.remove('show');
}

async function executeSQLQuery() {
    const query = document.getElementById('sql-query').value.trim();
    if (!query) {
        showModal('Validation', 'Please enter a SQL query', 'warning');
        return;
    }

    try {
        const res = await fetch('/api/sql', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query })
        });

        const json = await res.json();

        if (json.success) {
            state.data = json.data;
            renderDataGrid(json.data);
            closeSQLModal();
            showModal('Query executed', `${json.data.rows.length} rows returned`, 'success');
        } else {
            showModal('Error', json.message || 'Failed to execute query', 'error');
        }
    } catch (err) {
        showModal('Error', err.message, 'error');
    }
}

const originalSelectTable = selectTable;
selectTable = async function (tableName) {
    await originalSelectTable(tableName);
    if (state.data && state.data.columns) {
        currentColumns = state.data.columns;
    }
};
function filterIndexItems() {
    const query = document.getElementById('search-tables').value.toLowerCase();
    const items = document.querySelectorAll('#tables-list .table-item');
    items.forEach(item => {
        const name = item.textContent.toLowerCase();
        item.style.display = name.includes(query) ? 'flex' : 'none';
    });
}

function showCreateTableForm() {
    window.location.href = '/schema#create-table';
}

// Branch Management
async function loadBranches() {
    try {
        const response = await fetch('/api/branches');
        const data = await response.json();

        const selector = document.getElementById('branch-selector');
        selector.innerHTML = '';

        if (data.branches.length === 1) {
            selector.style.display = 'none';
            return;
        }

        selector.style.display = 'inline-block';

        data.branches.forEach(branch => {
            const option = document.createElement('option');
            option.value = branch.name;
            option.textContent = `${branch.name}${branch.is_default ? ' (default)' : ''}`;
            if (branch.name === data.current) {
                option.selected = true;
            }
            selector.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load branches:', error);
    }
}

async function switchBranch(branchName) {
    if (!branchName) return;

    try {
        const response = await fetch('/api/branches/switch', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ branch: branchName })
        });

        if (response.ok) {
            showNotification(`Switched to branch: ${branchName}`, 'success');
            location.reload(); // Reload to show data from new branch
        } else {
            showNotification('Failed to switch branch', 'error');
        }
    } catch (error) {
        console.error('Failed to switch branch:', error);
        showNotification('Failed to switch branch', 'error');
    }
}

// Load branches on page load
document.addEventListener('DOMContentLoaded', () => {
    loadBranches();
});

// Show notification toast
function showNotification(message, type = 'info') {
    const existing = document.querySelector('.notification-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.className = `notification-toast notification-${type}`;

    const icons = {
        success: '✓',
        error: '✕',
        warning: '⚠',
        info: 'ℹ'
    };

    toast.innerHTML = `
        <span class="notification-icon">${icons[type] || icons.info}</span>
        <span class="notification-message">${message}</span>
    `;

    document.body.appendChild(toast);

    setTimeout(() => toast.classList.add('show'), 10);
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// Dropdown toggle
function toggleDropdown(dropdownId) {
    const dropdown = document.getElementById(dropdownId);
    const allDropdowns = document.querySelectorAll('.dropdown-menu');

    // Close all other dropdowns
    allDropdowns.forEach(d => {
        if (d.id !== dropdownId) {
            d.classList.remove('show');
        }
    });

    dropdown.classList.toggle('show');
}

// Close dropdown when clicking outside
document.addEventListener('click', (e) => {
    if (!e.target.closest('.dropdown')) {
        document.querySelectorAll('.dropdown-menu').forEach(d => d.classList.remove('show'));
    }
});

// Export database
async function exportDatabase(exportType) {
    // Close dropdown
    document.querySelectorAll('.dropdown-menu').forEach(d => d.classList.remove('show'));

    showNotification('Exporting database...', 'info');

    try {
        const response = await fetch(`/api/export/${exportType}`);
        const responseData = await response.json();

        // Check for error response
        if (responseData.success === false && responseData.message) {
            showNotification(responseData.message, 'error');
            return;
        }

        // Extract the actual export data (unwrap from {success, data} wrapper if present)
        const exportData = responseData.data ? responseData.data : responseData;

        // Create and download JSON file (save only the export data, not the wrapper)
        const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;

        const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
        a.download = `database_export_${exportType}_${timestamp}.json`;

        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        showNotification(`Export completed: ${exportType}`, 'success');
    } catch (err) {
        console.error('Export failed:', err);
        showNotification('Export failed: ' + err.message, 'error');
    }
}

// Trigger import file selection
function triggerImport() {
    document.getElementById('import-file-input').click();
}

// Handle import file selection
async function handleImportFile(event) {
    const file = event.target.files[0];
    if (!file) return;

    // Reset file input for future imports
    event.target.value = '';

    try {
        const content = await file.text();
        let importData;

        try {
            importData = JSON.parse(content);
        } catch (parseErr) {
            showNotification('Invalid JSON file', 'error');
            return;
        }

        // Handle both formats: raw export data OR wrapped with {success, data}
        // If the file has a "data" wrapper (old format), unwrap it
        if (importData.success !== undefined && importData.data) {
            importData = importData.data;
        }

        // Validate import data structure
        if (!importData.version || !importData.tables) {
            showNotification('Invalid export file format', 'error');
            return;
        }

        // Show confirmation dialog with import details
        const tableCount = importData.tables.length;
        const hasSchema = importData.tables.some(t => t.schema);
        const hasData = importData.tables.some(t => t.data && t.data.length > 0);
        const enumCount = importData.enum_types ? importData.enum_types.length : 0;

        let details = `<div style="margin-bottom: 16px;">
            <p><strong>File:</strong> ${file.name}</p>
            <p><strong>Export Type:</strong> ${importData.export_type || 'unknown'}</p>
            <p><strong>Database:</strong> ${importData.database_provider || 'unknown'}</p>
            ${enumCount > 0 ? `<p><strong>Enum Types:</strong> ${enumCount}</p>` : ''}
            <p><strong>Tables:</strong> ${tableCount}</p>
            <p><strong>Contains Schema:</strong> ${hasSchema ? 'Yes' : 'No'}</p>
            <p><strong>Contains Data:</strong> ${hasData ? 'Yes' : 'No'}</p>
        </div>
        ${enumCount > 0 ? `<div style="background: #2a2a2a; padding: 12px; border-radius: 6px; margin-bottom: 16px;">
            <strong>Enum types to create:</strong>
            <ul style="margin: 8px 0 0 20px;">
                ${importData.enum_types.map(e => `<li>${e.name} (${e.values.length} values)</li>`).join('')}
            </ul>
        </div>` : ''}
        <div style="background: #2a2a2a; padding: 12px; border-radius: 6px; margin-bottom: 16px;">
            <strong>Tables to import:</strong>
            <ul style="margin: 8px 0 0 20px; max-height: 150px; overflow-y: auto;">
                ${importData.tables.map(t => `<li>${t.name} ${t.data ? `(${t.data.length} rows)` : '(schema only)'}</li>`).join('')}
            </ul>
        </div>
        <p style="color: #f59e0b;">This will create enum types, tables, and add missing columns/data.</p>`;

        showConfirm('Import Database', details, async () => {
            await performImport(importData);
        });

    } catch (err) {
        console.error('Import failed:', err);
        showNotification('Import failed: ' + err.message, 'error');
    }
}

// Perform the actual import
async function performImport(importData) {
    showNotification('Importing database...', 'info');

    try {
        const response = await fetch('/api/import', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(importData)
        });

        const result = await response.json();

        if (!result.success) {
            showNotification(result.message || 'Import failed', 'error');
            return;
        }

        // Show import results
        const r = result.result;
        let summary = [];

        if (r.enum_types_created && r.enum_types_created.length > 0) {
            summary.push(`Enum types created: ${r.enum_types_created.join(', ')}`);
        }
        if (r.tables_created && r.tables_created.length > 0) {
            summary.push(`Tables created: ${r.tables_created.join(', ')}`);
        }
        if (r.tables_updated && r.tables_updated.length > 0) {
            summary.push(`Tables updated: ${r.tables_updated.join(', ')}`);
        }
        if (r.columns_added > 0) {
            summary.push(`Columns added: ${r.columns_added}`);
        }
        if (r.rows_inserted > 0) {
            summary.push(`Rows inserted: ${r.rows_inserted}`);
        }
        if (r.rows_updated > 0) {
            summary.push(`Rows updated: ${r.rows_updated}`);
        }
        if (r.errors && r.errors.length > 0) {
            summary.push(`<span style="color: #ef4444;">Errors: ${r.errors.length}</span>`);
        }

        const summaryHtml = summary.length > 0
            ? `<ul style="margin: 0; padding-left: 20px;">${summary.map(s => `<li>${s}</li>`).join('')}</ul>`
            : 'No changes were made.';

        showModal('Import Complete', summaryHtml, r.errors && r.errors.length > 0 ? 'warning' : 'success');

        // Refresh the data
        refreshData();
        loadTables();

    } catch (err) {
        console.error('Import failed:', err);
        showNotification('Import failed: ' + err.message, 'error');
    }
}
