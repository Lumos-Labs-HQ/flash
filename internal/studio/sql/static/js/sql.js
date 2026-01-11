let editor;
let currentResults = null;
let queryHistory = [];
let historyIndex = -1;

// Storage key for SQL editor state
const SQL_STORAGE_KEY = 'flashorm_sql_editor_state';

// Save SQL editor state
function saveSqlState() {
    const state = {
        content: editor ? editor.getValue() : '',
        queryHistory: queryHistory,
        historyIndex: historyIndex
    };
    try {
        sessionStorage.setItem(SQL_STORAGE_KEY, JSON.stringify(state));
    } catch (e) {
        console.warn('Failed to save SQL state:', e);
    }
}

// Restore SQL editor state
function restoreSqlState() {
    try {
        const saved = sessionStorage.getItem(SQL_STORAGE_KEY);
        if (saved) {
            const state = JSON.parse(saved);
            if (state.content && editor) {
                editor.setValue(state.content);
            }
            if (state.queryHistory) {
                queryHistory = state.queryHistory;
            }
            if (typeof state.historyIndex === 'number') {
                historyIndex = state.historyIndex;
            }
            return true;
        }
    } catch (e) {
        console.warn('Failed to restore SQL state:', e);
    }
    return false;
}

document.addEventListener('DOMContentLoaded', () => {
    editor = CodeMirror.fromTextArea(document.getElementById('sql-editor'), {
        mode: 'text/x-sql',
        theme: 'material-darker',
        lineNumbers: true,
        lineWrapping: true,
        autofocus: true,
        matchBrackets: true,
        autoCloseBrackets: true,
        extraKeys: {
            'Ctrl-Enter': runQuery,
            'Cmd-Enter': runQuery,
            'Ctrl-Space': 'autocomplete',
            'Ctrl-/': toggleComment,
            'Cmd-/': toggleComment,
            'Ctrl-Up': () => navigateHistory(-1),
            'Ctrl-Down': () => navigateHistory(1),
            'F5': runQuery
        },
        hintOptions: {
            tables: {}
        }
    });

    // Try to restore previous state, otherwise use default content
    if (!restoreSqlState()) {
        editor.setValue('-- Welcome to SQL Editor\\n-- Press Ctrl+Enter (or Cmd+Enter on Mac) to run query\\n-- Press F5 to run query\\n-- Press Ctrl+/ to toggle comment\\n-- Press Ctrl+Up/Down to navigate history\\n\\nSELECT * FROM users LIMIT 10;');
    }

    // Save state on every change (debounced)
    let saveTimeout;
    editor.on('change', () => {
        clearTimeout(saveTimeout);
        saveTimeout = setTimeout(saveSqlState, 500);
    });

    // Save state before page unload
    window.addEventListener('beforeunload', saveSqlState);

    // Also save state when clicking any navigation link
    document.querySelectorAll('a[href]').forEach(link => {
        link.addEventListener('click', saveSqlState);
    });

    setupResize();
    loadTableHints();
});

// Load table names for autocomplete
async function loadTableHints() {
    try {
        const res = await fetch('/api/tables');
        const json = await res.json();
        if (json.success && json.data) {
            const tables = {};
            json.data.forEach(t => {
                tables[t.name] = [];
            });
            editor.options.hintOptions.tables = tables;
        }
    } catch (e) {
        console.error('Failed to load table hints:', e);
    }
}

// Toggle comment for selected lines
function toggleComment() {
    const from = editor.getCursor('from');
    const to = editor.getCursor('to');

    for (let i = from.line; i <= to.line; i++) {
        const line = editor.getLine(i);
        if (line.trimStart().startsWith('--')) {
            editor.replaceRange(
                line.replace(/^(\s*)--\s?/, '$1'),
                { line: i, ch: 0 },
                { line: i, ch: line.length }
            );
        } else {
            editor.replaceRange('-- ' + line, { line: i, ch: 0 }, { line: i, ch: line.length });
        }
    }
}

// Navigate through query history
function navigateHistory(direction) {
    if (queryHistory.length === 0) return;

    historyIndex += direction;
    if (historyIndex < 0) historyIndex = 0;
    if (historyIndex >= queryHistory.length) historyIndex = queryHistory.length - 1;

    editor.setValue(queryHistory[historyIndex]);
    editor.setCursor(editor.lineCount(), 0);
}

async function runQuery() {
    let query = editor.getSelection() || editor.getValue();
    query = query.trim();
    if (!query) return;

    // Remove comments for execution but keep them in editor
    const cleanQuery = query.split('\n')
        .filter(line => !line.trim().startsWith('--'))
        .join('\n')
        .trim();

    if (!cleanQuery) {
        displayError('No executable SQL found. Remove or bypass comments.');
        return;
    }

    // Add to history
    if (queryHistory[queryHistory.length - 1] !== query) {
        queryHistory.push(query);
        if (queryHistory.length > 50) queryHistory.shift();
    }
    historyIndex = queryHistory.length;

    document.getElementById('results-info').textContent = 'Executing query...';
    document.getElementById('results-body').innerHTML = '<div class="empty-state"><div class="spinner"></div><div>Running query...</div></div>';

    const startTime = Date.now();

    try {
        const res = await fetch('/api/sql', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query: cleanQuery })
        });

        const data = await res.json();
        const elapsed = Date.now() - startTime;

        // Debug logging - check console for data structure
        console.log('[SQL Debug] Response:', {
            success: data.success,
            columns: data.data?.columns,
            rowCount: data.data?.rows?.length,
            firstRow: data.data?.rows?.[0]
        });

        if (data.success) {
            currentResults = data.data;
            displayResults(data.data, cleanQuery, elapsed);
        } else {
            displayError(data.message);
        }
    } catch (err) {
        displayError(err.message);
    }
}

// Detect query type
function getQueryType(query) {
    const upper = query.trim().toUpperCase();
    if (upper.startsWith('SELECT') || upper.startsWith('WITH') || upper.startsWith('SHOW') || upper.startsWith('DESCRIBE') || upper.startsWith('EXPLAIN')) {
        return 'SELECT';
    }
    if (upper.startsWith('INSERT')) return 'INSERT';
    if (upper.startsWith('UPDATE')) return 'UPDATE';
    if (upper.startsWith('DELETE')) return 'DELETE';
    if (upper.startsWith('CREATE')) return 'CREATE';
    if (upper.startsWith('ALTER')) return 'ALTER';
    if (upper.startsWith('DROP')) return 'DROP';
    if (upper.startsWith('TRUNCATE')) return 'TRUNCATE';
    if (upper.startsWith('SET')) return 'SET';
    if (upper.startsWith('BEGIN') || upper.startsWith('START')) return 'TRANSACTION';
    if (upper.startsWith('COMMIT')) return 'COMMIT';
    if (upper.startsWith('ROLLBACK')) return 'ROLLBACK';
    return 'OTHER';
}

// Format value for display with proper type handling
function formatCellValue(value, colName) {
    if (value === null || value === undefined) {
        return '<span class="cell-null">NULL</span>';
    }

    // Handle different types
    if (typeof value === 'boolean') {
        return `<span class="cell-bool">${value ? 'true' : 'false'}</span>`;
    }

    if (typeof value === 'number') {
        return `<span class="cell-number">${value}</span>`;
    }

    if (typeof value === 'object') {
        // Handle Date objects
        if (value instanceof Date) {
            return `<span class="cell-date">${value.toISOString()}</span>`;
        }
        // Handle JSON/Array
        try {
            const jsonStr = JSON.stringify(value, null, 2);
            const escaped = escapeHtml(jsonStr);
            return `<span class="cell-json" title="${escaped}">${escapeHtml(JSON.stringify(value))}</span>`;
        } catch {
            return `<span class="cell-object">[Object]</span>`;
        }
    }

    const strValue = String(value);

    // UUID detection (8-4-4-4-12 format)
    if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(strValue)) {
        return `<span class="cell-uuid" title="${strValue}">${strValue}</span>`;
    }

    // Date/Time detection
    if (/^\d{4}-\d{2}-\d{2}(T|\s)\d{2}:\d{2}:\d{2}/.test(strValue)) {
        return `<span class="cell-date">${escapeHtml(strValue)}</span>`;
    }

    // Email detection
    if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(strValue)) {
        return `<span class="cell-email">${escapeHtml(strValue)}</span>`;
    }

    // URL detection
    if (/^https?:\/\//.test(strValue)) {
        return `<a href="${escapeHtml(strValue)}" target="_blank" class="cell-url">${escapeHtml(strValue)}</a>`;
    }

    // Long text (truncate for display)
    if (strValue.length > 100) {
        const truncated = strValue.substring(0, 100) + '...';
        return `<span class="cell-text cell-truncated" title="${escapeHtml(strValue)}">${escapeHtml(truncated)}</span>`;
    }

    return `<span class="cell-text">${escapeHtml(strValue)}</span>`;
}

// HTML escape utility
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function displayResults(data, query, elapsed) {
    const resultsBody = document.getElementById('results-body');
    const queryType = getQueryType(query);

    // Handle non-SELECT queries (INSERT, UPDATE, DELETE, SET, etc.)
    if (!data || !data.rows || data.rows.length === 0) {
        let message = '';
        let icon = '✓';

        switch (queryType) {
            case 'INSERT':
                message = 'Row(s) inserted successfully';
                break;
            case 'UPDATE':
                message = 'Row(s) updated successfully';
                break;
            case 'DELETE':
                message = 'Row(s) deleted successfully';
                break;
            case 'CREATE':
                message = 'Object created successfully';
                break;
            case 'ALTER':
                message = 'Object altered successfully';
                break;
            case 'DROP':
                message = 'Object dropped successfully';
                icon = '⚠️';
                break;
            case 'TRUNCATE':
                message = 'Table truncated successfully';
                icon = '⚠️';
                break;
            case 'SET':
                message = 'Variable set successfully';
                break;
            case 'TRANSACTION':
                message = 'Transaction started';
                break;
            case 'COMMIT':
                message = 'Transaction committed';
                break;
            case 'ROLLBACK':
                message = 'Transaction rolled back';
                break;
            case 'SELECT':
                message = 'Query executed successfully. No rows returned.';
                break;
            default:
                message = 'Query executed successfully';
        }

        document.getElementById('results-info').textContent = `Query completed in ${elapsed}ms`;
        resultsBody.innerHTML = `
            <div class="success-message">
                <div class="success-icon">${icon}</div>
                <div class="success-text">${message}</div>
                <div class="success-details">Execution time: ${elapsed}ms</div>
            </div>
        `;
        document.getElementById('export-btn').style.display = 'none';
        return;
    }

    const rowCount = data.rows.length;
    document.getElementById('results-info').textContent = `${rowCount} row${rowCount !== 1 ? 's' : ''} returned in ${elapsed}ms`;
    document.getElementById('export-btn').style.display = 'block';

    const columns = data.columns && data.columns.length > 0
        ? data.columns.map(col => col.name || col)
        : Object.keys(data.rows[0]);

    let html = '<table class="results-table"><thead><tr>';
    html += '<th class="row-num">#</th>';
    columns.forEach(col => {
        html += `<th>${escapeHtml(col)}</th>`;
    });
    html += '</tr></thead><tbody>';

    data.rows.forEach((row, idx) => {
        html += '<tr>';
        html += `<td class="row-num">${idx + 1}</td>`;
        columns.forEach(col => {
            const value = row[col];
            html += `<td>${formatCellValue(value, col)}</td>`;
        });
        html += '</tr>';
    });

    html += '</tbody></table>';
    resultsBody.innerHTML = html;

    // Add click-to-copy functionality
    resultsBody.querySelectorAll('td:not(.row-num)').forEach(td => {
        td.addEventListener('click', () => {
            const text = td.textContent;
            navigator.clipboard.writeText(text).then(() => {
                showToast('Copied to clipboard');
            }).catch(() => { });
        });
        td.style.cursor = 'pointer';
        td.title = 'Click to copy';
    });
}

function displayError(message) {
    document.getElementById('results-info').textContent = 'Query failed';
    document.getElementById('results-body').innerHTML = `
        <div class="error-message">
            <div class="error-icon">✕</div>
            <div class="error-title">Query Error</div>
            <div class="error-text">${escapeHtml(message)}</div>
            <div class="error-hint">Check your SQL syntax and try again</div>
        </div>
    `;
    document.getElementById('export-btn').style.display = 'none';
}

// Show toast notification
function showToast(message, duration = 2000) {
    const existing = document.querySelector('.sql-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.className = 'sql-toast';
    toast.textContent = message;
    document.body.appendChild(toast);

    setTimeout(() => toast.classList.add('show'), 10);
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 300);
    }, duration);
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
