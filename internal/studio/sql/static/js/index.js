// Filter state - synced with state object in studio.js
let filters = [];
let currentColumns = [];

// Restore filters from state (called from studio.js on page load)
function restoreFilters(savedFilters) {
    if (!savedFilters || savedFilters.length === 0) return;

    // Clear existing filter rows
    const filterRows = document.getElementById('filter-rows');
    if (!filterRows) return;
    filterRows.innerHTML = '';

    // Rebuild filter rows from saved state
    savedFilters.forEach((filter, index) => {
        const logic = index === 0 ? 'where' : filter.logic;
        addFilterRow(logic, filter.column, filter.operator, filter.value);
    });

    // Apply the filters
    filters = savedFilters;

    // Sync with state
    if (typeof state !== 'undefined' && state.filters) {
        state.filters = savedFilters;
    }

    // Update filter badge count
    updateFilterCount();

    // Apply filters to data if available
    if (typeof state !== 'undefined' && state.data && state.data.rows) {
        applyFilters();
    }
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

    // UUID
    if (type.includes('uuid')) return 'uuid';
    // Numbers
    if (type.includes('int') || type.includes('serial') || type.includes('decimal') ||
        type.includes('numeric') || type.includes('float') || type.includes('double') ||
        type.includes('real') || type.includes('money')) return 'number';
    // Boolean
    if (type.includes('bool')) return 'boolean';
    // Date/Time
    if (type.includes('date') || type.includes('time') || type.includes('timestamp')) return 'datetime';
    // JSON
    if (type.includes('json')) return 'json';
    // Default to text
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

    // Update operators based on column type
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
    if (state.currentTable) {
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

    // Store in state for persistence
    if (typeof state !== 'undefined') {
        state.filters = filters;
        if (typeof state.save === 'function') {
            state.save();
        }
    }

    if (!state.data || !state.data.rows) {
        return;
    }

    if (filters.length === 0) {
        renderDataGrid(state.data);
        document.getElementById('row-count').textContent = `${state.data.rows.length} of ${state.data.total || state.data.rows.length}`;
        return;
    }

    const allRows = state.data.rows;
    let filteredRows = allRows;

    filters.forEach((filter, index) => {
        const { logic, column, operator, value } = filter;
        const colType = getColumnType(column);

        const matchesFilter = (row) => {
            const cellValue = row[column];

            // Handle null checks first
            if (operator === 'is_null') {
                return cellValue === null || cellValue === undefined;
            }
            if (operator === 'is_not_null') {
                return cellValue !== null && cellValue !== undefined;
            }
            if (operator === 'is_empty') {
                return cellValue === null || cellValue === undefined || cellValue === '' ||
                    (Array.isArray(cellValue) && cellValue.length === 0);
            }
            if (operator === 'is_not_empty') {
                return cellValue !== null && cellValue !== undefined && cellValue !== '' &&
                    !(Array.isArray(cellValue) && cellValue.length === 0);
            }

            // For other operators, handle null values
            if (cellValue === null || cellValue === undefined) {
                return false;
            }

            // Type-aware comparison
            let result = false;

            switch (colType) {
                case 'uuid':
                    // UUID comparison - case insensitive exact match
                    const cellUuid = String(cellValue).toLowerCase().trim();
                    const filterUuid = value.toLowerCase().trim();

                    switch (operator) {
                        case 'equals':
                            result = cellUuid === filterUuid;
                            break;
                        case 'not_equals':
                            result = cellUuid !== filterUuid;
                            break;
                        case 'contains':
                            result = cellUuid.includes(filterUuid);
                            break;
                        case 'not_contains':
                            result = !cellUuid.includes(filterUuid);
                            break;
                        case 'starts_with':
                            result = cellUuid.startsWith(filterUuid);
                            break;
                        case 'ends_with':
                            result = cellUuid.endsWith(filterUuid);
                            break;
                        default:
                            result = cellUuid === filterUuid;
                    }
                    break;

                case 'number':
                    const numCell = parseFloat(cellValue);
                    const numFilter = parseFloat(value);

                    if (isNaN(numCell) || isNaN(numFilter)) {
                        // Fall back to string comparison
                        result = String(cellValue).toLowerCase().includes(value.toLowerCase());
                    } else {
                        switch (operator) {
                            case 'equals':
                                result = numCell === numFilter;
                                break;
                            case 'not_equals':
                                result = numCell !== numFilter;
                                break;
                            case 'gt':
                                result = numCell > numFilter;
                                break;
                            case 'lt':
                                result = numCell < numFilter;
                                break;
                            case 'gte':
                                result = numCell >= numFilter;
                                break;
                            case 'lte':
                                result = numCell <= numFilter;
                                break;
                            case 'contains':
                                result = String(numCell).includes(value);
                                break;
                            default:
                                result = numCell === numFilter;
                        }
                    }
                    break;

                case 'boolean':
                    const boolCell = String(cellValue).toLowerCase();
                    const boolFilter = value.toLowerCase();
                    const isTrueCell = boolCell === 'true' || boolCell === '1' || boolCell === 'yes';
                    const isTrueFilter = boolFilter === 'true' || boolFilter === '1' || boolFilter === 'yes';

                    switch (operator) {
                        case 'equals':
                            result = isTrueCell === isTrueFilter;
                            break;
                        case 'not_equals':
                            result = isTrueCell !== isTrueFilter;
                            break;
                        default:
                            result = isTrueCell === isTrueFilter;
                    }
                    break;

                case 'datetime':
                    // Try to parse as date
                    const dateCell = new Date(cellValue);
                    const dateFilter = new Date(value);

                    if (isNaN(dateCell.getTime()) || isNaN(dateFilter.getTime())) {
                        // Fall back to string comparison
                        const strCell = String(cellValue).toLowerCase();
                        const strFilter = value.toLowerCase();
                        result = strCell.includes(strFilter);
                    } else {
                        switch (operator) {
                            case 'equals':
                                result = dateCell.getTime() === dateFilter.getTime();
                                break;
                            case 'not_equals':
                                result = dateCell.getTime() !== dateFilter.getTime();
                                break;
                            case 'gt':
                                result = dateCell.getTime() > dateFilter.getTime();
                                break;
                            case 'lt':
                                result = dateCell.getTime() < dateFilter.getTime();
                                break;
                            case 'gte':
                                result = dateCell.getTime() >= dateFilter.getTime();
                                break;
                            case 'lte':
                                result = dateCell.getTime() <= dateFilter.getTime();
                                break;
                            case 'contains':
                                result = String(cellValue).toLowerCase().includes(value.toLowerCase());
                                break;
                            default:
                                result = dateCell.getTime() === dateFilter.getTime();
                        }
                    }
                    break;

                case 'json':
                    // JSON - stringify and search
                    const jsonStr = typeof cellValue === 'object' ?
                        JSON.stringify(cellValue).toLowerCase() :
                        String(cellValue).toLowerCase();
                    const jsonFilter = value.toLowerCase();

                    switch (operator) {
                        case 'contains':
                            result = jsonStr.includes(jsonFilter);
                            break;
                        case 'not_contains':
                            result = !jsonStr.includes(jsonFilter);
                            break;
                        case 'equals':
                            result = jsonStr === jsonFilter;
                            break;
                        case 'not_equals':
                            result = jsonStr !== jsonFilter;
                            break;
                        default:
                            result = jsonStr.includes(jsonFilter);
                    }
                    break;

                default:
                    // Text comparison
                    const textCell = String(cellValue).toLowerCase();
                    const textFilter = value.toLowerCase();

                    switch (operator) {
                        case 'equals':
                            result = textCell === textFilter;
                            break;
                        case 'not_equals':
                            result = textCell !== textFilter;
                            break;
                        case 'contains':
                            result = textCell.includes(textFilter);
                            break;
                        case 'not_contains':
                            result = !textCell.includes(textFilter);
                            break;
                        case 'starts_with':
                            result = textCell.startsWith(textFilter);
                            break;
                        case 'ends_with':
                            result = textCell.endsWith(textFilter);
                            break;
                        case 'gt':
                            result = textCell > textFilter;
                            break;
                        case 'lt':
                            result = textCell < textFilter;
                            break;
                        case 'gte':
                            result = textCell >= textFilter;
                            break;
                        case 'lte':
                            result = textCell <= textFilter;
                            break;
                        default:
                            result = textCell.includes(textFilter);
                    }
            }

            return result;
        };

        if (index === 0) {
            filteredRows = allRows.filter(matchesFilter);
        } else if (logic === 'and') {
            filteredRows = filteredRows.filter(matchesFilter);
        } else if (logic === 'or') {
            const orMatches = allRows.filter(matchesFilter);
            // Combine and deduplicate
            const seen = new Set();
            filteredRows = [...filteredRows, ...orMatches].filter(row => {
                const key = JSON.stringify(row);
                if (seen.has(key)) return false;
                seen.add(key);
                return true;
            });
        }
    });

    const filteredData = {
        ...state.data,
        rows: filteredRows,
        total: filteredRows.length
    };

    renderDataGrid(filteredData);
    document.getElementById('row-count').textContent = `${filteredRows.length} of ${allRows.length}`;
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
