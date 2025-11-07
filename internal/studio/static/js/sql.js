let editor;
let currentResults = null;

document.addEventListener('DOMContentLoaded', () => {
    editor = CodeMirror.fromTextArea(document.getElementById('sql-editor'), {
        mode: 'text/x-sql',
        theme: 'material-darker',
        lineNumbers: true,
        lineWrapping: true,
        autofocus: true,
        extraKeys: {
            'Ctrl-Enter': runQuery,
            'Cmd-Enter': runQuery
        }
    });
    
    editor.setValue('SELECT * FROM users LIMIT 10;');
    setupResize();
});

async function runQuery() {
    const query = editor.getValue().trim();
    if (!query) return;
    
    document.getElementById('results-info').textContent = 'Executing query...';
    document.getElementById('results-body').innerHTML = '<div class="empty-state"><div>Loading...</div></div>';
    
    try {
        const res = await fetch('/api/sql', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query })
        });
        
        const data = await res.json();
        
        if (data.success) {
            currentResults = data.data;
            displayResults(data.data);
        } else {
            displayError(data.message);
        }
    } catch (err) {
        displayError(err.message);
    }
}

function displayResults(data) {
    const resultsBody = document.getElementById('results-body');
    
    if (!data || !data.rows || data.rows.length === 0) {
        document.getElementById('results-info').textContent = 'Query executed successfully (0 rows)';
        resultsBody.innerHTML = '<div class="success-message">✓ Query executed successfully. No rows returned.</div>';
        document.getElementById('export-btn').style.display = 'none';
        return;
    }
    
    const rowCount = data.rows.length;
    document.getElementById('results-info').textContent = `${rowCount} row${rowCount !== 1 ? 's' : ''} returned`;
    document.getElementById('export-btn').style.display = 'block';
    
    const columns = data.columns && data.columns.length > 0 
        ? data.columns.map(col => col.name || col) 
        : Object.keys(data.rows[0]);
    
    let html = '<table class="results-table"><thead><tr>';
    columns.forEach(col => {
        html += `<th>${col}</th>`;
    });
    html += '</tr></thead><tbody>';
    
    data.rows.forEach(row => {
        html += '<tr>';
        columns.forEach(col => {
            const value = row[col];
            const displayValue = value === null ? '<span style="color: #666; font-style: italic;">NULL</span>' : 
                               typeof value === 'object' ? JSON.stringify(value) : value;
            html += `<td>${displayValue}</td>`;
        });
        html += '</tr>';
    });
    
    html += '</tbody></table>';
    resultsBody.innerHTML = html;
}

function displayError(message) {
    document.getElementById('results-info').textContent = 'Query failed';
    document.getElementById('results-body').innerHTML = 
        `<div class="error-message">❌ Error: ${message}</div>`;
    document.getElementById('export-btn').style.display = 'none';
}

function clearEditor() {
    editor.setValue('');
    editor.focus();
}

function exportResults() {
    if (!currentResults || !currentResults.rows) return;
    
    const rows = currentResults.rows;
    const columns = currentResults.columns && currentResults.columns.length > 0 
        ? currentResults.columns.map(col => col.name || col) 
        : Object.keys(rows[0]);
    
    let csv = columns.join(',') + '\n';
    rows.forEach(row => {
        const values = columns.map(col => {
            const val = row[col];
            return val === null ? '' : `"${String(val).replace(/"/g, '""')}"`;
        });
        csv += values.join(',') + '\n';
    });
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `query_results_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
}

function setupResize() {
    const handle = document.getElementById('resize-handle');
    const editorSection = document.querySelector('.editor-section');
    let isResizing = false;
    
    handle.addEventListener('mousedown', (e) => {
        isResizing = true;
        document.body.style.cursor = 'ns-resize';
    });
    
    document.addEventListener('mousemove', (e) => {
        if (!isResizing) return;
        
        const containerHeight = document.querySelector('.container').offsetHeight;
        const newHeight = (e.clientY - 44) / containerHeight * 100;
        
        if (newHeight > 20 && newHeight < 80) {
            editorSection.style.flex = `0 0 ${newHeight}%`;
        }
    });
    
    document.addEventListener('mouseup', () => {
        isResizing = false;
        document.body.style.cursor = 'default';
    });
}
