// Filter state
let filters = [];
let currentColumns = [];

function toggleFilters() {
    const panel = document.getElementById('filter-panel');
    const btn = document.getElementById('filter-btn');
    panel.classList.toggle('show');
    btn.classList.toggle('active');
}

function addFilterRow(logic = 'where', column = '', operator = 'equals', value = '') {
    const row = document.createElement('div');
    row.className = 'filter-row';
    
    const logicSelect = logic === 'where' ? 
        `<select class="filter-logic" disabled><option>where</option></select>` :
        `<select class="filter-logic"><option>and</option><option>or</option></select>`;
    
    const columnOptions = currentColumns.map(col => 
        `<option value="${col.name}" ${col.name === column ? 'selected' : ''}>${col.name}</option>`
    ).join('');
    
    row.innerHTML = `
        ${logicSelect}
        <select class="filter-column">${columnOptions}</select>
        <select class="filter-operator">
            <option value="equals" ${operator === 'equals' ? 'selected' : ''}>equals</option>
            <option value="not_equals" ${operator === 'not_equals' ? 'selected' : ''}>not equals</option>
            <option value="contains" ${operator === 'contains' ? 'selected' : ''}>contains</option>
            <option value="starts_with" ${operator === 'starts_with' ? 'selected' : ''}>starts with</option>
            <option value="ends_with" ${operator === 'ends_with' ? 'selected' : ''}>ends with</option>
            <option value="gt" ${operator === 'gt' ? 'selected' : ''}>greater than</option>
            <option value="lt" ${operator === 'lt' ? 'selected' : ''}>less than</option>
            <option value="gte" ${operator === 'gte' ? 'selected' : ''}>≥</option>
            <option value="lte" ${operator === 'lte' ? 'selected' : ''}>≤</option>
        </select>
        <input type="text" class="filter-value" value="${value}" placeholder="Value">
        <button class="filter-remove" onclick="this.parentElement.remove(); updateFilterCount();">✕</button>
    `;
    
    document.getElementById('filter-rows').appendChild(row);
    updateFilterCount();
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
        
        if (column && value) {
            filters.push({ logic, column, operator, value });
        }
    }
    
    toggleFilters();
    
    if (!state.data || !state.data.rows) {
        return;
    }
    
    const allRows = state.data.rows;
    let filteredRows = allRows;
    
    filters.forEach((filter, index) => {
        const { logic, column, operator, value } = filter;
        
        const matchesFilter = (row) => {
            const cellValue = String(row[column] || '').toLowerCase();
            const filterValue = value.toLowerCase();
            
            let result = false;
            switch (operator) {
                case 'equals':
                    result = cellValue === filterValue;
                    break;
                case 'not_equals':
                    result = cellValue !== filterValue;
                    break;
                case 'contains':
                    result = cellValue.includes(filterValue);
                    break;
                case 'starts_with':
                    result = cellValue.startsWith(filterValue);
                    break;
                case 'ends_with':
                    result = cellValue.endsWith(filterValue);
                    break;
                case 'gt':
                    result = parseFloat(cellValue) > parseFloat(filterValue);
                    break;
                case 'lt':
                    result = parseFloat(cellValue) < parseFloat(filterValue);
                    break;
                case 'gte':
                    result = parseFloat(cellValue) >= parseFloat(filterValue);
                    break;
                case 'lte':
                    result = parseFloat(cellValue) <= parseFloat(filterValue);
                    break;
                default:
                    result = true;
            }
            return result;
        };
        
        if (index === 0) {
            filteredRows = allRows.filter(matchesFilter);
        } else if (logic === 'and') {
            filteredRows = filteredRows.filter(matchesFilter);
        } else if (logic === 'or') {
            const orMatches = allRows.filter(matchesFilter);
            const combined = [...filteredRows, ...orMatches];
            filteredRows = combined.filter((row, idx, self) => 
                idx === self.findIndex(r => JSON.stringify(r) === JSON.stringify(row))
            );
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
selectTable = async function(tableName) {
    await originalSelectTable(tableName);
    if (state.data && state.data.columns) {
        currentColumns = state.data.columns;
    }
};

// Tab management
let currentTab = 'tables';
let allEnums = [];

async function switchIndexTab(tab) {
    currentTab = tab;
    
    // Update tab active state
    document.querySelectorAll('.sidebar-tab').forEach(t => {
        t.classList.remove('active');
    });
    event.target.closest('.sidebar-tab').classList.add('active');
    
    // Show/hide appropriate lists and buttons
    if (tab === 'tables') {
        document.getElementById('tables-list').style.display = 'block';
        document.getElementById('enums-list').style.display = 'none';
        document.getElementById('add-table-btn-container').style.display = 'block';
        document.getElementById('add-enum-btn-container').style.display = 'none';
    } else {
        document.getElementById('tables-list').style.display = 'none';
        document.getElementById('enums-list').style.display = 'block';
        document.getElementById('add-table-btn-container').style.display = 'none';
        document.getElementById('add-enum-btn-container').style.display = 'block';
        await loadEnums();
    }
}

async function loadEnums() {
    try {
        const response = await fetch('/api/schema');
        const json = await response.json();
        console.log('Schema API response:', json);
        if (json.success && json.data) {
            allEnums = json.data.enums || [];
            console.log('Enums loaded:', allEnums);
            renderEnumsList(allEnums);
        }
    } catch (err) {
        console.error('Failed to load enums:', err);
    }
}

function renderEnumsList(enums) {
    const list = document.getElementById('enums-list');
    if (!enums || enums.length === 0) {
        list.innerHTML = '<div style="padding: 20px; text-align: center; color: #888;">No enums found</div>';
        return;
    }
    
    list.innerHTML = enums.map(e => {
        const name = e.name || e.Name || 'Unknown';
        const values = e.values || e.Values || [];
        return `
            <div class="table-item" onclick="showEnumDetailsIndex('${name}')">
                <span class="table-item-name">${name}</span>
                <span class="table-count">${values.length} values</span>
            </div>
        `;
    }).join('');
}

function filterIndexItems() {
    const query = document.getElementById('search-tables').value.toLowerCase();
    
    if (currentTab === 'tables') {
        const items = document.querySelectorAll('#tables-list .table-item');
        items.forEach(item => {
            const name = item.textContent.toLowerCase();
            item.style.display = name.includes(query) ? 'flex' : 'none';
        });
    } else {
        const items = document.querySelectorAll('#enums-list .table-item');
        items.forEach(item => {
            const name = item.textContent.toLowerCase();
            item.style.display = name.includes(query) ? 'flex' : 'none';
        });
    }
}

function showEnumDetailsIndex(enumName) {
    const enumData = allEnums.find(e => (e.name || e.Name) === enumName);
    if (!enumData) return;

    const name = enumData.name || enumData.Name || 'Unknown';
    const values = enumData.values || enumData.Values || [];

    // Render into main data area (grid-container) instead of modal
    const container = document.getElementById('grid-container');
    container.innerHTML = `
        <div class="table-schema-info">
            <div class="schema-title">Enum: ${name}</div>
            <div class="schema-columns">
                <div class="schema-column" style="grid-column: 1 / -1;">
                    <div style="display:flex; justify-content:space-between; align-items:center;">
                        <div style="font-weight:600;">Values (${values.length})</div>
                        <div>
                            <button class="btn btn-secondary" onclick="editEnumIndex('${name}')">Edit</button>
                            <button class="btn btn-danger" onclick="deleteEnumIndex('${name}')" style="margin-left:8px;">Delete</button>
                        </div>
                    </div>
                    <div style="margin-top:10px; display:flex; flex-direction:column; gap:6px;">
                        ${values.map(v => `<div style="padding:8px; background:#2a2a2a; border-radius:4px;">${v}</div>`).join('')}
                    </div>
                </div>
            </div>
        </div>
    `;
}

function showCreateTableForm() {
    window.location.href = '/schema#create-table';
}

function showCreateEnumForm() {
    // Show a simple create form in the main area
    const container = document.getElementById('grid-container');
    container.innerHTML = `
        <div class="panel-section" style="padding:20px;">
            <div class="section-title">Create New Enum</div>
            <div class="form-group">
                <label class="form-label">Enum Name</label>
                <input type="text" id="idx-enum-name" class="form-input" placeholder="status, role, priority...">
            </div>
            <div class="form-group">
                <label class="form-label">Values (one per line)</label>
                <textarea id="idx-enum-values" class="form-input" rows="8" placeholder="active\ninactive\npending"></textarea>
            </div>
            <div style="margin-top:12px;">
                <button class="btn btn-primary" onclick="createEnumIndex()">Create Enum</button>
                <button class="btn btn-secondary" onclick="location.reload()" style="margin-left:8px;">Cancel</button>
            </div>
        </div>
    `;
}

async function createEnumIndex() {
    const enumName = document.getElementById('idx-enum-name').value.trim();
    const valuesText = document.getElementById('idx-enum-values').value.trim();
    if (!enumName) return showModal('Validation', 'Please enter enum name', 'warning');
    if (!valuesText) return showModal('Validation', 'Please enter enum values', 'warning');

    const values = valuesText.split('\n').map(v => v.trim()).filter(v => v);
    const change = { type: 'create_enum', enum_name: enumName, enum_values: values };

    try {
        // Preview SQL
        const res = await fetch('/api/schema/preview', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(change) });
        const preview = await res.json();
        const message = `<div style="text-align:left;"><pre style="white-space:pre-wrap; background:#0f0f0f; padding:10px; border-radius:4px;">${preview.sql}</pre></div>`;
        showConfirm('Preview Create Enum', message, async () => {
            // Apply
            const applyRes = await fetch('/api/schema/apply', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(change) });
            const result = await applyRes.json();
            if (result.success) {
                showModal('Success', 'Enum created successfully', 'success');
                await loadEnums();
                // Show new enum
                renderEnumsList(allEnums);
            } else {
                showModal('Error', result.error || 'Failed to create enum', 'error');
            }
        });
    } catch (err) {
        console.error(err);
        showModal('Error', err.message || 'Failed to preview', 'error');
    }
}

function editEnumIndex(enumName) {
    const enumData = allEnums.find(e => (e.name || e.Name) === enumName);
    if (!enumData) return;
    const name = enumData.name || enumData.Name || enumName;
    const values = enumData.values || enumData.Values || [];

    const container = document.getElementById('grid-container');
    container.innerHTML = `
        <div class="panel-section" style="padding:20px;">
            <div class="section-title">Edit Enum: ${name}</div>
            <div class="form-group">
                <label class="form-label">Values (one per line)</label>
                <textarea id="idx-edit-enum-values" class="form-input" rows="10">${values.join('\n')}</textarea>
            </div>
            <div style="margin-top:12px;">
                <button class="btn btn-primary" onclick="updateEnumIndex('${name}')">Update Enum</button>
                <button class="btn btn-secondary" onclick="loadEnums(); location.reload();" style="margin-left:8px;">Cancel</button>
            </div>
        </div>
    `;
}

async function updateEnumIndex(enumName) {
    const valuesText = document.getElementById('idx-edit-enum-values').value.trim();
    if (!valuesText) return showModal('Validation', 'Please enter enum values', 'warning');
    const values = valuesText.split('\n').map(v => v.trim()).filter(v => v);
    const change = { type: 'alter_enum', enum_name: enumName, enum_values: values };

    try {
        const res = await fetch('/api/schema/preview', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(change) });
        const preview = await res.json();
        const message = `<div style="text-align:left;"><pre style="white-space:pre-wrap; background:#0f0f0f; padding:10px; border-radius:4px;">${preview.sql}</pre></div>`;
        showConfirm('Preview Update Enum', message, async () => {
            const applyRes = await fetch('/api/schema/apply', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(change) });
            const result = await applyRes.json();
            if (result.success) {
                showModal('Success', 'Enum updated successfully', 'success');
                await loadEnums();
                renderEnumsList(allEnums);
            } else {
                showModal('Error', result.error || 'Failed to update enum', 'error');
            }
        });
    } catch (err) {
        console.error(err);
        showModal('Error', err.message || 'Failed to preview', 'error');
    }
}

async function deleteEnumIndex(enumName) {
    const change = { type: 'drop_enum', enum_name: enumName };
    try {
        const res = await fetch('/api/schema/preview', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(change) });
        const preview = await res.json();
        const message = `<div style="text-align:left; color:#f59e0b;">Deleting an enum may break columns that use it.<pre style="white-space:pre-wrap; background:#0f0f0f; padding:10px; border-radius:4px;">${preview.sql}</pre></div>`;
        showConfirm('Delete Enum', message, async () => {
            const applyRes = await fetch('/api/schema/apply', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(change) });
            const result = await applyRes.json();
            if (result.success) {
                showModal('Success', 'Enum deleted successfully', 'success');
                await loadEnums();
                renderEnumsList(allEnums);
                document.getElementById('grid-container').innerHTML = '';
            } else {
                showModal('Error', result.error || 'Failed to delete enum', 'error');
            }
        });
    } catch (err) {
        console.error(err);
        showModal('Error', err.message || 'Failed to preview', 'error');
    }
}
