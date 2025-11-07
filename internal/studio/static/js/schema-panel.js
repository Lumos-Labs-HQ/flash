// Schema Edit Panel
let currentEditTable = '';
let currentAction = '';

// Open edit panel when table is clicked
window.openTableEdit = function(tableName) {
    currentEditTable = tableName;
    document.getElementById('panel-table-name').textContent = tableName;
    document.getElementById('edit-panel').classList.add('open');
    showActionMenu();
};

function closeEditPanel() {
    document.getElementById('edit-panel').classList.remove('open');
    currentEditTable = '';
    currentAction = '';
}

function showActionMenu() {
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <div class="panel-section">
            <div class="section-title">Choose Action</div>
            
            <div class="action-card" onclick="showAddColumn()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:table-column-plus-after" style="color: #10b981;"></span>
                    Add Column
                </div>
                <div class="action-card-desc">Add a new column to this table</div>
            </div>
            
            <div class="action-card" onclick="showDropColumn()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:table-column-remove" style="color: #f59e0b;"></span>
                    Drop Column
                </div>
                <div class="action-card-desc">Remove an existing column</div>
            </div>
            
            <div class="action-card" onclick="showModifyColumn()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:table-edit" style="color: #3b82f6;"></span>
                    Modify Column
                </div>
                <div class="action-card-desc">Change column type or properties</div>
            </div>
        </div>
    `;
}

function showAddColumn() {
    currentAction = 'add_column';
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <button class="back-btn" onclick="showActionMenu()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back
        </button>
        
        <div class="panel-section">
            <div class="section-title">Add New Column</div>
            
            <div class="form-group">
                <label class="form-label">Column Name</label>
                <input type="text" id="column-name" class="form-input" placeholder="phone_number">
            </div>
            
            <div class="form-group">
                <label class="form-label">Data Type</label>
                <select id="column-type" class="form-select">
                    <option value="VARCHAR(255)">VARCHAR(255)</option>
                    <option value="TEXT">TEXT</option>
                    <option value="INTEGER">INTEGER</option>
                    <option value="BIGINT">BIGINT</option>
                    <option value="BOOLEAN">BOOLEAN</option>
                    <option value="TIMESTAMP">TIMESTAMP</option>
                    <option value="DATE">DATE</option>
                    <option value="DECIMAL(10,2)">DECIMAL(10,2)</option>
                </select>
            </div>
            
            <div class="form-group">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-nullable" checked>
                    <span>Nullable</span>
                </label>
            </div>
            
            <div class="form-group">
                <label class="form-label">Default Value (optional)</label>
                <input type="text" id="column-default" class="form-input" placeholder="NULL">
            </div>
            
            <div class="btn-group">
                <button class="btn btn-primary" onclick="previewChange()">Preview</button>
            </div>
        </div>
    `;
}

function showDropColumn() {
    currentAction = 'drop_column';
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <button class="back-btn" onclick="showActionMenu()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back
        </button>
        
        <div class="panel-section">
            <div class="section-title">Drop Column</div>
            
            <div class="form-group">
                <label class="form-label">Column Name</label>
                <input type="text" id="column-name" class="form-input" placeholder="old_column">
            </div>
            
            <div style="background: #dc262620; border: 1px solid #dc2626; border-radius: 6px; padding: 12px; margin: 15px 0; color: #fca5a5; font-size: 13px;">
                <strong>⚠️ Warning:</strong> This action cannot be undone. All data in this column will be lost.
            </div>
            
            <div class="btn-group">
                <button class="btn btn-primary" onclick="previewChange()">Preview</button>
            </div>
        </div>
    `;
}

function showModifyColumn() {
    currentAction = 'modify_column';
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <button class="back-btn" onclick="showActionMenu()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back
        </button>
        
        <div class="panel-section">
            <div class="section-title">Modify Column</div>
            
            <div class="form-group">
                <label class="form-label">Column Name</label>
                <input type="text" id="column-name" class="form-input" placeholder="existing_column">
            </div>
            
            <div class="form-group">
                <label class="form-label">New Data Type</label>
                <select id="column-type" class="form-select">
                    <option value="VARCHAR(255)">VARCHAR(255)</option>
                    <option value="TEXT">TEXT</option>
                    <option value="INTEGER">INTEGER</option>
                    <option value="BIGINT">BIGINT</option>
                    <option value="BOOLEAN">BOOLEAN</option>
                    <option value="TIMESTAMP">TIMESTAMP</option>
                </select>
            </div>
            
            <div class="btn-group">
                <button class="btn btn-primary" onclick="previewChange()">Preview</button>
            </div>
        </div>
    `;
}

async function previewChange() {
    const columnName = document.getElementById('column-name')?.value;
    
    if (!columnName) {
        alert('Please enter column name');
        return;
    }
    
    const change = {
        type: currentAction,
        table: currentEditTable,
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
        showPreview(preview, change);
    } catch (error) {
        alert('Error: ' + error.message);
    }
}

function showPreview(preview, change) {
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <button class="back-btn" onclick="showActionMenu()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back
        </button>
        
        <div class="panel-section">
            <div class="section-title">Preview Changes</div>
            
            <div class="preview-box">${preview.sql}</div>
            
            <div class="section-title" style="margin-top: 20px;">What will happen:</div>
            <ul class="change-list">
                <li>Apply to database immediately</li>
                <li>Create migration file</li>
                <li>Update db/schema/schema.sql</li>
            </ul>
            
            <div class="btn-group">
                <button class="btn btn-primary" onclick='applyChange(${JSON.stringify(change)})'>
                    <span class="iconify" data-icon="mdi:check"></span> Apply Changes
                </button>
                <button class="btn btn-secondary" onclick="showActionMenu()">Cancel</button>
            </div>
        </div>
    `;
}

async function applyChange(change) {
    if (!confirm('Apply these changes to the database?')) {
        return;
    }
    
    try {
        const response = await fetch('/api/schema/apply', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        
        const result = await response.json();
        
        if (result.success) {
            const content = document.getElementById('panel-content');
            content.innerHTML = `
                <div class="panel-section">
                    <div style="text-align: center; padding: 40px 20px;">
                        <div style="font-size: 48px; margin-bottom: 20px;">✅</div>
                        <div style="font-size: 18px; color: #10b981; font-weight: 600; margin-bottom: 10px;">
                            Success!
                        </div>
                        <div style="color: #888; font-size: 14px; margin-bottom: 30px;">
                            Schema updated successfully
                        </div>
                        <ul class="change-list">
                            <li>Database updated</li>
                            <li>Migration file created</li>
                            <li>Schema file synced</li>
                        </ul>
                        <button class="btn btn-primary" onclick="closeEditPanel(); window.location.reload();">
                            Done
                        </button>
                    </div>
                </div>
            `;
        } else {
            alert('Error: ' + result.error);
        }
    } catch (error) {
        alert('Error: ' + error.message);
    }
}
