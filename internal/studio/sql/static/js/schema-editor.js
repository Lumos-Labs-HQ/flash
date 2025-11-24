// Schema Editor Functions
let currentTable = '';
let currentSchemaChange = null;

function openSchemaEditor() {
    const tableName = document.getElementById('current-table').textContent;
    if (tableName === 'Select a model') {
        alert('Please select a table first');
        return;
    }
    currentTable = tableName;
    document.getElementById('schema-modal').classList.add('open');
    updateSchemaForm();
}

function closeSchemaModal() {
    document.getElementById('schema-modal').classList.remove('open');
    document.getElementById('schema-preview').style.display = 'none';
    document.getElementById('apply-schema-btn').style.display = 'none';
}

function updateSchemaForm() {
    const action = document.getElementById('schema-action').value;
    const formDiv = document.getElementById('schema-form');
    
    let html = '';
    
    if (action === 'add_column') {
        html = `
            <div class="form-group">
                <label class="form-label">Column Name</label>
                <input type="text" id="column-name" class="form-input" placeholder="phone_number">
            </div>
            <div class="form-group">
                <label class="form-label">Column Type</label>
                <input type="text" id="column-type" class="form-input" placeholder="VARCHAR(20)">
            </div>
            <div class="form-group">
                <label class="form-label">
                    <input type="checkbox" id="column-nullable" checked> Nullable
                </label>
            </div>
            <div class="form-group">
                <label class="form-label">Default Value (optional)</label>
                <input type="text" id="column-default" class="form-input" placeholder="NULL">
            </div>
        `;
    } else if (action === 'drop_column') {
        html = `
            <div class="form-group">
                <label class="form-label">Column Name to Drop</label>
                <input type="text" id="column-name" class="form-input" placeholder="old_column">
            </div>
        `;
    } else if (action === 'modify_column') {
        html = `
            <div class="form-group">
                <label class="form-label">Column Name</label>
                <input type="text" id="column-name" class="form-input" placeholder="existing_column">
            </div>
            <div class="form-group">
                <label class="form-label">New Type</label>
                <input type="text" id="column-type" class="form-input" placeholder="TEXT">
            </div>
        `;
    }
    
    formDiv.innerHTML = html;
}

async function previewSchemaChange() {
    const action = document.getElementById('schema-action').value;
    const columnName = document.getElementById('column-name')?.value;
    
    if (!columnName) {
        alert('Please enter column name');
        return;
    }
    
    const change = {
        type: action,
        table: currentTable,
        column: {
            name: columnName,
            type: document.getElementById('column-type')?.value || '',
            nullable: document.getElementById('column-nullable')?.checked || false,
            default: document.getElementById('column-default')?.value || ''
        }
    };
    
    try {
        const response = await fetch('/api/schema/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        
        const preview = await response.json();
        
        document.getElementById('preview-sql').textContent = preview.sql;
        document.getElementById('schema-preview').style.display = 'block';
        document.getElementById('apply-schema-btn').style.display = 'inline-block';
        
        currentSchemaChange = change;
    } catch (error) {
        alert('Error previewing change: ' + error.message);
    }
}

async function applySchemaChange() {
    if (!currentSchemaChange) {
        alert('Please preview changes first');
        return;
    }
    
    if (!confirm('This will apply changes to the database. Continue?')) {
        return;
    }
    
    try {
        const response = await fetch('/api/schema/apply', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(currentSchemaChange)
        });
        
        const result = await response.json();
        
        if (result.success) {
            alert('✅ Schema updated successfully!\n\n' +
                  '• Database updated\n' +
                  '• Migration file created\n' +
                  '• Schema file synced');
            closeSchemaModal();
            loadTableData(currentTable);
        } else {
            alert('Error: ' + result.error);
        }
    } catch (error) {
        alert('Error applying change: ' + error.message);
    }
}

// Side Panel Functions
function openSidePanel(rowData, rowId) {
    const panel = document.getElementById('side-panel');
    const content = document.getElementById('panel-content');
    
    let html = '<form id="row-form">';
    html += `<input type="hidden" id="row-id" value="${rowId}">`;
    
    for (const [key, value] of Object.entries(rowData)) {
        html += `
            <div class="form-group">
                <label class="form-label">${key}</label>
                <input type="text" 
                       class="form-input" 
                       name="${key}" 
                       value="${value || ''}"
                       ${key === 'id' ? 'readonly' : ''}>
            </div>
        `;
    }
    
    html += `
        <div class="form-actions">
            <button type="button" class="btn btn-success" onclick="saveRow()">Save</button>
            <button type="button" class="btn btn-danger" onclick="deleteCurrentRow()">Delete</button>
            <button type="button" class="btn btn-secondary" onclick="closeSidePanel()">Cancel</button>
        </div>
    </form>`;
    
    content.innerHTML = html;
    panel.classList.add('open');
}

function closeSidePanel() {
    document.getElementById('side-panel').classList.remove('open');
}

async function saveRow() {
    const form = document.getElementById('row-form');
    const formData = new FormData(form);
    const rowId = document.getElementById('row-id').value;
    
    const data = {};
    for (const [key, value] of formData.entries()) {
        if (key !== 'id') {
            data[key] = value;
        }
    }
    
    try {
        const response = await fetch(`/api/tables/${currentTable}/rows/${rowId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        if (result.success) {
            alert('✅ Row updated successfully!');
            closeSidePanel();
            loadTableData(currentTable);
        } else {
            alert('Error: ' + result.error);
        }
    } catch (error) {
        alert('Error saving row: ' + error.message);
    }
}

async function deleteCurrentRow() {
    if (!confirm('Are you sure you want to delete this row?')) {
        return;
    }
    
    const rowId = document.getElementById('row-id').value;
    
    try {
        const response = await fetch(`/api/tables/${currentTable}/rows/${rowId}`, {
            method: 'DELETE'
        });
        
        const result = await response.json();
        
        if (result.success) {
            alert('✅ Row deleted successfully!');
            closeSidePanel();
            loadTableData(currentTable);
        } else {
            alert('Error: ' + result.error);
        }
    } catch (error) {
        alert('Error deleting row: ' + error.message);
    }
}

// Add click handler to table rows
document.addEventListener('click', function(e) {
    const row = e.target.closest('tr[data-row-id]');
    if (row && !e.target.closest('input[type="checkbox"]')) {
        const rowId = row.getAttribute('data-row-id');
        const cells = row.querySelectorAll('td');
        const headers = document.querySelectorAll('th');
        
        const rowData = {};
        headers.forEach((header, index) => {
            const key = header.textContent.trim();
            const value = cells[index]?.textContent.trim();
            if (key && key !== '☐') {
                rowData[key] = value;
            }
        });
        
        openSidePanel(rowData, rowId);
    }
});
