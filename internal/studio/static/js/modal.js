// Custom Modal System - Replace browser alerts with beautiful modals

function showModal(title, message, type = 'info') {
    const modal = document.createElement('div');
    modal.className = 'custom-modal show';
    
    const icons = {
        success: '✓',
        error: '✕',
        warning: '⚠',
        info: 'ℹ'
    };
    
    const colors = {
        success: '#10b981',
        error: '#dc2626',
        warning: '#f59e0b',
        info: '#4a9eff'
    };
    
    modal.innerHTML = `
        <div class="custom-modal-content">
            <div class="custom-modal-header">
                <div class="custom-modal-title" style="color: ${colors[type]}">
                    <span style="font-size: 24px; margin-right: 8px;">${icons[type]}</span>
                    ${title}
                </div>
                <button class="custom-modal-close" onclick="this.closest('.custom-modal').remove()">×</button>
            </div>
            <div class="custom-modal-body">
                ${message}
            </div>
            <div class="custom-modal-footer">
                <button class="btn btn-primary" onclick="this.closest('.custom-modal').remove()">OK</button>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    if (type === 'success') {
        setTimeout(() => modal.remove(), 5000);
    }
}

function showConfirm(title, message, onConfirm) {
    const modal = document.createElement('div');
    modal.className = 'custom-modal show';
    
    modal.innerHTML = `
        <div class="custom-modal-content">
            <div class="custom-modal-header">
                <div class="custom-modal-title" style="color: #f59e0b">
                    <span style="font-size: 24px; margin-right: 8px;">⚠</span>
                    ${title}
                </div>
                <button class="custom-modal-close" onclick="this.closest('.custom-modal').remove()">×</button>
            </div>
            <div class="custom-modal-body">
                ${message}
            </div>
            <div class="custom-modal-footer">
                <button class="btn btn-secondary" onclick="this.closest('.custom-modal').remove()">Cancel</button>
                <button class="btn btn-primary" style="background: #dc2626;" id="confirm-btn">Confirm</button>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    document.getElementById('confirm-btn').onclick = () => {
        modal.remove();
        onConfirm();
    };
}

function showAddRowModal(columns, onSave) {
    const modal = document.createElement('div');
    modal.className = 'custom-modal show';
    
    const seen = new Set();
    const uniqueColumns = columns.filter(col => {
        if (seen.has(col.name)) return false;
        seen.add(col.name);
        return true;
    });
    
    const fields = uniqueColumns.map(col => {
        const isAutoIncrement = col.auto_increment === true;
        const hasDefault = col.default && col.default.trim() !== '';
        const isRequired = !col.nullable && !isAutoIncrement && !hasDefault;
        const readonlyAttr = isAutoIncrement ? 'readonly style="background: #1a1a1a; cursor: not-allowed;"' : '';
        const placeholderText = isAutoIncrement ? 'Auto-generated' : hasDefault ? `Default: ${col.default}` : `Enter ${col.name}`;
        
        return `
            <div class="form-group">
                <label class="form-label">
                    ${col.name}
                    <span class="type-badge">${col.type}</span>
                    ${col.primary_key ? '<span class="schema-column-badge">PK</span>' : ''}
                    ${isRequired ? '<span style="color: #dc2626;">*</span>' : ''}
                </label>
                <input 
                    type="text" 
                    class="form-input" 
                    id="field-${col.name}" 
                    placeholder="${placeholderText}"
                    ${readonlyAttr}
                    ${isRequired ? 'required' : ''}
                />
                ${isRequired ? '<div class="form-hint">Required field</div>' : ''}
                ${isAutoIncrement ? '<div class="form-hint">This field will be auto-generated</div>' : ''}
                ${hasDefault && !isAutoIncrement ? `<div class="form-hint">Optional - has default value</div>` : ''}
            </div>
        `;
    }).join('');
    
    modal.innerHTML = `
        <div class="custom-modal-content">
            <div class="custom-modal-header">
                <div class="custom-modal-title">
                    ➕ Add New Record
                </div>
                <button class="custom-modal-close" onclick="this.closest('.custom-modal').remove()">×</button>
            </div>
            <div class="custom-modal-body">
                <form id="add-row-form">
                    ${fields}
                </form>
            </div>
            <div class="custom-modal-footer">
                <button class="btn btn-secondary" onclick="this.closest('.custom-modal').remove()">Cancel</button>
                <button class="btn btn-success" id="save-new-row-btn">💾 Save Record</button>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    document.getElementById('save-new-row-btn').onclick = () => {
        const data = {};
        let hasError = false;
        
        uniqueColumns.forEach(col => {
            const input = document.getElementById(`field-${col.name}`);
            if (!input) return;
            
            const value = input.value.trim();
            const isAutoIncrement = col.auto_increment === true;
            const hasDefault = col.default && col.default.trim() !== '';
            const isRequired = !col.nullable && !isAutoIncrement && !hasDefault;
            
            if (isAutoIncrement) return;
            
            if (isRequired && !value) {
                input.style.borderColor = '#dc2626';
                hasError = true;
                return;
            } else {
                input.style.borderColor = '#3a3a3a';
            }
            
            if (value) {
                data[col.name] = value;
            }
        });
        
        if (hasError) {
            showModal('Validation Error', 'Please fill in all required fields', 'error');
            return;
        }
        
        modal.remove();
        onSave(data);
    };
}

window.showModal = showModal;
window.showConfirm = showConfirm;
window.showAddRowModal = showAddRowModal;
