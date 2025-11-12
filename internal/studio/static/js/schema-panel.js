// Schema Edit Panel
let currentEditTable = '';
let currentAction = '';
let currentTableData = null;

// Custom Modal Functions
function showModal(title, message, onConfirm, isDanger = false) {
    const modal = document.createElement('div');
    modal.className = 'custom-modal';
    modal.innerHTML = `
        <div class="modal-overlay"></div>
        <div class="modal-content">
            <div class="modal-header ${isDanger ? 'modal-danger' : ''}">
                <h3>${title}</h3>
            </div>
            <div class="modal-body">
                <p>${message}</p>
            </div>
            <div class="modal-footer">
                <button class="modal-btn modal-btn-cancel" onclick="this.closest('.custom-modal').remove()">Cancel</button>
                <button class="modal-btn ${isDanger ? 'modal-btn-danger' : 'modal-btn-primary'}" id="modal-confirm">
                    ${isDanger ? 'Delete' : 'Confirm'}
                </button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
    
    document.getElementById('modal-confirm').onclick = () => {
        modal.remove();
        if (onConfirm) onConfirm();
    };
    
    modal.querySelector('.modal-overlay').onclick = () => modal.remove();
}

function showAlert(title, message, type = 'info') {
    const modal = document.createElement('div');
    modal.className = 'custom-modal';
    const icon = type === 'error' ? '❌' : type === 'success' ? '✅' : 'ℹ️';
    modal.innerHTML = `
        <div class="modal-overlay"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h3>${icon} ${title}</h3>
            </div>
            <div class="modal-body">
                <p>${message}</p>
            </div>
            <div class="modal-footer">
                <button class="modal-btn modal-btn-primary" onclick="this.closest('.custom-modal').remove()">OK</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
    modal.querySelector('.modal-overlay').onclick = () => modal.remove();
}

// Open edit panel when table is clicked
window.openTableEdit = async function(tableName) {
    currentEditTable = tableName;
    document.getElementById('panel-table-name').textContent = tableName;
    document.getElementById('edit-panel').classList.add('open');
    
    // Fetch table data
    try {
        const response = await fetch('/api/schema');
        const data = await response.json();
        if (data.success && data.data) {
            const tableNode = data.data.nodes.find(n => n.data.label === tableName);
            if (tableNode) {
                currentTableData = tableNode.data;
                showTableDetails();
            }
        }
    } catch (error) {
        console.error('Failed to fetch table data:', error);
        showActionMenu();
    }
};

function closeEditPanel() {
    document.getElementById('edit-panel').classList.remove('open');
    currentEditTable = '';
    currentAction = '';
    currentTableData = null;
}

function showTableDetails() {
    const content = document.getElementById('panel-content');
    
    if (!currentTableData || !currentTableData.columns) {
        showActionMenu();
        return;
    }
    
    const columnsHTML = currentTableData.columns.map(col => {
        const nullable = col.nullable ? '<span class="badge badge-success">NULL</span>' : '<span class="badge badge-warning">NOT NULL</span>';
        const defaultValue = col.default ? `<div class="col-detail"><strong>Default:</strong> ${col.default}</div>` : '';
        const foreignKey = col.isForeign && col.foreignKeyTable ? 
            `<div class="col-detail"><strong>References:</strong> ${col.foreignKeyTable}.${col.foreignKeyColumn || 'id'}</div>` : '';
        const primaryKey = col.isPrimary ? '<span class="badge badge-primary">PRIMARY KEY</span>' : '';
        const unique = col.isUnique ? '<span class="badge badge-info">UNIQUE</span>' : '';
        const autoIncrement = col.isAutoIncrement ? '<span class="badge badge-accent">AUTO INCREMENT</span>' : '';
        
        let icon = '•';
        if (col.isPrimary) icon = '🔑';
        else if (col.isForeign) icon = '🔗';
        else if (col.isUnique) icon = '⚡';
        
        return `
            <div class="column-item">
                <div class="column-header">
                    <div class="column-left">
                        <span class="column-icon">${icon}</span>
                        <span class="column-name">${col.name}</span>
                        <span class="column-type">${col.type}</span>
                    </div>
                    <div class="column-right">
                        ${!col.isPrimary ? `<button class="icon-btn delete-btn" onclick="deleteColumn('${col.name}')" title="Delete column">
                            <span class="iconify" data-icon="mdi:delete"></span>
                        </button>` : ''}
                    </div>
                </div>
                <div class="column-details">
                    <div class="col-badges">
                        ${nullable}
                        ${primaryKey}
                        ${unique}
                        ${autoIncrement}
                    </div>
                    ${defaultValue}
                    ${foreignKey}
                </div>
            </div>
        `;
    }).join('');
    
    content.innerHTML = `
        <div class="panel-section">
            <div class="section-title">Table: ${currentTableData.label}</div>
            <div class="table-stats">
                ${currentTableData.columns.length} columns
            </div>
        </div>
        
        <div class="panel-section">
            <div class="section-title">Columns</div>
            <div class="columns-list">
                ${columnsHTML}
            </div>
        </div>
        
        <div class="panel-section">
            <div class="section-title">Column Actions</div>
            
            <div class="action-card" onclick="showAddColumn()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:table-column-plus-after" style="color: #10b981;"></span>
                    Add Column
                </div>
            </div>
            
            <div class="action-card" onclick="showModifyColumn()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:table-edit" style="color: #3b82f6;"></span>
                    Modify Column
                </div>
            </div>
        </div>
        
        <div class="panel-section">
            <div class="section-title">Enum Management</div>
            
            <div class="action-card" onclick="showCreateEnum()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:format-list-bulleted-square" style="color: #10b981;"></span>
                    Create Enum
                </div>
            </div>
            
            <div class="action-card" onclick="showManageEnums()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:format-list-bulleted" style="color: #a855f7;"></span>
                    Manage Enums
                </div>
            </div>
        </div>
        
        <div class="panel-section">
            <div class="section-title">Danger Zone</div>
            
            <div class="action-card action-card-danger" onclick="deleteTable()">
                <div class="action-card-title">
                    <span class="iconify" data-icon="mdi:table-remove" style="color: #dc2626;"></span>
                    Delete Table
                </div>
            </div>
        </div>
    `;
}

async function deleteColumn(columnName) {
    showModal(
        'Delete Column',
        `Are you sure you want to delete column <strong>"${columnName}"</strong> from table <strong>"${currentEditTable}"</strong>?<br><br>⚠️ <strong>This action cannot be undone!</strong> All data in this column will be lost.`,
        async () => {
            const change = {
                type: 'drop_column',
                table: currentEditTable,
                column: { name: columnName }
            };
            
            try {
                const response = await fetch('/api/schema/preview', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(change)
                });
                
                const preview = await response.json();
                currentAction = 'drop_column';
                await showPreview(preview, change);
            } catch (error) {
                showAlert('Error', 'Failed to delete column: ' + error.message, 'error');
            }
        },
        true
    );
}

async function deleteTable() {
    showModal(
        'Delete Table',
        `Are you sure you want to delete the entire table <strong>"${currentEditTable}"</strong>?<br><br>⚠️ <strong>THIS IS EXTREMELY DANGEROUS!</strong><br>All data and structure will be permanently lost!`,
        async () => {
            const change = {
                type: 'drop_table',
                table: currentEditTable
            };
            
            try {
                const response = await fetch('/api/schema/preview', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(change)
                });
                
                const preview = await response.json();
                currentAction = 'drop_table';
                await showPreview(preview, change);
            } catch (error) {
                showAlert('Error', 'Failed to delete table: ' + error.message, 'error');
            }
        },
        true
    );
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

async function showAddColumn() {
    currentAction = 'add_column';
    
    // Fetch schema to get available tables and enums
    let tables = [];
    let enums = [];
    try {
        const response = await fetch('/api/schema');
        const data = await response.json();
        if (data.success && data.data) {
            tables = data.data.nodes || [];
            enums = data.data.enums || [];
        }
    } catch (error) {
        console.error('Failed to fetch schema:', error);
    }
    
    const content = document.getElementById('panel-content');
    
    // Build enum options
    let enumOptions = '';
    if (enums.length > 0) {
        enumOptions = enums.map(e => `<option value="${e.name}">${e.name}</option>`).join('');
    }
    
    // Build table options for foreign keys
    const tableOptions = tables
        .filter(t => t.data.label !== currentEditTable)
        .map(t => `<option value="${t.data.label}">${t.data.label}</option>`)
        .join('');
    
    content.innerHTML = `
        <button class="back-btn" onclick="showTableDetails()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back to Table
        </button>
        
        <div class="panel-section">
            <div class="section-title">Add New Column</div>
            
            <div class="form-group">
                <label class="form-label">Column Name</label>
                <input type="text" id="column-name" class="form-input" placeholder="phone_number">
            </div>
            
            <div class="form-group">
                <label class="form-label">Data Type</label>
                <select id="column-type" class="form-select" onchange="handleTypeChange()">
                    <optgroup label="String Types">
                        <option value="VARCHAR(255)">VARCHAR(255)</option>
                        <option value="VARCHAR(100)">VARCHAR(100)</option>
                        <option value="VARCHAR(500)">VARCHAR(500)</option>
                        <option value="TEXT">TEXT</option>
                        <option value="CHAR(10)">CHAR(10)</option>
                    </optgroup>
                    <optgroup label="Numeric Types">
                        <option value="INTEGER">INTEGER</option>
                        <option value="BIGINT">BIGINT</option>
                        <option value="SMALLINT">SMALLINT</option>
                        <option value="DECIMAL(10,2)">DECIMAL(10,2)</option>
                        <option value="NUMERIC(10,2)">NUMERIC(10,2)</option>
                        <option value="FLOAT">FLOAT</option>
                        <option value="DOUBLE PRECISION">DOUBLE PRECISION</option>
                        <option value="REAL">REAL</option>
                    </optgroup>
                    <optgroup label="Date/Time Types">
                        <option value="TIMESTAMP">TIMESTAMP</option>
                        <option value="TIMESTAMPTZ">TIMESTAMP WITH TIME ZONE</option>
                        <option value="DATE">DATE</option>
                        <option value="TIME">TIME</option>
                    </optgroup>
                    <optgroup label="Boolean">
                        <option value="BOOLEAN">BOOLEAN</option>
                    </optgroup>
                    <optgroup label="JSON">
                        <option value="JSON">JSON</option>
                        <option value="JSONB">JSONB</option>
                    </optgroup>
                    <optgroup label="Binary">
                        <option value="BYTEA">BYTEA</option>
                    </optgroup>
                    <optgroup label="UUID">
                        <option value="UUID">UUID</option>
                    </optgroup>
                    ${enumOptions ? '<optgroup label="Enum"><option value="ENUM">ENUM (Custom)</option></optgroup>' : ''}
                </select>
            </div>
            
            <div id="enum-section" style="display: none;">
                <div class="form-group">
                    <label class="form-label">Select Enum Type</label>
                    <select id="column-enum" class="form-select" onchange="handleEnumSelection()">
                        <option value="" disabled selected>-- Select Enum --</option>
                        ${enumOptions}
                        <option value="__CREATE_NEW__">+ Create New Enum</option>
                    </select>
                </div>
            </div>
            
            <div class="form-group">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-nullable" checked>
                    <span>Nullable</span>
                </label>
            </div>
            
            <div class="form-group">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-unique">
                    <span>⚡Unique Constraint</span>
                </label>
            </div>
            
            <div class="form-group">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-auto-increment">
                    <span>🔢Auto Increment</span>
                </label>
            </div>
            
            <div class="form-group">
                <label class="form-label">Default Value (optional)</label>
                <div id="default-value-container">
                    <input type="text" id="column-default" class="form-input" placeholder="NULL or expression">
                    <div style="font-size: 11px; color: #888; margin-top: 4px;">
                        Examples: NULL, 0, 'value', NOW(), gen_random_uuid()
                    </div>
                </div>
                <div id="enum-default-container" style="display: none;">
                    <select id="column-enum-default" class="form-select">
                        <option value="">-- No Default --</option>
                    </select>
                    <div style="font-size: 11px; color: #888; margin-top: 4px;">
                        Select a default value from the enum
                    </div>
                </div>
            </div>
            
            <div class="form-group" style="margin-top: 20px; padding-top: 20px; border-top: 1px solid #444;">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-is-foreign" onchange="toggleForeignKeySection()">
                    <span>🔗Foreign Key Relationship</span>
                </label>
            </div>
            
            <div id="foreign-key-section" style="display: none;">
                <div class="form-group">
                    <label class="form-label">References Table</label>
                    <select id="foreign-table" class="form-select">
                        <option value="">-- Select Table --</option>
                        ${tableOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label class="form-label">References Column</label>
                    <input type="text" id="foreign-column" class="form-input" placeholder="id" value="id">
                </div>
            </div>
            
            <div class="btn-group">
                <button class="btn btn-primary" onclick="previewChange()">Preview</button>
            </div>
        </div>
    `;
    
    // Store enums in global scope for later use
    window.currentEnums = enums;
}

function handleTypeChange() {
    const typeSelect = document.getElementById('column-type');
    const enumSection = document.getElementById('enum-section');
    const defaultContainer = document.getElementById('default-value-container');
    const enumDefaultContainer = document.getElementById('enum-default-container');
    
    if (typeSelect && enumSection) {
        // Remove any inline enum creation forms
        const inlineForm = document.querySelector('.inline-enum-form');
        if (inlineForm) {
            inlineForm.remove();
        }
        
        if (typeSelect.value === 'ENUM') {
            enumSection.style.display = 'block';
            if (defaultContainer && enumDefaultContainer) {
                defaultContainer.style.display = 'none';
                enumDefaultContainer.style.display = 'none'; // Will show after enum selection
            }
        } else {
            enumSection.style.display = 'none';
            if (defaultContainer && enumDefaultContainer) {
                defaultContainer.style.display = 'block';
                enumDefaultContainer.style.display = 'none';
            }
        }
    }
}

async function handleEnumSelection() {
    const enumSelect = document.getElementById('column-enum');
    const enumDefaultContainer = document.getElementById('enum-default-container');
    const enumDefaultSelect = document.getElementById('column-enum-default');
    
    if (!enumSelect || !enumDefaultContainer || !enumDefaultSelect) return;
    
    // Remove any existing inline enum forms first
    const existingForm = document.querySelector('.inline-enum-form');
    if (existingForm) {
        existingForm.remove();
    }
    
    const selectedValue = enumSelect.value;
    
    if (selectedValue === '__CREATE_NEW__') {
        // Show inline create enum form
        showInlineCreateEnum();
    } else if (selectedValue) {
        // Show default value dropdown with enum values
        const enums = window.currentEnums || [];
        const selectedEnum = enums.find(e => e.name === selectedValue);
        
        if (selectedEnum && selectedEnum.values) {
            // Populate default value dropdown
            enumDefaultSelect.innerHTML = '<option value="">-- No Default --</option>' +
                selectedEnum.values.map(v => `<option value="'${v}'">${v}</option>`).join('');
            enumDefaultContainer.style.display = 'block';
        }
    } else {
        enumDefaultContainer.style.display = 'none';
    }
}

function showInlineCreateEnum() {
    const enumSection = document.getElementById('enum-section');
    if (!enumSection) return;
    
    enumSection.innerHTML = `
        <div class="form-group">
            <label class="form-label">Select Enum Type</label>
            <select id="column-enum" class="form-select" onchange="handleEnumSelection()">
                <option value="" disabled>-- Select Enum --</option>
                ${window.currentEnums ? window.currentEnums.map(e => `<option value="${e.name}">${e.name}</option>`).join('') : ''}
                <option value="__CREATE_NEW__" selected>+ Create New Enum</option>
            </select>
        </div>
        <div style="background: #2a2a2a; padding: 15px; border-radius: 6px; margin-top: 10px;">
            <div style="font-weight: 600; margin-bottom: 10px; color: #10b981;">Create New Enum</div>
            <div class="form-group">
                <label class="form-label">Enum Name</label>
                <input type="text" id="inline-enum-name" class="form-input" placeholder="status, role, priority...">
            </div>
            <div class="form-group">
                <label class="form-label">Values (one per line)</label>
                <textarea id="inline-enum-values" class="form-input" rows="5" placeholder="active&#10;inactive&#10;pending"></textarea>
            </div>
            <div style="display: flex; gap: 8px;">
                <button class="btn btn-primary" onclick="createInlineEnum()" style="flex: 1;">Create &amp; Use</button>
                <button class="btn btn-secondary" onclick="cancelInlineEnum()">Cancel</button>
            </div>
        </div>
    `;
}

async function createInlineEnum() {
    const enumName = document.getElementById('inline-enum-name')?.value.trim();
    const valuesText = document.getElementById('inline-enum-values')?.value.trim();
    
    if (!enumName) {
        showModal('Validation', 'Please enter enum name', 'warning');
        return;
    }
    if (!valuesText) {
        showModal('Validation', 'Please enter enum values', 'warning');
        return;
    }
    
    const values = valuesText.split('\n').map(v => v.trim()).filter(v => v);
    if (values.length === 0) {
        showModal('Validation', 'Please enter at least one enum value', 'warning');
        return;
    }
    
    const change = {
        type: 'create_enum',
        enum_name: enumName,
        enum_values: values
    };
    
    try {
        // Preview
        const previewRes = await fetch('/api/schema/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        const preview = await previewRes.json();
        
        const message = `<div style="text-align:left;"><pre style="white-space:pre-wrap; background:#0f0f0f; padding:10px; border-radius:4px;">${preview.sql}</pre></div>`;
        
        showConfirm('Create Enum', message, async () => {
            // Apply
            const applyRes = await fetch('/api/schema/apply', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(change)
            });
            const result = await applyRes.json();
            
            if (result.success) {
                showModal('Success', 'Enum created successfully', 'success');
                
                // Remove the inline form
                const inlineForm = document.querySelector('.inline-enum-form');
                if (inlineForm) {
                    inlineForm.remove();
                }
                
                // Refresh the enum list in the dropdown
                const response = await fetch('/api/schema');
                const data = await response.json();
                if (data.success && data.data) {
                    window.currentEnums = data.data.enums || [];
                    
                    // Update the enum dropdown
                    const enumSelect = document.getElementById('column-enum');
                    if (enumSelect) {
                        const enumOptions = window.currentEnums.map(e => 
                            `<option value="${e.name}">${e.name}</option>`
                        ).join('');
                        enumSelect.innerHTML = `
                            <option value="">-- Select Enum --</option>
                            ${enumOptions}
                            <option value="__CREATE_NEW__">+ Create New Enum</option>
                        `;
                        
                        // Select the newly created enum
                        enumSelect.value = enumName;
                        handleEnumSelection();
                    }
                }
            } else {
                showModal('Error', result.error || 'Failed to create enum', 'error');
            }
        });
    } catch (err) {
        console.error(err);
        showModal('Error', err.message || 'Failed to create enum', 'error');
    }
}

function cancelInlineEnum() {
    const enumSection = document.getElementById('enum-section');
    if (!enumSection || !window.currentEnums) return;
    
    const enumOptions = window.currentEnums.map(e => `<option value="${e.name}">${e.name}</option>`).join('');
    enumSection.innerHTML = `
        <div class="form-group">
            <label class="form-label">Select Enum Type</label>
            <select id="column-enum" class="form-select" onchange="handleEnumSelection()">
                <option value="" disabled selected>-- Select Enum --</option>
                ${enumOptions}
                <option value="__CREATE_NEW__">+ Create New Enum</option>
            </select>
        </div>
    `;
}

function toggleForeignKeySection() {
    const checkbox = document.getElementById('column-is-foreign');
    const section = document.getElementById('foreign-key-section');
    
    if (checkbox && section) {
        section.style.display = checkbox.checked ? 'block' : 'none';
    }
}

function showDropColumn() {
    currentAction = 'drop_column';
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <button class="back-btn" onclick="showTableDetails()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back to Table
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

async function showModifyColumn() {
    currentAction = 'modify_column';
    
    // Fetch schema to get available tables and enums
    let tables = [];
    let enums = [];
    try {
        const response = await fetch('/api/schema');
        const data = await response.json();
        if (data.success && data.data) {
            tables = data.data.nodes || [];
            enums = data.data.enums || [];
        }
    } catch (error) {
        console.error('Failed to fetch schema:', error);
    }
    
    // Find current table's columns
    const currentTable = tables.find(t => t.data.label === currentEditTable);
    const columns = currentTable ? currentTable.data.columns : [];
    
    const content = document.getElementById('panel-content');
    
    // Build enum options
    let enumOptions = '';
    if (enums.length > 0) {
        enumOptions = enums.map(e => `<option value="${e.name}">${e.name}</option>`).join('');
    }
    
    // Build column dropdown
    const columnOptions = columns.map(col => 
        `<option value="${col.name}" data-type="${col.type}" data-nullable="${col.nullable}" data-default="${col.default || ''}" data-unique="${col.isUnique || false}" data-auto-increment="${col.isAutoIncrement || false}">${col.name}</option>`
    ).join('');
    
    // Build table options for foreign keys
    const tableOptions = tables
        .filter(t => t.data.label !== currentEditTable)
        .map(t => `<option value="${t.data.label}">${t.data.label}</option>`)
        .join('');
    
    content.innerHTML = `
        <button class="back-btn" onclick="showTableDetails()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back to Table
        </button>
        
        <div class="panel-section">
            <div class="section-title">Modify Column</div>
            
            <div class="form-group">
                <label class="form-label">Select Column to Modify</label>
                <select id="column-selector" class="form-select" onchange="loadColumnData()">
                    <option value="">-- Select Column --</option>
                    ${columnOptions}
                </select>
            </div>
            
            <div class="form-group">
                <label class="form-label">Column Name</label>
                <input type="text" id="column-name" class="form-input" placeholder="column_name" readonly style="background: #2a2a2a;">
                <div style="font-size: 11px; color: #888; margin-top: 4px;">
                    Column name cannot be changed directly. Use rename operation if needed.
                </div>
            </div>
            
            <div class="form-group">
                <label class="form-label">New Data Type</label>
                <select id="column-type" class="form-select" onchange="handleTypeChange()">
                    <optgroup label="String Types">
                        <option value="VARCHAR(255)">VARCHAR(255)</option>
                        <option value="VARCHAR(100)">VARCHAR(100)</option>
                        <option value="VARCHAR(500)">VARCHAR(500)</option>
                        <option value="TEXT">TEXT</option>
                        <option value="CHAR(10)">CHAR(10)</option>
                    </optgroup>
                    <optgroup label="Numeric Types">
                        <option value="INTEGER">INTEGER</option>
                        <option value="BIGINT">BIGINT</option>
                        <option value="SMALLINT">SMALLINT</option>
                        <option value="DECIMAL(10,2)">DECIMAL(10,2)</option>
                        <option value="NUMERIC(10,2)">NUMERIC(10,2)</option>
                        <option value="FLOAT">FLOAT</option>
                        <option value="DOUBLE PRECISION">DOUBLE PRECISION</option>
                        <option value="REAL">REAL</option>
                    </optgroup>
                    <optgroup label="Date/Time Types">
                        <option value="TIMESTAMP">TIMESTAMP</option>
                        <option value="TIMESTAMPTZ">TIMESTAMP WITH TIME ZONE</option>
                        <option value="DATE">DATE</option>
                        <option value="TIME">TIME</option>
                    </optgroup>
                    <optgroup label="Boolean">
                        <option value="BOOLEAN">BOOLEAN</option>
                    </optgroup>
                    <optgroup label="JSON">
                        <option value="JSON">JSON</option>
                        <option value="JSONB">JSONB</option>
                    </optgroup>
                    <optgroup label="Binary">
                        <option value="BYTEA">BYTEA</option>
                    </optgroup>
                    <optgroup label="UUID">
                        <option value="UUID">UUID</option>
                    </optgroup>
                    ${enumOptions ? '<optgroup label="Enum"><option value="ENUM">ENUM (Custom)</option></optgroup>' : ''}
                </select>
            </div>
            
            <div id="enum-section" style="display: none;">
                <div class="form-group">
                    <label class="form-label">Select Enum Type</label>
                    <select id="column-enum" class="form-select" onchange="handleEnumSelection()">
                        <option value="" disabled selected>-- Select Enum --</option>
                        ${enumOptions}
                        <option value="__CREATE_NEW__">+ Create New Enum</option>
                    </select>
                </div>
            </div>
            
            <div class="form-group">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-nullable" checked>
                    <span>Nullable</span>
                </label>
            </div>
            
            <div class="form-group">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-unique">
                    <span>⚡Unique Constraint</span>
                </label>
            </div>
            
            <div class="form-group">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-auto-increment">
                    <span>🔢Auto Increment</span>
                </label>
            </div>
            
            <div class="form-group">
                <label class="form-label">Default Value (optional)</label>
                <div id="default-value-container">
                    <input type="text" id="column-default" class="form-input" placeholder="NULL or expression">
                    <div style="font-size: 11px; color: #888; margin-top: 4px;">
                        Examples: NULL, 0, 'value', NOW(), gen_random_uuid()
                    </div>
                </div>
                <div id="enum-default-container" style="display: none;">
                    <select id="column-enum-default" class="form-select">
                        <option value="">-- No Default --</option>
                    </select>
                    <div style="font-size: 11px; color: #888; margin-top: 4px;">
                        Select a default value from the enum
                    </div>
                </div>
            </div>
            
            <div class="form-group" style="margin-top: 20px; padding-top: 20px; border-top: 1px solid #444;">
                <label class="form-checkbox">
                    <input type="checkbox" id="column-is-foreign" onchange="toggleForeignKeySection()">
                    <span>🔗Foreign Key Relationship</span>
                </label>
            </div>
            
            <div id="foreign-key-section" style="display: none;">
                <div class="form-group">
                    <label class="form-label">References Table</label>
                    <select id="foreign-table" class="form-select">
                        <option value="">-- Select Table --</option>
                        ${tableOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label class="form-label">References Column</label>
                    <input type="text" id="foreign-column" class="form-input" placeholder="id" value="id">
                </div>
            </div>
            
            <div class="btn-group">
                <button class="btn btn-primary" onclick="previewChange()">Preview</button>
            </div>
        </div>
    `;
    
    // Store enums in global scope for later use
    window.currentEnums = enums;
}

function loadColumnData() {
    const selector = document.getElementById('column-selector');
    const selectedOption = selector.options[selector.selectedIndex];
    
    if (!selectedOption || !selectedOption.value) {
        return;
    }
    
    const columnName = selectedOption.value;
    const columnType = selectedOption.dataset.type;
    const nullable = selectedOption.dataset.nullable === 'true';
    const defaultValue = selectedOption.dataset.default;
    const unique = selectedOption.dataset.unique === 'true';
    const autoIncrement = selectedOption.dataset.autoIncrement === 'true';
    
    // Fill in the form fields
    document.getElementById('column-name').value = columnName;
    document.getElementById('column-type').value = columnType;
    
    // Handle ENUM type
    if (columnType && !columnType.includes('(') && columnType.toUpperCase() !== 'INTEGER' && 
        columnType.toUpperCase() !== 'BIGINT' && columnType.toUpperCase() !== 'TEXT' && 
        columnType.toUpperCase() !== 'VARCHAR' && columnType.toUpperCase() !== 'BOOLEAN' &&
        columnType.toUpperCase() !== 'TIMESTAMP' && columnType.toUpperCase() !== 'TIMESTAMPTZ' &&
        columnType.toUpperCase() !== 'DATE' && columnType.toUpperCase() !== 'TIME' &&
        columnType.toUpperCase() !== 'JSON' && columnType.toUpperCase() !== 'JSONB' &&
        columnType.toUpperCase() !== 'UUID' && columnType.toUpperCase() !== 'BYTEA' &&
        columnType.toUpperCase() !== 'SMALLINT' && columnType.toUpperCase() !== 'FLOAT' &&
        columnType.toUpperCase() !== 'DOUBLE PRECISION' && columnType.toUpperCase() !== 'REAL') {
        // Likely an enum type
        document.getElementById('column-type').value = 'ENUM';
        document.getElementById('column-enum').value = columnType;
        document.getElementById('enum-section').style.display = 'block';
    } else {
        document.getElementById('enum-section').style.display = 'none';
    }
    
    if (document.getElementById('column-nullable')) {
        document.getElementById('column-nullable').checked = nullable;
    }
    if (document.getElementById('column-default')) {
        document.getElementById('column-default').value = defaultValue || '';
    }
    if (document.getElementById('column-unique')) {
        document.getElementById('column-unique').checked = unique;
    }
    if (document.getElementById('column-auto-increment')) {
        document.getElementById('column-auto-increment').checked = autoIncrement;
    }
}

async function previewChange() {
    const columnName = document.getElementById('column-name')?.value;
    
    if (!columnName) {
        showAlert('Required Field', 'Please enter column name', 'error');
        return;
    }
    
    let columnType = document.getElementById('column-type')?.value || '';
    let defaultValue = '';
    
    // Handle ENUM type
    if (columnType === 'ENUM') {
        const enumType = document.getElementById('column-enum')?.value;
        if (!enumType || enumType === '__CREATE_NEW__') {
            showAlert('Required Field', 'Please select an enum type or create a new one first', 'error');
            return;
        }
        columnType = enumType;
        
        // Get enum default value if available
        const enumDefaultSelect = document.getElementById('column-enum-default');
        if (enumDefaultSelect && enumDefaultSelect.style.display !== 'none') {
            defaultValue = enumDefaultSelect.value || '';
        }
    } else {
        // Get regular default value
        defaultValue = document.getElementById('column-default')?.value || '';
    }
    
    const change = {
        type: currentAction,
        table: currentEditTable,
        column: {
            name: columnName,
            type: columnType,
            nullable: document.getElementById('column-nullable')?.checked || false,
            default: defaultValue,
            unique: document.getElementById('column-unique')?.checked || false,
            auto_increment: document.getElementById('column-auto-increment')?.checked || false
        }
    };
    
    // Add foreign key if specified
    const isForeign = document.getElementById('column-is-foreign')?.checked;
    if (isForeign) {
        const foreignTable = document.getElementById('foreign-table')?.value;
        const foreignColumn = document.getElementById('foreign-column')?.value || 'id';
        
        if (!foreignTable) {
            showAlert('Required Field', 'Please select a reference table', 'error');
            return;
        }
        
        change.column.foreign_key = {
            table: foreignTable,
            column: foreignColumn
        };
    }
    
    try {
        const response = await fetch('/api/schema/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        
        const preview = await response.json();
        showPreview(preview, change);
    } catch (error) {
        showAlert('Error', 'Failed to preview change: ' + error.message, 'error');
    }
}

async function showPreview(preview, change) {
    // Check if config file exists
    let hasConfig = false;
    try {
        const response = await fetch('/api/config/check');
        const data = await response.json();
        hasConfig = data.exists || false;
    } catch (error) {
        console.error('Failed to check config:', error);
    }
    
    const content = document.getElementById('panel-content');
    
    const changeList = hasConfig ? `
        <li>Apply to database immediately</li>
        <li>Create migration file in db/migrations/</li>
        <li>Update db/schema/schema.sql</li>
    ` : `
        <li>Apply to database immediately</li>
        <li style="color: #f59e0b;">⚠️ Migration files will NOT be created (no config found)</li>
        <li style="color: #888;">Config files: flash.config.json or graft.config.json not found</li>
    `;
    
    content.innerHTML = `
        <button class="back-btn" onclick="showTableDetails()">
            <span class="iconify" data-icon="mdi:arrow-left"></span> Back to Table
        </button>
        
        <div class="panel-section">
            <div class="section-title">Preview Changes</div>
            
            <div class="preview-box">${preview.sql}</div>
            
            <div class="section-title" style="margin-top: 20px;">What will happen:</div>
            <ul class="change-list">
                ${changeList}
            </ul>
            
            ${!hasConfig ? `
            <div style="background: #f59e0b20; border: 1px solid #f59e0b; border-radius: 6px; padding: 12px; margin: 15px 0; color: #fbbf24; font-size: 13px;">
                <strong>ℹ️ Note:</strong> No flash.config.json or graft.config.json found. 
                Migration files will not be generated. Only the database will be updated.
            </div>
            ` : ''}
            
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
    showModal(
        'Apply Changes',
        'Apply these changes to the database?<br><br>The changes will be executed immediately.',
        async () => {
            await executeApplyChange(change);
        }
    );
}

async function executeApplyChange(change) {
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
            showAlert('Error', result.error || 'Failed to apply changes', 'error');
        }
    } catch (error) {
        showAlert('Error', 'Failed to apply changes: ' + error.message, 'error');
    }
}

// Sidebar Management
let currentTab = 'tables';
let schemaData = null;

async function loadSidebarData() {
    try {
        const response = await fetch('/api/schema');
        const data = await response.json();
        if (data.success && data.data) {
            schemaData = data.data;
            updateSidebar();
        }
    } catch (error) {
        console.error('Failed to load schema data:', error);
    }
}

function switchTab(tab) {
    currentTab = tab;
    document.querySelectorAll('.sidebar-tab').forEach(t => t.classList.remove('active'));
    event.target.closest('.sidebar-tab').classList.add('active');
    document.getElementById('sidebar-search').value = '';
    updateSidebar();
}

function updateSidebar() {
    const content = document.getElementById('sidebar-content');
    if (!schemaData) {
        content.innerHTML = '<div style="padding: 20px; text-align: center; color: #888;">Loading...</div>';
        return;
    }
    
    if (currentTab === 'tables') {
        const tables = schemaData.nodes || [];
        content.innerHTML = `
            <button class="sidebar-add-btn" onclick="showCreateTable()">
                <span class="iconify" data-icon="mdi:plus"></span>
                Add New Table
            </button>
            ${tables.map(table => `
                <div class="sidebar-item" onclick="window.openTableEdit('${table.data.label}')">
                    <span class="sidebar-item-icon">📋</span>
                    <span class="sidebar-item-name">${table.data.label}</span>
                    <span class="sidebar-item-count">${table.data.columns.length}</span>
                </div>
            `).join('')}
        `;
    } else {
        const enums = schemaData.enums || [];
        content.innerHTML = `
            <button class="sidebar-add-btn" onclick="showCreateEnum()">
                <span class="iconify" data-icon="mdi:plus"></span>
                Add New Enum
            </button>
            ${enums.map(enumType => `
                <div class="sidebar-item" onclick="showEnumDetails('${enumType.name}')">
                    <span class="sidebar-item-icon">🏷️</span>
                    <span class="sidebar-item-name">${enumType.name}</span>
                    <span class="sidebar-item-count">${enumType.values ? enumType.values.length : 0}</span>
                </div>
            `).join('')}
        `;
    }
}

function filterSidebarItems() {
    const search = document.getElementById('sidebar-search').value.toLowerCase();
    const items = document.querySelectorAll('.sidebar-item');
    
    items.forEach(item => {
        const name = item.querySelector('.sidebar-item-name').textContent.toLowerCase();
        if (name.includes(search)) {
            item.style.display = 'flex';
        } else {
            item.style.display = 'none';
        }
    });
}

// Load sidebar on page load
window.addEventListener('DOMContentLoaded', () => {
    loadSidebarData();
});

// Reload sidebar after schema changes
window.reloadSidebar = loadSidebarData;

// Create New Table
function showCreateTable() {
    currentEditTable = '';
    currentAction = 'create_table';
    document.getElementById('panel-table-name').textContent = 'New Table';
    document.getElementById('edit-panel').classList.add('open');
    
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <div class="panel-section">
            <div class="section-title">Create New Table</div>
            
            <div class="form-group">
                <label class="form-label">Table Name</label>
                <input type="text" id="new-table-name" class="form-input" placeholder="users, products, orders...">
            </div>
            
            <div id="columns-builder">
                <div class="section-title" style="margin-top: 20px;">Columns</div>
                <div id="column-list"></div>
                <button class="btn btn-secondary" onclick="addColumnToBuilder()" style="width: 100%; margin-top: 10px;">
                    <span class="iconify" data-icon="mdi:plus"></span> Add Column
                </button>
            </div>
            
            <div class="btn-group" style="margin-top: 20px;">
                <button class="btn btn-primary" onclick="createTablePreview()">Create Table</button>
                <button class="btn btn-secondary" onclick="closeEditPanel()">Cancel</button>
            </div>
        </div>
    `;
    
    // Add initial ID column
    addColumnToBuilder('id', 'INTEGER', false, '', true, true, false);
}

let columnBuilderIndex = 0;

async function addColumnToBuilder(name = '', type = 'VARCHAR(255)', nullable = true, defaultVal = '', isPrimary = false, autoIncrement = false, unique = false, foreignTable = '', foreignColumn = '') {
    const index = columnBuilderIndex++;
    const columnList = document.getElementById('column-list');
    
    // Fetch enums and tables
    let enums = [];
    let tables = [];
    try {
        const response = await fetch('/api/schema');
        const data = await response.json();
        if (data.success && data.data) {
            enums = data.data.enums || [];
            tables = data.data.nodes || [];
        }
    } catch (error) {
        console.error('Failed to fetch schema:', error);
    }
    
    const enumOptions = enums.map(e => `<option value="${e.name}" ${type === e.name ? 'selected' : ''}>${e.name}</option>`).join('');
    const tableOptions = tables.map(t => `<option value="${t.data.label}" ${foreignTable === t.data.label ? 'selected' : ''}>${t.data.label}</option>`).join('');
    
    const columnDiv = document.createElement('div');
    columnDiv.className = 'column-builder-item';
    columnDiv.id = `column-${index}`;
    columnDiv.innerHTML = `
        <div class="column-builder-header">
            <button class="icon-btn collapse-btn" onclick="toggleColumnFields(${index})" style="margin-right: 8px;">
                <span class="iconify" data-icon="mdi:minus" id="collapse-icon-${index}"></span>
            </button>
            <input type="text" class="form-input" value="${name}" placeholder="column_name" style="flex: 1; margin-right: 8px;" id="col-name-${index}">
            <button class="icon-btn delete-btn" onclick="removeColumnFromBuilder(${index})" ${isPrimary ? 'disabled style="opacity: 0.3;"' : ''}>
                <span class="iconify" data-icon="mdi:delete"></span>
            </button>
        </div>
        <div class="column-builder-fields" id="fields-${index}">
            <select class="form-select" id="col-type-${index}" onchange="handleBuilderTypeChange(${index})" style="margin-bottom: 8px;">
                <optgroup label="String Types">
                    <option value="VARCHAR(255)" ${type === 'VARCHAR(255)' ? 'selected' : ''}>VARCHAR(255)</option>
                    <option value="TEXT" ${type === 'TEXT' ? 'selected' : ''}>TEXT</option>
                </optgroup>
                <optgroup label="Numeric Types">
                    <option value="INTEGER" ${type === 'INTEGER' ? 'selected' : ''}>INTEGER</option>
                    <option value="BIGINT" ${type === 'BIGINT' ? 'selected' : ''}>BIGINT</option>
                    <option value="DECIMAL(10,2)" ${type === 'DECIMAL(10,2)' ? 'selected' : ''}>DECIMAL</option>
                </optgroup>
                <optgroup label="Date/Time">
                    <option value="TIMESTAMP" ${type === 'TIMESTAMP' ? 'selected' : ''}>TIMESTAMP</option>
                    <option value="DATE" ${type === 'DATE' ? 'selected' : ''}>DATE</option>
                </optgroup>
                <optgroup label="Other">
                    <option value="BOOLEAN" ${type === 'BOOLEAN' ? 'selected' : ''}>BOOLEAN</option>
                    <option value="JSON" ${type === 'JSON' ? 'selected' : ''}>JSON</option>
                    <option value="UUID" ${type === 'UUID' ? 'selected' : ''}>UUID</option>
                </optgroup>
                ${enumOptions ? `<optgroup label="Enums">${enumOptions}<option value="__CREATE_ENUM__">+ Create New Enum</option></optgroup>` : '<optgroup label="Enums"><option value="__CREATE_ENUM__">+ Create New Enum</option></optgroup>'}
            </select>
            <div style="display: flex; gap: 8px; flex-wrap: wrap;">
                <label class="form-checkbox" style="flex: 1;">
                    <input type="checkbox" id="col-nullable-${index}" ${nullable ? 'checked' : ''}>
                    <span>NULL</span>
                </label>
                <label class="form-checkbox" style="flex: 1;">
                    <input type="checkbox" id="col-unique-${index}" ${unique ? 'checked' : ''}>
                    <span>UNIQUE</span>
                </label>
                <label class="form-checkbox" style="flex: 1;">
                    <input type="checkbox" id="col-auto-${index}" ${autoIncrement ? 'checked' : ''}>
                    <span>AUTO</span>
                </label>
                <label class="form-checkbox" style="flex: 1;">
                    <input type="checkbox" id="col-primary-${index}" ${isPrimary ? 'checked' : ''}>
                    <span>PK</span>
                </label>
            </div>
            <input type="text" class="form-input" value="${defaultVal}" placeholder="Default value..." id="col-default-${index}" style="margin-top: 8px; font-size: 12px;">
            
            <div style="margin-top: 8px; padding-top: 8px; border-top: 1px solid #333;">
                <label class="form-checkbox">
                    <input type="checkbox" id="col-is-foreign-${index}" onchange="toggleBuilderForeignKey(${index})" ${foreignTable ? 'checked' : ''}>
                    <span style="font-size: 11px;">🔗 Foreign Key</span>
                </label>
                <div id="col-fk-section-${index}" style="display: ${foreignTable ? 'block' : 'none'}; margin-top: 8px;">
                    <select class="form-select" id="col-fk-table-${index}" style="margin-bottom: 4px; font-size: 11px;">
                        <option value="">-- Select Table --</option>
                        ${tableOptions}
                    </select>
                    <input type="text" class="form-input" value="${foreignColumn}" placeholder="Referenced column (e.g., id)" id="col-fk-column-${index}" style="font-size: 11px;">
                </div>
            </div>
        </div>
    `;
    
    columnList.appendChild(columnDiv);
}

function handleBuilderTypeChange(index) {
    const typeSelect = document.getElementById(`col-type-${index}`);
    
    // Always remove any existing enum form first
    const columnDiv = document.getElementById(`column-${index}`);
    if (columnDiv) {
        const existingForm = columnDiv.querySelector('.enum-create-form');
        if (existingForm) {
            existingForm.remove();
        }
    }
    
    // Show enum creation form if selected
    if (typeSelect.value === '__CREATE_ENUM__') {
        showQuickCreateEnum(index);
    }
}

function toggleBuilderForeignKey(index) {
    const checkbox = document.getElementById(`col-is-foreign-${index}`);
    const section = document.getElementById(`col-fk-section-${index}`);
    if (checkbox && section) {
        section.style.display = checkbox.checked ? 'block' : 'none';
    }
}

async function showQuickCreateEnum(columnIndex) {
    const columnDiv = document.getElementById(`column-${columnIndex}`);
    if (!columnDiv) return;
    
    // Store original type select value
    const typeSelect = document.getElementById(`col-type-${columnIndex}`);
    const originalValue = typeSelect.value;
    
    // Find the column-builder-fields div to insert enum form after it
    const fieldsDiv = columnDiv.querySelector('.column-builder-fields');
    
    // Check if enum form already exists
    let enumFormDiv = columnDiv.querySelector('.enum-create-form');
    if (enumFormDiv) {
        enumFormDiv.remove();
    }
    
    // Create enum form
    enumFormDiv = document.createElement('div');
    enumFormDiv.className = 'enum-create-form';
    enumFormDiv.style.cssText = 'background: #2a2a2a; padding: 12px; border-radius: 6px; margin-top: 10px; border: 1px solid #10b981;';
    enumFormDiv.innerHTML = `
        <div style="font-weight: 600; margin-bottom: 10px; color: #10b981; font-size: 12px;">
            <span class="iconify" data-icon="mdi:format-list-bulleted-square"></span> Create New Enum
        </div>
        <div class="form-group" style="margin-bottom: 8px;">
            <label class="form-label" style="font-size: 11px;">Enum Name</label>
            <input type="text" id="builder-enum-name-${columnIndex}" class="form-input" placeholder="status, role, priority..." style="font-size: 11px;">
        </div>
        <div class="form-group" style="margin-bottom: 10px;">
            <label class="form-label" style="font-size: 11px;">Values (one per line)</label>
            <textarea id="builder-enum-values-${columnIndex}" class="form-input" rows="4" placeholder="active&#10;inactive&#10;pending" style="font-size: 11px;"></textarea>
        </div>
        <div style="display: flex; gap: 6px;">
            <button class="btn btn-primary" onclick="createBuilderEnum(${columnIndex})" style="flex: 1; font-size: 11px; padding: 6px;">Create</button>
            <button class="btn btn-secondary" onclick="cancelBuilderEnum(${columnIndex})" style="font-size: 11px; padding: 6px;">Cancel</button>
        </div>
    `;
    
    fieldsDiv.appendChild(enumFormDiv);
}

async function createBuilderEnum(columnIndex) {
    const enumNameInput = document.getElementById(`builder-enum-name-${columnIndex}`);
    const enumValuesInput = document.getElementById(`builder-enum-values-${columnIndex}`);
    
    const enumName = enumNameInput?.value.trim();
    const valuesText = enumValuesInput?.value.trim();
    
    if (!enumName) {
        showModal('Validation', 'Please enter enum name', 'warning');
        return;
    }
    
    if (!valuesText) {
        showModal('Validation', 'Please enter enum values', 'warning');
        return;
    }
    
    const values = valuesText.split('\n').map(v => v.trim()).filter(v => v);
    if (values.length === 0) {
        showModal('Validation', 'Please enter at least one enum value', 'warning');
        return;
    }
    
    const change = {
        type: 'create_enum',
        enum_name: enumName,
        enum_values: values
    };
    
    try {
        const previewRes = await fetch('/api/schema/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        const preview = await previewRes.json();
        
        // Show confirmation in preview style
        const message = `<div style="text-align:left;"><pre style="white-space:pre-wrap; background:#0f0f0f; padding:10px; border-radius:4px; font-size: 11px;">${preview.sql}</pre></div>`;
        
        showConfirm('Create Enum', message, async () => {
            const applyRes = await fetch('/api/schema/apply', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(change)
            });
            const result = await applyRes.json();
            
            if (result.success) {
                showModal('Success', `Enum "${enumName}" created successfully!`, 'success');
                
                // Remove the enum form first
                const columnDiv = document.getElementById(`column-${columnIndex}`);
                const enumForm = columnDiv?.querySelector('.enum-create-form');
                if (enumForm) {
                    enumForm.remove();
                }
                
                // Update the select to use the new enum
                const typeSelect = document.getElementById(`col-type-${columnIndex}`);
                if (typeSelect) {
                    // Fetch updated enum list
                    const response = await fetch('/api/schema');
                    const data = await response.json();
                    if (data.success && data.data) {
                        const enums = data.data.enums || [];
                        const enumOptions = enums.map(e => `<option value="${e.name}" ${e.name === enumName ? 'selected' : ''}>${e.name}</option>`).join('');
                        
                        // Find and update the Enums optgroup
                        const enumGroup = typeSelect.querySelector('optgroup[label="Enums"]');
                        if (enumGroup) {
                            enumGroup.innerHTML = `${enumOptions}<option value="__CREATE_ENUM__">+ Create New Enum</option>`;
                        }
                        
                        // Set the newly created enum as selected
                        typeSelect.value = enumName;
                    }
                }
            } else {
                showModal('Error', 'Failed to create enum: ' + (result.error || 'Unknown error'), 'error');
            }
        });
    } catch (err) {
        console.error(err);
        showModal('Error', 'Failed to create enum: ' + err.message, 'error');
    }
}

function cancelBuilderEnum(columnIndex) {
    const columnDiv = document.getElementById(`column-${columnIndex}`);
    if (!columnDiv) return;
    
    const enumFormDiv = columnDiv.querySelector('.enum-create-form');
    if (enumFormDiv) {
        enumFormDiv.remove();
    }
    
    // Reset type select to VARCHAR
    const typeSelect = document.getElementById(`col-type-${columnIndex}`);
    if (typeSelect) {
        typeSelect.value = 'VARCHAR(255)';
    }
}

function removeColumnFromBuilder(index) {
    const column = document.getElementById(`column-${index}`);
    if (column) column.remove();
}

function toggleColumnFields(index) {
    const fieldsDiv = document.getElementById(`fields-${index}`);
    const icon = document.getElementById(`collapse-icon-${index}`);
    
    if (fieldsDiv && icon) {
        if (fieldsDiv.style.display === 'none') {
            fieldsDiv.style.display = 'block';
            icon.setAttribute('data-icon', 'mdi:minus');
        } else {
            fieldsDiv.style.display = 'none';
            icon.setAttribute('data-icon', 'mdi:plus');
        }
    }
}

async function createTablePreview() {
    const tableName = document.getElementById('new-table-name').value.trim();
    if (!tableName) {
        showAlert('Required Field', 'Please enter table name', 'error');
        return;
    }
    
    // Collect columns
    const columns = [];
    document.querySelectorAll('.column-builder-item').forEach(item => {
        const index = item.id.split('-')[1];
        const name = document.getElementById(`col-name-${index}`).value.trim();
        if (name) {
            const column = {
                name,
                type: document.getElementById(`col-type-${index}`).value,
                nullable: document.getElementById(`col-nullable-${index}`).checked,
                default: document.getElementById(`col-default-${index}`).value.trim(),
                unique: document.getElementById(`col-unique-${index}`).checked,
                auto_increment: document.getElementById(`col-auto-${index}`).checked,
                is_primary: document.getElementById(`col-primary-${index}`).checked
            };
            
            // Check for foreign key
            const isForeign = document.getElementById(`col-is-foreign-${index}`);
            if (isForeign && isForeign.checked) {
                const fkTable = document.getElementById(`col-fk-table-${index}`).value;
                const fkColumn = document.getElementById(`col-fk-column-${index}`).value.trim() || 'id';
                
                if (fkTable) {
                    column.foreign_key = {
                        table: fkTable,
                        column: fkColumn
                    };
                }
            }
            
            columns.push(column);
        }
    });
    
    if (columns.length === 0) {
        showAlert('Required Field', 'Please add at least one column', 'error');
        return;
    }
    
    const change = {
        type: 'create_table',
        table: tableName,
        columns: columns
    };
    
    try {
        const response = await fetch('/api/schema/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        
        const preview = await response.json();
        currentEditTable = tableName;
        await showPreview(preview, change);
    } catch (error) {
        showAlert('Error', 'Failed to create table: ' + error.message, 'error');
    }
}

// Create New Enum
function showCreateEnum() {
    currentAction = 'create_enum';
    document.getElementById('panel-table-name').textContent = 'New Enum';
    document.getElementById('edit-panel').classList.add('open');
    
    const content = document.getElementById('panel-content');
    content.innerHTML = `
        <div class="panel-section">
            <div class="section-title">Create New Enum Type</div>
            
            <div class="form-group">
                <label class="form-label">Enum Name</label>
                <input type="text" id="enum-name" class="form-input" placeholder="status, role, priority...">
            </div>
            
            <div class="form-group">
                <label class="form-label">Values (one per line)</label>
                <textarea id="enum-values" class="form-input" rows="8" placeholder="active&#10;inactive&#10;pending&#10;completed"></textarea>
            </div>
            
            <div class="btn-group">
                <button class="btn btn-primary" onclick="createEnumPreview()">Create Enum</button>
                <button class="btn btn-secondary" onclick="closeEditPanel()">Cancel</button>
            </div>
        </div>
    `;
}

async function createEnumPreview() {
    const enumName = document.getElementById('enum-name').value.trim();
    const enumValuesText = document.getElementById('enum-values').value.trim();
    
    if (!enumName) {
        showAlert('Required Field', 'Please enter enum name', 'error');
        return;
    }
    
    if (!enumValuesText) {
        showAlert('Required Field', 'Please enter enum values', 'error');
        return;
    }
    
    const values = enumValuesText.split('\n').map(v => v.trim()).filter(v => v);
    
    if (values.length === 0) {
        showAlert('Required Field', 'Please enter at least one enum value', 'error');
        return;
    }
    
    const change = {
        type: 'create_enum',
        enum_name: enumName,
        enum_values: values
    };
    
    try {
        const response = await fetch('/api/schema/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        
        const preview = await response.json();
        await showPreview(preview, change);
    } catch (error) {
        showAlert('Error', 'Failed to create enum: ' + error.message, 'error');
    }
}

// Manage All Enums
async function showManageEnums() {
    currentAction = 'manage_enums';
    document.getElementById('panel-table-name').textContent = 'Manage Enums';
    document.getElementById('edit-panel').classList.add('open');
    
    // Fetch enums
    let enums = [];
    try {
        const response = await fetch('/api/schema');
        const data = await response.json();
        if (data.success && data.data) {
            enums = data.data.enums || [];
        }
    } catch (error) {
        console.error('Failed to fetch enums:', error);
    }
    
    const content = document.getElementById('panel-content');
    
    if (enums.length === 0) {
        content.innerHTML = `
            <div class="panel-section">
                <div class="section-title">All Enums</div>
                <div style="text-align: center; padding: 40px 20px; color: #888;">
                    <span class="iconify" data-icon="mdi:format-list-bulleted" style="font-size: 48px; opacity: 0.3;"></span>
                    <p style="margin-top: 15px;">No enums found</p>
                    <button class="btn btn-primary" onclick="showCreateEnum()" style="margin-top: 15px;">
                        <span class="iconify" data-icon="mdi:plus"></span> Create First Enum
                    </button>
                </div>
            </div>
        `;
        return;
    }
    
    const enumsList = enums.map(e => {
        const name = e.name || 'Unknown';
        const values = e.values || [];
        const valuesBadges = values.slice(0, 3).map(v => `<span class="badge badge-info">${v}</span>`).join(' ');
        const moreCount = values.length > 3 ? `<span style="color: #888; font-size: 11px;">+${values.length - 3} more</span>` : '';
        
        return `
            <div class="sidebar-item" onclick="showEnumDetails('${name}')">
                <span class="iconify sidebar-item-icon" data-icon="mdi:format-list-bulleted" style="color: #a855f7;"></span>
                <div style="flex: 1;">
                    <div class="sidebar-item-name">${name}</div>
                    <div style="margin-top: 4px; display: flex; gap: 4px; align-items: center; flex-wrap: wrap;">
                        ${valuesBadges}
                        ${moreCount}
                    </div>
                </div>
                <span class="sidebar-item-count">${values.length} values</span>
            </div>
        `;
    }).join('');
    
    content.innerHTML = `
        <div class="panel-section">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                <div class="section-title" style="margin: 0;">All Enums (${enums.length})</div>
                <button class="btn btn-primary" onclick="showCreateEnum()" style="padding: 6px 12px; font-size: 12px;">
                    <span class="iconify" data-icon="mdi:plus"></span> New
                </button>
            </div>
            ${enumsList}
        </div>
    `;
}

// Show Enum Details
function showEnumDetails(enumName) {
    // Need to fetch current enums since we don't have schemaData here
    fetch('/api/schema')
        .then(res => res.json())
        .then(data => {
            if (!data.success || !data.data) return;
            const enumData = data.data.enums?.find(e => e.name === enumName);
            if (!enumData) return;
            
            document.getElementById('panel-table-name').textContent = enumName;
            document.getElementById('edit-panel').classList.add('open');
            
            const content = document.getElementById('panel-content');
            const valuesList = enumData.values ? enumData.values.map(v => 
                `<div class="badge badge-info" style="margin: 4px; display: inline-block;">${v}</div>`
            ).join('') : '';
            
            content.innerHTML = `
                <button class="back-btn" onclick="showManageEnums()">
                    <span class="iconify" data-icon="mdi:arrow-left"></span> Back to Enums
                </button>
                
                <div class="panel-section">
                    <div class="section-title">Enum: ${enumName}</div>
                    <div style="margin: 15px 0;">
                        ${valuesList}
                    </div>
                </div>
                
                <div class="panel-section">
                    <div class="section-title">Actions</div>
                    
                    <div class="action-card" onclick="showEditEnum('${enumName}')">
                        <div class="action-card-title">
                            <span class="iconify" data-icon="mdi:pencil" style="color: #3b82f6;"></span>
                            Edit Enum Values
                        </div>
                    </div>
                    
                    <div class="action-card action-card-danger" onclick="deleteEnum('${enumName}')">
                        <div class="action-card-title">
                            <span class="iconify" data-icon="mdi:delete" style="color: #dc2626;"></span>
                            Delete Enum
                        </div>
                    </div>
                </div>
            `;
        })
        .catch(error => {
            console.error('Failed to fetch enum details:', error);
        });
}

function showEditEnum(enumName) {
    // Fetch current enum data
    fetch('/api/schema')
        .then(res => res.json())
        .then(data => {
            if (!data.success || !data.data) return;
            const enumData = data.data.enums?.find(e => e.name === enumName);
            if (!enumData) return;
            
            const valuesText = enumData.values ? enumData.values.join('\n') : '';
            
            const content = document.getElementById('panel-content');
            content.innerHTML = `
                <button class="back-btn" onclick="showEnumDetails('${enumName}')">
                    <span class="iconify" data-icon="mdi:arrow-left"></span> Back
                </button>
                
                <div class="panel-section">
                    <div class="section-title">Edit Enum: ${enumName}</div>
                    
                    <div class="form-group">
                        <label class="form-label">Values (one per line)</label>
                        <textarea id="enum-values" class="form-input" rows="10">${valuesText}</textarea>
                    </div>
                    
                    <div class="btn-group">
                        <button class="btn btn-primary" onclick="updateEnumPreview('${enumName}')">Update Enum</button>
                        <button class="btn btn-secondary" onclick="showEnumDetails('${enumName}')">Cancel</button>
                    </div>
                </div>
            `;
        })
        .catch(error => {
            console.error('Failed to fetch enum for editing:', error);
        });
}

async function updateEnumPreview(enumName) {
    const enumValuesText = document.getElementById('enum-values').value.trim();
    
    if (!enumValuesText) {
        showAlert('Required Field', 'Please enter enum values', 'error');
        return;
    }
    
    const values = enumValuesText.split('\n').map(v => v.trim()).filter(v => v);
    
    const change = {
        type: 'alter_enum',
        enum_name: enumName,
        enum_values: values
    };
    
    try {
        const response = await fetch('/api/schema/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(change)
        });
        
        const preview = await response.json();
        await showPreview(preview, change);
    } catch (error) {
        showAlert('Error', 'Failed to update enum: ' + error.message, 'error');
    }
}

async function deleteEnum(enumName) {
    showModal(
        'Delete Enum',
        `Are you sure you want to delete enum type <strong>"${enumName}"</strong>?<br><br>⚠️ This may break columns that use this enum type!`,
        async () => {
            const change = {
                type: 'drop_enum',
                enum_name: enumName
            };
            
            try {
                const response = await fetch('/api/schema/preview', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(change)
                });
                
                const preview = await response.json();
                await showPreview(preview, change);
            } catch (error) {
                showAlert('Error', 'Failed to delete enum: ' + error.message, 'error');
            }
        },
        true
    );
}

// Handle hash navigation from index page
window.addEventListener('DOMContentLoaded', () => {
    const hash = window.location.hash;
    if (hash === '#create-table') {
        setTimeout(() => {
            showCreateTable();
            window.location.hash = '';
        }, 1000);
    } else if (hash === '#create-enum') {
        setTimeout(() => {
            showCreateEnum();
            window.location.hash = '';
        }, 1000);
    }
});
