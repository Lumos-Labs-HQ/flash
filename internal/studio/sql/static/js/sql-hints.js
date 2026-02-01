// Schema cache - populated on page load
let schemaCache = null;
let dbProvider = 'sql';
let schemaLoaded = false;

// All SQL keywords organized by category
const SQL_KEYWORDS_BY_CATEGORY = {
    statements: ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'ALTER', 'DROP', 'TRUNCATE', 'WITH', 'EXPLAIN', 'ANALYZE'],
    clauses: ['FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'FULL', 'CROSS', 'ON', 'AND', 'OR', 'NOT', 'IN', 'EXISTS', 'BETWEEN', 'LIKE', 'ILIKE', 'IS', 'AS', 'USING'],
    ordering: ['ORDER', 'BY', 'ASC', 'DESC', 'NULLS', 'FIRST', 'LAST', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET'],
    functions: ['COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'COALESCE', 'NULLIF', 'CAST', 'EXTRACT', 'NOW', 'CURRENT_TIMESTAMP', 'CURRENT_DATE', 'LOWER', 'UPPER', 'TRIM', 'LENGTH', 'SUBSTRING', 'CONCAT', 'REPLACE', 'ROUND', 'FLOOR', 'CEIL', 'ABS', 'DATE_TRUNC', 'TO_CHAR', 'TO_DATE', 'ARRAY_AGG', 'STRING_AGG', 'JSON_AGG', 'ROW_NUMBER', 'RANK', 'DENSE_RANK', 'LAG', 'LEAD', 'FIRST_VALUE', 'LAST_VALUE'],
    operators: ['NULL', 'TRUE', 'FALSE', 'ALL', 'ANY', 'SOME', 'DISTINCT', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END'],
    setOps: ['UNION', 'INTERSECT', 'EXCEPT'],
    insert: ['INTO', 'VALUES', 'RETURNING', 'DEFAULT'],
    update: ['SET'],
    create: ['TABLE', 'INDEX', 'VIEW', 'SCHEMA', 'DATABASE', 'TYPE', 'FUNCTION', 'PROCEDURE', 'TRIGGER', 'SEQUENCE', 'CONSTRAINT', 'PRIMARY', 'FOREIGN', 'KEY', 'REFERENCES', 'UNIQUE', 'CHECK', 'NOT', 'IF'],
    alter: ['ADD', 'DROP', 'RENAME', 'COLUMN', 'TO', 'MODIFY', 'ALTER'],
    types: ['INTEGER', 'INT', 'BIGINT', 'SMALLINT', 'SERIAL', 'BIGSERIAL', 'NUMERIC', 'DECIMAL', 'REAL', 'FLOAT', 'DOUBLE', 'PRECISION', 'VARCHAR', 'CHAR', 'TEXT', 'BOOLEAN', 'BOOL', 'DATE', 'TIME', 'TIMESTAMP', 'TIMESTAMPTZ', 'INTERVAL', 'UUID', 'JSON', 'JSONB', 'ARRAY', 'BYTEA'],
    transaction: ['BEGIN', 'COMMIT', 'ROLLBACK', 'TRANSACTION', 'SAVEPOINT'],
    window: ['OVER', 'PARTITION', 'WINDOW', 'ROWS', 'RANGE', 'PRECEDING', 'FOLLOWING', 'UNBOUNDED', 'CURRENT', 'ROW'],
    other: ['CASCADE', 'RESTRICT', 'NATURAL', 'LATERAL', 'RECURSIVE', 'TEMPORARY', 'TEMP', 'ONLY', 'VERBOSE', 'CONCURRENTLY', 'NOTHING', 'CONFLICT', 'DO', 'EXCLUDED']
};

// Flatten all keywords
const ALL_SQL_KEYWORDS = Object.values(SQL_KEYWORDS_BY_CATEGORY).flat();

// What can follow each keyword (grammar rules)
const NEXT_KEYWORDS = {
    'SELECT': ['DISTINCT', 'ALL', '*', 'COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'CASE', 'COALESCE', 'NULLIF', 'CAST', 'EXISTS', 'NOT'],
    'FROM': [],  // expects table
    'WHERE': ['NOT', 'EXISTS'],  // expects column or condition
    'AND': ['NOT', 'EXISTS'],
    'OR': ['NOT', 'EXISTS'],
    'JOIN': [],  // expects table
    'LEFT': ['JOIN', 'OUTER'],
    'RIGHT': ['JOIN', 'OUTER'],
    'INNER': ['JOIN'],
    'OUTER': ['JOIN'],
    'FULL': ['OUTER', 'JOIN'],
    'CROSS': ['JOIN'],
    'NATURAL': ['JOIN', 'LEFT', 'RIGHT', 'INNER', 'FULL'],
    'ON': [],  // expects condition
    'ORDER': ['BY'],
    'GROUP': ['BY'],
    'BY': [],  // expects column
    'HAVING': ['NOT', 'EXISTS', 'COUNT', 'SUM', 'AVG', 'MIN', 'MAX'],
    'LIMIT': [],
    'OFFSET': [],
    'ASC': ['NULLS', 'LIMIT', 'OFFSET'],
    'DESC': ['NULLS', 'LIMIT', 'OFFSET'],
    'NULLS': ['FIRST', 'LAST'],
    'UNION': ['ALL', 'SELECT'],
    'INTERSECT': ['ALL', 'SELECT'],
    'EXCEPT': ['ALL', 'SELECT'],
    'INSERT': ['INTO'],
    'INTO': [],  // expects table
    'VALUES': [],
    'UPDATE': [],  // expects table
    'SET': [],  // expects column
    'DELETE': ['FROM'],
    'CREATE': ['TABLE', 'INDEX', 'UNIQUE', 'VIEW', 'SCHEMA', 'DATABASE', 'TEMPORARY', 'TEMP', 'OR', 'IF', 'TYPE', 'FUNCTION', 'PROCEDURE', 'TRIGGER', 'SEQUENCE'],
    'DROP': ['TABLE', 'INDEX', 'VIEW', 'SCHEMA', 'DATABASE', 'COLUMN', 'CONSTRAINT', 'IF', 'TYPE', 'FUNCTION', 'PROCEDURE', 'TRIGGER', 'SEQUENCE'],
    'ALTER': ['TABLE', 'INDEX', 'VIEW', 'SCHEMA', 'DATABASE', 'COLUMN', 'TYPE', 'SEQUENCE'],
    'TABLE': [],  // expects table name
    'ADD': ['COLUMN', 'CONSTRAINT', 'PRIMARY', 'FOREIGN', 'UNIQUE', 'CHECK', 'INDEX', 'IF'],
    'PRIMARY': ['KEY'],
    'FOREIGN': ['KEY'],
    'KEY': ['REFERENCES'],
    'REFERENCES': [],  // expects table
    'UNIQUE': ['INDEX'],
    'CONSTRAINT': [],
    'INDEX': ['ON', 'IF', 'CONCURRENTLY'],
    'VIEW': ['AS', 'IF'],
    'IF': ['NOT', 'EXISTS'],
    'NOT': ['NULL', 'IN', 'EXISTS', 'LIKE', 'ILIKE', 'BETWEEN'],
    'IS': ['NULL', 'NOT', 'TRUE', 'FALSE', 'DISTINCT'],
    'IN': [],
    'BETWEEN': [],
    'LIKE': [],
    'ILIKE': [],
    'EXISTS': [],
    'CASE': ['WHEN'],
    'WHEN': [],  // expects condition
    'THEN': [],
    'ELSE': [],
    'END': ['AS', 'FROM', 'WHERE', 'AND', 'OR', 'ORDER', 'GROUP', 'LIMIT'],
    'AS': [],
    'CAST': [],
    'WITH': ['RECURSIVE'],
    'RECURSIVE': [],
    'BEGIN': ['TRANSACTION', 'WORK'],
    'COMMIT': ['TRANSACTION', 'WORK'],
    'ROLLBACK': ['TRANSACTION', 'WORK', 'TO'],
    'EXPLAIN': ['ANALYZE', 'VERBOSE', 'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'WITH'],
    'ANALYZE': ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'WITH', 'VERBOSE'],
    'RETURNING': ['*'],
    'OVER': [],
    'PARTITION': ['BY'],
    'ON': ['CONFLICT', 'UPDATE', 'DELETE'],
    'CONFLICT': ['DO'],
    'DO': ['NOTHING', 'UPDATE']
};

// Keywords that expect table names after them
const TABLE_CONTEXT = new Set(['FROM', 'JOIN', 'INTO', 'UPDATE', 'TABLE', 'REFERENCES']);

// Keywords that expect column names after them
const COLUMN_CONTEXT = new Set(['SELECT', 'WHERE', 'AND', 'OR', 'SET', 'ON', 'BY', 'HAVING', 'RETURNING', 'ORDER']);

/**
 * Load editor hints from server (called once on page load)
 */
async function loadEditorHints() {
    try {
        const res = await fetch('/api/editor/hints');
        const json = await res.json();
        if (json.success && json.data) {
            schemaCache = json.data.schema || {};
            dbProvider = json.data.provider || 'sql';
            schemaLoaded = true;
            console.log('[SQL Hints] Schema loaded:', Object.keys(schemaCache).length, 'tables, provider:', dbProvider);
        } else {
            console.warn('[SQL Hints] Failed to load hints:', json.message || 'Unknown error');
            schemaCache = {};
            schemaLoaded = true;
        }
    } catch (e) {
        console.error('[SQL Hints] Failed to load editor hints:', e);
        schemaCache = {};
        schemaLoaded = true;
    }
}

/**
 * Get CodeMirror mode based on database provider
 */
function getCodeMirrorMode(provider) {
    switch (provider) {
        case 'postgresql':
        case 'postgres':
            return 'text/x-pgsql';
        case 'mysql':
            return 'text/x-mysql';
        case 'sqlite':
        case 'sqlite3':
            return 'text/x-sqlite';
        default:
            return 'text/x-sql';
    }
}

function getDbProvider() {
    return dbProvider;
}

function isSchemaLoaded() {
    return schemaLoaded;
}

/**
 * Smart SQL hint function - main entry point
 */
function smartSqlHint(cm) {
    const cur = cm.getCursor();
    const line = cm.getLine(cur.line);

    // Skip if in comment
    const token = cm.getTokenAt(cur);
    if (token.type && token.type.includes('comment')) {
        return null;
    }

    // Find word start
    let start = cur.ch;
    while (start > 0 && /[\w]/.test(line.charAt(start - 1))) {
        start--;
    }

    const word = line.substring(start, cur.ch);
    const prefix = word.toLowerCase();

    // Analyze context
    const context = analyzeContext(cm, cur, line);

    // Build completions
    let completions = [];

    // Check for table.column pattern
    const beforeWord = line.substring(0, start);
    const dotMatch = beforeWord.match(/(\w+)\.\s*$/);
    if (dotMatch) {
        const tableOrAlias = dotMatch[1].toLowerCase();
        const tableName = resolveTableOrAlias(cm, tableOrAlias);
        if (tableName) {
            completions = getColumnsForTable(tableName, prefix);
            return formatResult(completions, cur, start);
        }
    }

    // Get completions based on context
    if (context.expectsTable) {
        // After FROM, JOIN, etc. - show tables first, then keywords
        completions.push(...getTableCompletions(prefix));
        completions.push(...getKeywordCompletions(prefix, context.lastKeyword));
    } else if (context.expectsColumn) {
        // After SELECT, WHERE, etc. - show columns first, then tables, then keywords
        completions.push(...getColumnCompletions(prefix, context.tables));
        completions.push(...getTableCompletions(prefix));
        completions.push(...getKeywordCompletions(prefix, context.lastKeyword));
    } else {
        // General context - keywords first, then tables, then columns
        completions.push(...getKeywordCompletions(prefix, context.lastKeyword));
        completions.push(...getTableCompletions(prefix));
        completions.push(...getColumnCompletions(prefix, context.tables));
    }

    return formatResult(completions, cur, start);
}

/**
 * Analyze the SQL context at cursor position
 */
function analyzeContext(cm, cur, line) {
    const fullText = cm.getValue();
    const cursorOffset = cm.indexFromPos(cur);
    const textBefore = fullText.substring(0, cursorOffset);

    // Extract tables from query
    const tables = extractTables(fullText);

    // Find the last SQL keyword before cursor
    const lastKeyword = findLastKeyword(textBefore);

    // Determine what's expected
    const expectsTable = TABLE_CONTEXT.has(lastKeyword);
    const expectsColumn = COLUMN_CONTEXT.has(lastKeyword);

    return {
        lastKeyword,
        tables,
        expectsTable,
        expectsColumn
    };
}

/**
 * Find the last SQL keyword in text
 */
function findLastKeyword(text) {
    const upper = text.toUpperCase();

    // Find all keywords and their positions
    let lastKw = '';
    let lastPos = -1;

    for (const kw of ALL_SQL_KEYWORDS) {
        // Use word boundary regex to find keyword
        const regex = new RegExp(`\\b${kw}\\b`, 'gi');
        let match;
        while ((match = regex.exec(upper)) !== null) {
            if (match.index > lastPos) {
                // Check if there's a non-identifier after this keyword
                const afterKw = upper.substring(match.index + kw.length);
                // If only whitespace or nothing after, this is the last context keyword
                if (/^\s*\w*$/.test(afterKw) || afterKw.length === 0) {
                    lastPos = match.index;
                    lastKw = kw;
                }
                // For keywords that establish context even with content after
                if (TABLE_CONTEXT.has(kw) || COLUMN_CONTEXT.has(kw)) {
                    lastPos = match.index;
                    lastKw = kw;
                }
            }
        }
    }

    return lastKw;
}

/**
 * Extract table names from query
 */
function extractTables(queryText) {
    const tables = [];
    if (!schemaCache) return tables;

    const patterns = [
        /\bFROM\s+(\w+)/gi,
        /\bJOIN\s+(\w+)/gi,
        /\bUPDATE\s+(\w+)/gi,
        /\bINTO\s+(\w+)/gi
    ];

    for (const pattern of patterns) {
        let match;
        while ((match = pattern.exec(queryText)) !== null) {
            const name = match[1].toLowerCase();
            if (schemaCache[name] && !tables.includes(name)) {
                tables.push(name);
            }
        }
    }

    return tables;
}

/**
 * Resolve table name or alias
 */
function resolveTableOrAlias(cm, alias) {
    if (!schemaCache) return null;

    // Direct table name
    if (schemaCache[alias]) {
        return alias;
    }

    // Check for alias: "table alias" or "table AS alias"
    const text = cm.getValue();
    const regex = new RegExp(`(\\w+)\\s+(?:AS\\s+)?${alias}\\b`, 'gi');
    const match = regex.exec(text);
    if (match) {
        const tableName = match[1].toLowerCase();
        if (schemaCache[tableName]) {
            return tableName;
        }
    }

    return null;
}

/**
 * Get keyword completions
 */
function getKeywordCompletions(prefix, lastKeyword) {
    const completions = [];
    const added = new Set();

    const nextKws = NEXT_KEYWORDS[lastKeyword] || [];
    for (const kw of nextKws) {
        if (matchesPrefix(kw, prefix) && !added.has(kw)) {
            added.add(kw);
            completions.push({
                text: kw,
                displayText: kw,
                className: 'hint-grammar',
                hint: applyKeyword
            });
        }
    }

    for (const kw of ALL_SQL_KEYWORDS) {
        if (matchesPrefix(kw, prefix) && !added.has(kw)) {
            added.add(kw);
            completions.push({
                text: kw,
                displayText: kw,
                className: 'hint-keyword',
                hint: applyKeyword
            });
        }
    }

    return completions;
}

/**
 * Get table completions
 */
function getTableCompletions(prefix) {
    if (!schemaCache) return [];

    const completions = [];
    for (const tableName of Object.keys(schemaCache)) {
        if (matchesPrefix(tableName, prefix)) {
            completions.push({
                text: tableName,
                displayText: tableName,
                className: 'hint-table',
                hint: applyTable
            });
        }
    }

    return completions;
}

/**
 * Get column completions for tables in context
 */
function getColumnCompletions(prefix, tables) {
    if (!schemaCache) return [];

    const completions = [];
    const seen = new Set();

    // Search in context tables first, then all tables
    const tablesToSearch = tables.length > 0 ? tables : Object.keys(schemaCache);

    for (const tableName of tablesToSearch) {
        const columns = schemaCache[tableName];
        if (!columns) continue;

        for (const col of columns) {
            const colName = col.name;
            const colLower = colName.toLowerCase();

            if (matchesPrefix(colName, prefix) && !seen.has(colLower)) {
                seen.add(colLower);
                completions.push({
                    text: colName,
                    displayText: `${colName} (${tableName})`,
                    className: 'hint-column',
                    hint: applyColumn
                });
            }
        }
    }

    return completions;
}

/**
 * Get columns for a specific table
 */
function getColumnsForTable(tableName, prefix) {
    if (!schemaCache || !schemaCache[tableName]) return [];

    const completions = [];
    const columns = schemaCache[tableName];

    for (const col of columns) {
        if (matchesPrefix(col.name, prefix)) {
            completions.push({
                text: col.name,
                displayText: `${col.name} : ${col.type}`,
                className: 'hint-column',
                hint: applyColumn
            });
        }
    }

    return completions;
}

/**
 * Check if text matches prefix (case-insensitive)
 */
function matchesPrefix(text, prefix) {
    if (!prefix) return true;  // Empty prefix matches everything
    return text.toLowerCase().startsWith(prefix.toLowerCase());
}

/**
 * Format the hint result
 */
function formatResult(completions, cur, start) {
    if (completions.length === 0) {
        return null;
    }

    // Remove duplicates
    const seen = new Set();
    const unique = completions.filter(c => {
        const key = c.text.toLowerCase();
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
    });

    // Sort: grammar first, then keywords, then tables, then columns
    const order = { 'hint-grammar': 0, 'hint-keyword': 1, 'hint-table': 2, 'hint-column': 3 };
    unique.sort((a, b) => {
        const aOrder = order[a.className] ?? 4;
        const bOrder = order[b.className] ?? 4;
        if (aOrder !== bOrder) return aOrder - bOrder;
        return a.text.localeCompare(b.text);
    });

    // Limit results
    const limited = unique.slice(0, 20);

    return {
        list: limited,
        from: CodeMirror.Pos(cur.line, start),
        to: CodeMirror.Pos(cur.line, cur.ch)
    };
}

/**
 * Apply keyword completion (add space after)
 */
function applyKeyword(cm, data, completion) {
    const text = completion.text;
    const line = cm.getLine(data.to.line);
    const after = line.substring(data.to.ch);

    // Add space if needed
    const needsSpace = !after.startsWith(' ') && !after.startsWith('(') && !after.startsWith('.');
    cm.replaceRange(text + (needsSpace ? ' ' : ''), data.from, data.to);
}

/**
 * Apply table completion
 */
function applyTable(cm, data, completion) {
    const text = completion.text;
    const line = cm.getLine(data.to.line);
    const after = line.substring(data.to.ch);

    // Add space after table name
    const needsSpace = !after.startsWith(' ') && !after.startsWith('.');
    cm.replaceRange(text + (needsSpace ? ' ' : ''), data.from, data.to);
}

/**
 * Apply column completion
 */
function applyColumn(cm, data, completion) {
    cm.replaceRange(completion.text, data.from, data.to);
}

/**
 * Show hint manually
 */
function showSmartHint(cm) {
    cm.showHint({
        hint: smartSqlHint,
        completeSingle: false
    });
}

/**
 * Setup auto-hint on typing
 */
function setupAutoHint(cm) {
    let hintTimeout = null;

    cm.on('inputRead', (editor, change) => {
        // Only trigger on character input or space (for context hints)
        if (change.origin !== '+input') return;

        if (hintTimeout) clearTimeout(hintTimeout);

        // Skip if in comment
        const cur = editor.getCursor();
        const token = editor.getTokenAt(cur);
        if (token.type && token.type.includes('comment')) return;

        // Get current word
        const line = editor.getLine(cur.line);
        let start = cur.ch;
        while (start > 0 && /[\w]/.test(line.charAt(start - 1))) {
            start--;
        }
        const word = line.substring(start, cur.ch);

        // Show hints after typing 1+ character OR after space following a keyword
        const charTyped = change.text[0];
        const shouldShow = word.length >= 1 || (charTyped === ' ' && start === cur.ch);

        if (shouldShow) {
            hintTimeout = setTimeout(() => {
                if (!editor.state.completionActive) {
                    editor.showHint({
                        hint: smartSqlHint,
                        completeSingle: false
                    });
                }
            }, 30);  // Very fast response
        }
    });

    // Also trigger on space after keywords
    cm.on('keyHandled', (editor, name, event) => {
        if (name === 'Space') {
            if (hintTimeout) clearTimeout(hintTimeout);
            hintTimeout = setTimeout(() => {
                if (!editor.state.completionActive) {
                    editor.showHint({
                        hint: smartSqlHint,
                        completeSingle: false
                    });
                }
            }, 50);
        }
    });
}

// Export
window.SqlHints = {
    loadEditorHints,
    getCodeMirrorMode,
    getDbProvider,
    isSchemaLoaded,
    smartSqlHint,
    showSmartHint,
    setupAutoHint
};
