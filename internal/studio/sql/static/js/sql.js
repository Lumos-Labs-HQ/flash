/**
 * SQL Editor Main Module
 * Uses CodeMirror 6
 */

const {
  EditorView, EditorState, Compartment,
  keymap, placeholder, lineNumbers, drawSelection, dropCursor,
  highlightActiveLine, highlightActiveLineGutter, highlightSpecialChars, Decoration,
  history, defaultKeymap, historyKeymap, indentWithTab, toggleComment: cmToggleComment,
  sql, bracketMatching, indentOnInput, foldGutter, syntaxHighlighting, defaultHighlightStyle,
  oneDark,
  autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap,
  search, searchKeymap, highlightSelectionMatches,
} = CodeMirror6;



let editor;
let highlightCompartment = new Compartment();
let languageCompartment = new Compartment();
let currentResults = null;
let queryHistory = [];
let historyIndex = -1;

const SQL_STORAGE_KEY = 'flashorm_sql_editor_state';

const DEFAULT_CONTENT = `-- SQL Editor | Ctrl+Enter to run | Ctrl+Space for hints
SELECT * FROM `;

const executeLineDecoration = Decoration.line({ attributes: { class: 'executing-line' } });

// --- Helper functions for CM6 position conversion ---

function offsetToLineCh(doc, offset) {
  const line = doc.lineAt(offset);
  return { line: line.number - 1, ch: offset - line.from };
}

function lineChToOffset(doc, { line, ch }) {
  const l = doc.line(line + 1);
  return l.from + ch;
}

function getValue() {
  return editor.state.doc.toString();
}

function setValue(text) {
  const doc = editor.state.doc;
  editor.dispatch({ changes: { from: 0, to: doc.length, insert: text } });
}

function getSelection() {
  const sel = editor.state.selection.main;
  return editor.state.doc.sliceString(sel.from, sel.to);
}

function somethingSelected() {
  const sel = editor.state.selection.main;
  return sel.from !== sel.to;
}

function getCursor(start) {
  const sel = editor.state.selection.main;
  let pos = start === 'from' ? sel.from : (start === 'to' ? sel.to : sel.head);
  return offsetToLineCh(editor.state.doc, pos);
}

function getLine(n) {
  return editor.state.doc.line(n + 1).text;
}

function lineCount() {
  return editor.state.doc.lines;
}

function setCursor(pos) {
  const offset = lineChToOffset(editor.state.doc, pos);
  editor.dispatch({ selection: { anchor: offset } });
}

function replaceRange(text, from, to) {
  const fromOffset = lineChToOffset(editor.state.doc, from);
  const toOffset = to ? lineChToOffset(editor.state.doc, to) : fromOffset;
  editor.dispatch({ changes: { from: fromOffset, to: toOffset, insert: text } });
}

// Save SQL editor state
function saveSqlState() {
  const state = {
    content: getValue(),
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
        setValue(state.content);
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
  initializeEditor();
  loadSchemaInBackground();
});

// Initialize the CodeMirror 6 editor
function initializeEditor() {
  const customKeymap = keymap.of([
    { key: 'Mod-Enter', run: () => { runQuery(); return true; } },
    { key: 'Ctrl-Enter', run: () => { runQuery(); return true; } },
    { key: 'Ctrl-Up', run: () => { navigateHistory(-1); return true; } },
    { key: 'Ctrl-Down', run: () => { navigateHistory(1); return true; } },
    { key: 'F5', run: () => { runQuery(); return true; } },
    { key: 'Ctrl-/', run: cmToggleComment },
    { key: 'Mod-/', run: cmToggleComment },
  ]);

  let saveTimeout;
  const updateListener = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      clearTimeout(saveTimeout);
      saveTimeout = setTimeout(saveSqlState, 500);
    }
    if (update.selectionSet) {
      updateRunButton();
    }
  });

  editor = new EditorView({
    state: EditorState.create({
      doc: DEFAULT_CONTENT,
      extensions: [
        lineNumbers(),
        highlightActiveLineGutter(),
        highlightSpecialChars(),
        history(),
        foldGutter(),
        dropCursor(),
        indentOnInput(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        bracketMatching(),
        closeBrackets(),
        autocompletion({ override: [SqlHints.sqlCompletionSource] }),
        highlightActiveLine(),
        highlightSelectionMatches(),
        search({ top: true }),
        keymap.of([
          ...defaultKeymap,
          ...historyKeymap,
          ...searchKeymap,
          ...completionKeymap,
          ...closeBracketsKeymap,
          indentWithTab,
        ]),
        customKeymap,
        languageCompartment.of(sql()),
        oneDark,
        updateListener,
        highlightCompartment.of(EditorView.decorations.of(Decoration.none)),
        placeholder('Enter your SQL query here...'),
        EditorState.allowMultipleSelections.of(true),
      ]
    }),
    parent: document.getElementById('sql-editor')
  });

  // Document-level Ctrl+Enter / Cmd+Enter handler — most reliable approach
  document.addEventListener('keydown', function onDocKeyDown(e) {
    if (!editor) return;
    const isCtrlEnter = (e.ctrlKey || e.metaKey) && e.key === 'Enter';
    const isF5 = e.key === 'F5';
    if (!isCtrlEnter && !isF5) return;

    // Only trigger if focus is inside the editor (contenteditable or editor DOM)
    const active = document.activeElement;
    const inEditor = active && (editor.dom === active || editor.dom.contains(active));
    if (!inEditor) return;

    e.preventDefault();
    e.stopPropagation();
    runQuery();
  }, true); // useCapture=true to catch before CM6's handlers

  // Try to restore previous state, otherwise use default content
  if (!restoreSqlState()) {
    setValue(DEFAULT_CONTENT);
  }

  const lastLine = lineCount() - 1;
  const lastLineLength = getLine(lastLine).length;
  setCursor({ line: lastLine, ch: lastLineLength });
  editor.focus();

  window.addEventListener('beforeunload', saveSqlState);

  // Also save state when clicking any navigation link
  document.querySelectorAll('a[href]').forEach(link => {
    link.addEventListener('click', saveSqlState);
  });

  updateRunButton();
  setupResize();
  setupGutterLineSelection();
}

// VS Code-style gutter line selection: click a line number to select the line,
// drag to select multiple lines.
function setupGutterLineSelection() {
  const container = document.getElementById('sql-editor');
  if (!container) return;

  container.addEventListener('mousedown', function onGutterMouseDown(e) {
    if (!editor) return;

    // Only handle clicks inside the line-number gutter
    const gutterEl = e.target.closest('.cm-gutterElement');
    const isLineNumberGutter = e.target.closest('.cm-lineNumbers');
    if (!gutterEl || !isLineNumberGutter) return;

    const lineNum = parseInt(gutterEl.textContent, 10);
    if (isNaN(lineNum) || lineNum < 1 || lineNum > editor.state.doc.lines) return;

    const startLine = editor.state.doc.line(lineNum);
    let lastLineNum = lineNum;

    editor.focus();
    editor.dispatch({
      selection: { anchor: startLine.from, head: startLine.to }
    });

    function onMouseMove(moveEvent) {
      const el = document.elementFromPoint(moveEvent.clientX, moveEvent.clientY);
      if (!el) return;

      const moveGutterEl = el.closest('.cm-gutterElement');
      const moveIsLineNum = el.closest('.cm-lineNumbers');
      if (!moveGutterEl || !moveIsLineNum) return;

      const moveLineNum = parseInt(moveGutterEl.textContent, 10);
      if (isNaN(moveLineNum) || moveLineNum < 1 || moveLineNum > editor.state.doc.lines) return;
      if (moveLineNum === lastLineNum) return;
      lastLineNum = moveLineNum;

      const fromLine = Math.min(lineNum, moveLineNum);
      const toLine = Math.max(lineNum, moveLineNum);
      const from = editor.state.doc.line(fromLine).from;
      const to = editor.state.doc.line(toLine).to;

      editor.dispatch({
        selection: { anchor: from, head: to }
      });
    }

    function onMouseUp() {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    }

    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);

    e.preventDefault();
    e.stopPropagation();
  }, true); // useCapture=true to run before CM6's handler
}

// Load schema in background - doesn't block UI
async function loadSchemaInBackground() {
  await SqlHints.loadEditorHints();
  // CM6 sql() provides generic SQL highlighting; no mode switching needed
}

// Navigate through query history
function navigateHistory(direction) {
  if (queryHistory.length === 0) return;

  historyIndex += direction;
  if (historyIndex < 0) historyIndex = 0;
  if (historyIndex >= queryHistory.length) historyIndex = queryHistory.length - 1;

  setValue(queryHistory[historyIndex]);
  setCursor({ line: lineCount() - 1, ch: getLine(lineCount() - 1).length });
}

// Update Run button text based on whether text is selected
function updateRunButton() {
  const btn = document.getElementById('run-btn');
  const hint = document.getElementById('editor-hint');
  const hasSelection = somethingSelected();

  if (hasSelection) {
    btn.textContent = '▶ Run Selection';
    hint.textContent = 'Running selected lines only • Ctrl+Enter to run';
  } else {
    btn.textContent = '▶ Run All';
    hint.textContent = 'Ctrl+Enter to run all • Select lines to run partial';
  }
}

// Flash-highlight lines that are being executed
function highlightExecutedLines(fromLine, toLine) {
  const decorations = [];
  for (let i = fromLine; i <= toLine; i++) {
    if (i < 0 || i >= lineCount()) continue;
    const line = editor.state.doc.line(i + 1);
    decorations.push(executeLineDecoration.range(line.from));
  }
  editor.dispatch({
    effects: highlightCompartment.reconfigure(
      EditorView.decorations.of(Decoration.set(decorations))
    )
  });
  setTimeout(() => {
    editor.dispatch({
      effects: highlightCompartment.reconfigure(
        EditorView.decorations.of(Decoration.none)
      )
    });
  }, 600);
}

async function runQuery() {
  const selection = getSelection();
  const hasSelection = selection && selection.trim().length > 0;
  let query, fromLine, toLine;

  if (hasSelection) {
    query = selection;
    fromLine = getCursor('from').line;
    toLine = getCursor('to').line;
  } else {
    query = getValue();
    fromLine = 0;
    toLine = lineCount() - 1;
  }

  query = query.trim();
  if (!query) return;

  const cleanQuery = query;

  // Flash the lines being executed
  highlightExecutedLines(fromLine, toLine);

  // Add to history
  if (queryHistory[queryHistory.length - 1] !== query) {
    queryHistory.push(query);
    if (queryHistory.length > 50) queryHistory.shift();
  }
  historyIndex = queryHistory.length;

  const runMode = hasSelection ? 'selection' : 'all';
  document.getElementById('results-info').textContent = hasSelection
    ? `Executing selection (lines ${fromLine + 1}-${toLine + 1})...`
    : 'Executing all queries...';
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

    if (data.success) {
      currentResults = data.data;
      displayResults(data.data, cleanQuery, elapsed, runMode, fromLine, toLine);
      SqlHints.updateSchemaFromQuery(cleanQuery);
    } else {
      displayError(data.message);
    }
  } catch (err) {
    displayError(err.message);
  }
}

// Detect query type (strips leading comments)
function getQueryType(query) {
  // Remove leading comments and whitespace to find the actual query
  let cleaned = query.trim();
  // Strip single-line comments at the start
  while (cleaned.startsWith('--') || cleaned.startsWith('#')) {
    const newline = cleaned.indexOf('\n');
    cleaned = newline >= 0 ? cleaned.slice(newline + 1).trimStart() : '';
  }
  // Strip block comments at the start
  while (cleaned.startsWith('/*')) {
    const end = cleaned.indexOf('*/');
    cleaned = end >= 0 ? cleaned.slice(end + 2).trimStart() : '';
  }
  const upper = cleaned.toUpperCase();
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
function formatCellValue(value) {
  if (value === null || value === undefined) {
    return '<span class="cell-null">NULL</span>';
  }

  if (typeof value === 'boolean') {
    return `<span class="cell-bool">${value ? 'true' : 'false'}</span>`;
  }

  if (typeof value === 'number') {
    return `<span class="cell-number">${value}</span>`;
  }

  if (typeof value === 'object') {
    if (value instanceof Date) {
      return `<span class="cell-date">${value.toISOString()}</span>`;
    }
    try {
      const jsonStr = JSON.stringify(value, null, 2);
      const escaped = escapeHtml(jsonStr);
      return `<span class="cell-json" title="${escaped}">${escapeHtml(JSON.stringify(value))}</span>`;
    } catch {
      return `<span class="cell-object">[Object]</span>`;
    }
  }

  const strValue = String(value);

  // UUID detection
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

  // Long text truncation
  if (strValue.length > 100) {
    const truncated = strValue.substring(0, 100) + '...';
    return `<span class="cell-text cell-truncated" title="${escapeHtml(strValue)}">${escapeHtml(truncated)}</span>`;
  }

  return `<span class="cell-text">${escapeHtml(strValue)}</span>`;
}

function displayResults(data, query, elapsed, runMode, fromLine, toLine) {
  const resultsBody = document.getElementById('results-body');
  const queryType = getQueryType(query);
  const modeLabel = runMode === 'selection'
    ? ` (selection: lines ${fromLine + 1}-${toLine + 1})`
    : '';

  // Handle non-SELECT queries
  if (!data || !data.rows || data.rows.length === 0) {
    let message = '';
    let icon = '✓';

    switch (queryType) {
      case 'INSERT': message = 'Row(s) inserted successfully'; break;
      case 'UPDATE': message = 'Row(s) updated successfully'; break;
      case 'DELETE': message = 'Row(s) deleted successfully'; break;
      case 'CREATE': message = 'Object created successfully'; break;
      case 'ALTER': message = 'Object altered successfully'; break;
      case 'DROP': message = 'Object dropped successfully'; icon = '⚠️'; break;
      case 'TRUNCATE': message = 'Table truncated successfully'; icon = '⚠️'; break;
      case 'SET': message = 'Variable set successfully'; break;
      case 'TRANSACTION': message = 'Transaction started'; break;
      case 'COMMIT': message = 'Transaction committed'; break;
      case 'ROLLBACK': message = 'Transaction rolled back'; break;
      case 'SELECT': message = 'Query executed successfully. No rows returned.'; break;
      default: message = 'Query executed successfully';
    }

    document.getElementById('results-info').textContent = `Query completed in ${elapsed}ms${modeLabel}`;
    resultsBody.innerHTML = `
      <div class="success-message">
        <div class="success-icon">${icon}</div>
        <div class="success-text">${message}</div>
        <div class="success-details">Execution time: ${elapsed}ms${modeLabel}</div>
      </div>
    `;
    document.getElementById('export-btn').style.display = 'none';
    return;
  }

  const rowCount = data.rows.length;
  document.getElementById('results-info').textContent = `${rowCount} row${rowCount !== 1 ? 's' : ''} returned in ${elapsed}ms${modeLabel}`;
  document.getElementById('export-btn').style.display = 'block';

  const columns = data.columns && data.columns.length > 0
    ? data.columns.map(col => col.name || col)
    : Object.keys(data.rows[0]);

  let html = '<table class="results-table"><thead><tr>';
  html += '<th class="row-num-header">#</th>';
  columns.forEach(col => {
    html += `<th>${escapeHtml(col)}</th>`;
  });
  html += '</tr></thead><tbody>';

  data.rows.forEach((row, idx) => {
    html += '<tr>';
    html += `<td class="row-num">${idx + 1}</td>`;
    columns.forEach(col => {
      const value = row[col];
      html += `<td>${formatCellValue(value)}</td>`;
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
        showToast('Copied to clipboard', 'success');
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

function clearEditor() {
  setValue('');
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

  handle.addEventListener('mousedown', () => {
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
