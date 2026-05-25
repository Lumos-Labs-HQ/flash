// Schema cache - populated on page load
let schemaCache = null;
let dbProvider = 'sql';
let schemaLoaded = false;

// ===== Common SQL keywords (all databases) =====
const COMMON_KEYWORDS = {
  statements: ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'ALTER', 'DROP', 'TRUNCATE', 'WITH', 'EXPLAIN', 'ANALYZE'],
  clauses: ['FROM', 'WHERE', 'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'FULL', 'CROSS', 'ON', 'AND', 'OR', 'NOT', 'IN', 'EXISTS', 'BETWEEN', 'LIKE', 'IS', 'AS', 'USING'],
  ordering: ['ORDER', 'BY', 'ASC', 'DESC', 'GROUP', 'HAVING', 'LIMIT', 'OFFSET'],
  functions: ['COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'COALESCE', 'NULLIF', 'CAST', 'LOWER', 'UPPER', 'TRIM', 'LENGTH', 'SUBSTRING', 'CONCAT', 'REPLACE', 'ROUND', 'FLOOR', 'CEIL', 'ABS', 'ROW_NUMBER', 'RANK', 'DENSE_RANK', 'LAG', 'LEAD', 'FIRST_VALUE', 'LAST_VALUE'],
  operators: ['NULL', 'TRUE', 'FALSE', 'ALL', 'ANY', 'SOME', 'DISTINCT', 'CASE', 'WHEN', 'THEN', 'ELSE', 'END'],
  setOps: ['UNION', 'INTERSECT', 'EXCEPT'],
  insert: ['INTO', 'VALUES', 'DEFAULT'],
  update: ['SET'],
  create: ['TABLE', 'INDEX', 'VIEW', 'SCHEMA', 'DATABASE', 'CONSTRAINT', 'PRIMARY', 'FOREIGN', 'KEY', 'REFERENCES', 'UNIQUE', 'CHECK', 'NOT', 'IF'],
  alter: ['ADD', 'DROP', 'RENAME', 'COLUMN', 'TO', 'ALTER'],
  types: ['INTEGER', 'INT', 'BIGINT', 'SMALLINT', 'NUMERIC', 'DECIMAL', 'REAL', 'FLOAT', 'DOUBLE', 'VARCHAR', 'CHAR', 'TEXT', 'BOOLEAN', 'DATE', 'TIME', 'TIMESTAMP'],
  transaction: ['BEGIN', 'COMMIT', 'ROLLBACK', 'TRANSACTION', 'SAVEPOINT'],
  window: ['OVER', 'PARTITION', 'WINDOW', 'ROWS', 'RANGE', 'PRECEDING', 'FOLLOWING', 'UNBOUNDED', 'CURRENT', 'ROW'],
  other: ['CASCADE', 'RESTRICT', 'NATURAL']
};

// ===== PostgreSQL-specific keywords =====
const POSTGRES_KEYWORDS = {
  clauses: ['ILIKE', 'RETURNING', 'LATERAL', 'RECURSIVE'],
  functions: ['NOW', 'CURRENT_TIMESTAMP', 'CURRENT_DATE', 'EXTRACT', 'DATE_TRUNC', 'TO_CHAR', 'TO_DATE', 'TO_NUMBER', 'ARRAY_AGG', 'STRING_AGG', 'JSON_AGG', 'JSONB_AGG', 'JSON_BUILD_OBJECT', 'JSONB_BUILD_OBJECT', 'GEN_RANDOM_UUID', 'GENERATE_SERIES', 'UNNEST', 'ARRAY_LENGTH'],
  types: ['SERIAL', 'BIGSERIAL', 'SMALLSERIAL', 'TIMESTAMPTZ', 'INTERVAL', 'UUID', 'JSON', 'JSONB', 'ARRAY', 'BYTEA', 'BOOL', 'PRECISION', 'MONEY', 'INET', 'CIDR', 'MACADDR'],
  ordering: ['NULLS', 'FIRST', 'LAST'],
  create: ['TYPE', 'FUNCTION', 'PROCEDURE', 'TRIGGER', 'SEQUENCE', 'EXTENSION', 'MATERIALIZED'],
  other: ['CONCURRENTLY', 'NOTHING', 'CONFLICT', 'DO', 'EXCLUDED', 'VERBOSE', 'ONLY', 'TEMPORARY', 'TEMP', 'INHERITS', 'TABLESPACE']
};

// ===== MySQL-specific keywords =====
const MYSQL_KEYWORDS = {
  statements: ['SHOW', 'DESCRIBE', 'USE', 'LOAD'],
  clauses: ['STRAIGHT_JOIN', 'HIGH_PRIORITY', 'LOW_PRIORITY', 'DELAYED', 'IGNORE'],
  functions: ['NOW', 'CURRENT_TIMESTAMP', 'CURRENT_DATE', 'IFNULL', 'IF', 'GROUP_CONCAT', 'DATE_FORMAT', 'STR_TO_DATE', 'UNIX_TIMESTAMP', 'FROM_UNIXTIME', 'FOUND_ROWS', 'LAST_INSERT_ID', 'UUID', 'CONCAT_WS', 'DATE_ADD', 'DATE_SUB', 'DATEDIFF'],
  types: ['TINYINT', 'MEDIUMINT', 'DATETIME', 'BLOB', 'TINYBLOB', 'MEDIUMBLOB', 'LONGBLOB', 'TINYTEXT', 'MEDIUMTEXT', 'LONGTEXT', 'ENUM', 'JSON', 'UNSIGNED', 'SIGNED', 'BINARY', 'VARBINARY'],
  create: ['FUNCTION', 'PROCEDURE', 'TRIGGER', 'EVENT'],
  alter: ['MODIFY', 'CHANGE'],
  other: ['AUTO_INCREMENT', 'ENGINE', 'CHARSET', 'CHARACTER', 'COLLATE', 'COMMENT', 'DATABASES', 'TABLES', 'COLUMNS', 'PROCESSLIST', 'STATUS', 'VARIABLES', 'GRANTS', 'DUPLICATE']
};

// ===== SQLite-specific keywords =====
const SQLITE_KEYWORDS = {
  statements: ['VACUUM', 'REINDEX', 'PRAGMA', 'ATTACH', 'DETACH'],
  clauses: ['GLOB'],
  functions: ['CURRENT_TIMESTAMP', 'CURRENT_DATE', 'CURRENT_TIME', 'TYPEOF', 'TOTAL', 'GROUP_CONCAT', 'SUBSTR', 'INSTR', 'PRINTF', 'UNICODE', 'ZEROBLOB', 'RANDOMBLOB', 'HEX', 'QUOTE', 'LAST_INSERT_ROWID'],
  types: ['BLOB'],
  other: ['AUTOINCREMENT', 'ROWID', 'WITHOUT', 'STRICT']
};

// All keywords combined (used for context detection only)
const ALL_SQL_KEYWORDS = [
  ...new Set([
    ...Object.values(COMMON_KEYWORDS).flat(),
    ...Object.values(POSTGRES_KEYWORDS).flat(),
    ...Object.values(MYSQL_KEYWORDS).flat(),
    ...Object.values(SQLITE_KEYWORDS).flat()
  ])
];

/**
 * Get active keywords for the current database provider
 */
function getActiveKeywords() {
  const common = Object.values(COMMON_KEYWORDS).flat();
  let extra = [];

  switch (dbProvider) {
    case 'postgresql':
    case 'postgres':
      extra = Object.values(POSTGRES_KEYWORDS).flat();
      break;
    case 'mysql':
      extra = Object.values(MYSQL_KEYWORDS).flat();
      break;
    case 'sqlite':
    case 'sqlite3':
      extra = Object.values(SQLITE_KEYWORDS).flat();
      break;
    default:
      extra = [
        ...Object.values(POSTGRES_KEYWORDS).flat(),
        ...Object.values(MYSQL_KEYWORDS).flat(),
        ...Object.values(SQLITE_KEYWORDS).flat()
      ];
  }

  return [...new Set([...common, ...extra])];
}

// What can follow each keyword (grammar rules)
const NEXT_KEYWORDS = {
  'SELECT': ['DISTINCT', 'ALL', '*', 'COUNT', 'SUM', 'AVG', 'MIN', 'MAX', 'CASE', 'COALESCE', 'NULLIF', 'CAST', 'EXISTS', 'NOT'],
  'FROM': [],
  'WHERE': ['NOT', 'EXISTS'],
  'AND': ['NOT', 'EXISTS'],
  'OR': ['NOT', 'EXISTS'],
  'JOIN': [],
  'LEFT': ['JOIN', 'OUTER'],
  'RIGHT': ['JOIN', 'OUTER'],
  'INNER': ['JOIN'],
  'OUTER': ['JOIN'],
  'FULL': ['OUTER', 'JOIN'],
  'CROSS': ['JOIN'],
  'NATURAL': ['JOIN', 'LEFT', 'RIGHT', 'INNER', 'FULL'],
  'ON': [],
  'ORDER': ['BY'],
  'GROUP': ['BY'],
  'BY': [],
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
  'INTO': [],
  'VALUES': [],
  'UPDATE': [],
  'SET': [],
  'DELETE': ['FROM'],
  'CREATE': ['TABLE', 'INDEX', 'UNIQUE', 'VIEW', 'SCHEMA', 'DATABASE', 'TEMPORARY', 'TEMP', 'OR', 'IF', 'TYPE', 'FUNCTION', 'PROCEDURE', 'TRIGGER', 'SEQUENCE', 'EXTENSION', 'MATERIALIZED'],
  'DROP': ['TABLE', 'INDEX', 'VIEW', 'SCHEMA', 'DATABASE', 'COLUMN', 'CONSTRAINT', 'IF', 'TYPE', 'FUNCTION', 'PROCEDURE', 'TRIGGER', 'SEQUENCE', 'EXTENSION'],
  'ALTER': ['TABLE', 'INDEX', 'VIEW', 'SCHEMA', 'DATABASE', 'COLUMN', 'TYPE', 'SEQUENCE'],
  'TABLE': [],
  'ADD': ['COLUMN', 'CONSTRAINT', 'PRIMARY', 'FOREIGN', 'UNIQUE', 'CHECK', 'INDEX', 'IF'],
  'PRIMARY': ['KEY'],
  'FOREIGN': ['KEY'],
  'KEY': ['REFERENCES'],
  'REFERENCES': [],
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
  'WHEN': [],
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
  'ON CONFLICT': ['DO'],
  'CONFLICT': ['DO'],
  'DO': ['NOTHING', 'UPDATE']
};

// Keywords that expect table names after them
const TABLE_CONTEXT = new Set(['FROM', 'JOIN', 'INTO', 'UPDATE', 'TABLE', 'REFERENCES']);

// Keywords that expect column names after them
const COLUMN_CONTEXT = new Set(['SELECT', 'WHERE', 'AND', 'OR', 'SET', 'ON', 'BY', 'HAVING', 'RETURNING', 'ORDER']);

// --- Helper: get editor doc ---
function getDoc() {
  return editor && editor.state ? editor.state.doc : null;
}

function getEditorText() {
  const doc = getDoc();
  return doc ? doc.toString() : '';
}

function getEditorLine(n) {
  const doc = getDoc();
  return doc ? doc.line(n + 1).text : '';
}

function offsetToLineCh(doc, offset) {
  const line = doc.lineAt(offset);
  return { line: line.number - 1, ch: offset - line.from };
}

function lineChToOffset(doc, { line, ch }) {
  const l = doc.line(line + 1);
  return l.from + ch;
}

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
 * Get CodeMirror mode based on database provider (legacy, not used in CM6)
 */
function getCodeMirrorMode(provider) {
  // CM6 sql() provides generic SQL highlighting for all dialects
  return 'text/x-sql';
}

function getDbProvider() {
  return dbProvider;
}

function isSchemaLoaded() {
  return schemaLoaded;
}

/**
 * CM6 CompletionSource - main entry point for autocomplete
 */
function sqlCompletionSource(context) {
  const doc = context.state.doc;
  const pos = context.pos;
  const line = doc.lineAt(pos);
  const lineText = line.text;
  const ch = pos - line.from;

  // Skip if in comment (simple check)
  const beforeCursor = lineText.substring(0, ch);
  if (beforeCursor.trimStart().startsWith('--')) {
    return null;
  }

  // Find word start
  let start = ch;
  while (start > 0 && /[\w]/.test(lineText.charAt(start - 1))) {
    start--;
  }

  const word = lineText.substring(start, ch);
  const prefix = word.toLowerCase();

  // Analyze context
  const cursorPos = offsetToLineCh(doc, pos);
  const contextInfo = analyzeContext(cursorPos);

  // Build completions
  let completions = [];

  // Check for table.column pattern
  const beforeWord = lineText.substring(0, start);
  const dotMatch = beforeWord.match(/(\w+)\.\s*$/);
  if (dotMatch) {
    const tableOrAlias = dotMatch[1].toLowerCase();
    const tableName = resolveTableOrAlias(tableOrAlias);
    if (tableName) {
      completions = getColumnsForTable(tableName, prefix);
      return formatCompletionResult(completions, line.from + start, pos);
    }
  }

  // Get completions based on context
  if (contextInfo.expectsTable) {
    completions.push(...getTableCompletions(prefix));
    completions.push(...getKeywordCompletions(prefix, contextInfo.lastKeyword));
  } else if (contextInfo.expectsColumn) {
    completions.push(...getColumnCompletions(prefix, contextInfo.tables));
    completions.push(...getTableCompletions(prefix));
    completions.push(...getKeywordCompletions(prefix, contextInfo.lastKeyword));
  } else {
    completions.push(...getKeywordCompletions(prefix, contextInfo.lastKeyword));
    completions.push(...getTableCompletions(prefix));
    completions.push(...getColumnCompletions(prefix, contextInfo.tables));
  }

  return formatCompletionResult(completions, line.from + start, pos);
}

/**
 * Analyze the SQL context at cursor position
 */
function analyzeContext(cur) {
  const fullText = getEditorText();
  const cursorOffset = lineChToOffset(getDoc(), cur);
  const textBefore = fullText.substring(0, cursorOffset);

  const tables = extractTables(fullText);
  const lastKeyword = findLastKeyword(textBefore);

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

  let lastKw = '';
  let lastPos = -1;

  for (const kw of ALL_SQL_KEYWORDS) {
    const regex = new RegExp(`\\b${kw}\\b`, 'gi');
    let match;
    while ((match = regex.exec(upper)) !== null) {
      if (match.index > lastPos) {
        const afterKw = upper.substring(match.index + kw.length);
        if (/^\s*\w*$/.test(afterKw) || afterKw.length === 0) {
          lastPos = match.index;
          lastKw = kw;
        }
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
function resolveTableOrAlias(alias) {
  if (!schemaCache) return null;

  if (schemaCache[alias]) {
    return alias;
  }

  const text = getEditorText();
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
  const activeSet = new Set(getActiveKeywords());

  // Grammar suggestions first
  const nextKws = NEXT_KEYWORDS[lastKeyword] || [];
  for (const kw of nextKws) {
    if (activeSet.has(kw) && matchesPrefix(kw, prefix) && !added.has(kw)) {
      added.add(kw);
      completions.push({
        text: kw,
        displayText: kw,
        className: 'hint-grammar',
        apply: applyKeyword
      });
    }
  }

  // Then active keywords
  for (const kw of activeSet) {
    if (matchesPrefix(kw, prefix) && !added.has(kw)) {
      added.add(kw);
      completions.push({
        text: kw,
        displayText: kw,
        className: 'hint-keyword',
        apply: applyKeyword
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
        apply: applyTable
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
          apply: applyColumn
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
        apply: applyColumn
      });
    }
  }

  return completions;
}

/**
 * Check if text matches prefix (case-insensitive)
 */
function matchesPrefix(text, prefix) {
  if (!prefix) return true;
  return text.toLowerCase().startsWith(prefix.toLowerCase());
}

/**
 * Format completions for CM6
 */
function formatCompletionResult(completions, from, to) {
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
    from,
    to,
    options: limited.map(c => ({
      label: c.text,
      detail: c.displayText !== c.text ? c.displayText : undefined,
      type: c.className === 'hint-keyword' ? 'keyword' :
            c.className === 'hint-table' ? 'table' :
            c.className === 'hint-column' ? 'column' : 'grammar',
      apply: c.apply
    }))
  };
}

/**
 * Apply keyword completion
 */
function applyKeyword(view, completion, from, to) {
  const after = view.state.doc.sliceString(to, Math.min(to + 20, view.state.doc.length));
  const needsSpace = !after.startsWith(' ') && !after.startsWith('(') && !after.startsWith('.');
  view.dispatch({
    changes: { from, to, insert: completion.label + (needsSpace ? ' ' : '') },
    selection: { anchor: from + completion.label.length + (needsSpace ? 1 : 0) }
  });
}

/**
 * Apply table completion
 */
function applyTable(view, completion, from, to) {
  const after = view.state.doc.sliceString(to, Math.min(to + 20, view.state.doc.length));
  const needsSpace = !after.startsWith(' ') && !after.startsWith('.');
  view.dispatch({
    changes: { from, to, insert: completion.label + (needsSpace ? ' ' : '') },
    selection: { anchor: from + completion.label.length + (needsSpace ? 1 : 0) }
  });
}

/**
 * Apply column completion
 */
function applyColumn(view, completion, from, to) {
  view.dispatch({
    changes: { from, to, insert: completion.label }
  });
}

// ===== Dynamic Schema Update from DDL =====

function unquoteIdentifier(id) {
  if (!id) return '';
  id = id.trim();
  if ((id.startsWith('"') && id.endsWith('"')) ||
      (id.startsWith('`') && id.endsWith('`'))) {
    id = id.slice(1, -1);
  }
  return id.toLowerCase();
}

function extractBaseType(typeStr) {
  typeStr = typeStr.trim();
  const upper = typeStr.toUpperCase();

  const multiWordTypes = [
    'DOUBLE PRECISION', 'CHARACTER VARYING',
    'TIMESTAMP WITH', 'TIMESTAMP WITHOUT',
    'TIME WITH', 'TIME WITHOUT'
  ];
  for (const mwt of multiWordTypes) {
    if (upper.startsWith(mwt)) return mwt;
  }

  const match = typeStr.match(/^(\w+)/);
  return match ? match[1].toUpperCase() : typeStr.toUpperCase();
}

function extractParenBody(sql, startIndex) {
  if (sql[startIndex] !== '(') return null;
  let depth = 0;
  for (let i = startIndex; i < sql.length; i++) {
    if (sql[i] === '(') depth++;
    else if (sql[i] === ')') {
      depth--;
      if (depth === 0) return sql.substring(startIndex + 1, i);
    }
  }
  return sql.substring(startIndex + 1);
}

function splitTopLevel(str) {
  const parts = [];
  let depth = 0;
  let current = '';
  for (let i = 0; i < str.length; i++) {
    const ch = str[i];
    if (ch === '(') depth++;
    else if (ch === ')') depth--;
    else if (ch === ',' && depth === 0) {
      parts.push(current);
      current = '';
      continue;
    }
    current += ch;
  }
  if (current.trim()) parts.push(current);
  return parts;
}

function parseColumnDefs(body) {
  const columns = [];
  const parts = splitTopLevel(body);
  const constraintPrefixes = new Set([
    'PRIMARY', 'FOREIGN', 'CONSTRAINT', 'UNIQUE', 'CHECK', 'INDEX', 'KEY', 'EXCLUDE'
  ]);

  for (const part of parts) {
    const trimmed = part.trim();
    if (!trimmed) continue;

    const firstWord = trimmed.split(/\s+/)[0].toUpperCase();
    if (constraintPrefixes.has(firstWord)) continue;

    const colMatch = trimmed.match(/^(["`]?[\w]+["`]?)\s+(\S+(?:\s*\([^)]*\))?)/i);
    if (colMatch) {
      columns.push({
        name: unquoteIdentifier(colMatch[1]),
        type: extractBaseType(colMatch[2])
      });
    }
  }
  return columns;
}

function handleCreateTable(sql) {
  const match = sql.match(
    /^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(["`]?[\w.]+["`]?)\s*\(/i
  );
  if (!match) return;

  const tableName = unquoteIdentifier(match[1]);
  const parenStart = sql.indexOf('(', match[0].length - 1);
  const body = extractParenBody(sql, parenStart);
  if (!body) return;

  const columns = parseColumnDefs(body);
  schemaCache[tableName] = columns;
  console.log('[SQL Hints] Schema updated: CREATE TABLE', tableName, '(' + columns.length + ' columns)');
}

function handleDropTable(sql) {
  const match = sql.match(
    /^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(.+?)(?:\s+CASCADE|\s+RESTRICT)?\s*$/i
  );
  if (!match) return;

  const tableList = match[1].split(',');
  for (const t of tableList) {
    const tableName = unquoteIdentifier(t.trim());
    if (tableName && schemaCache[tableName]) {
      delete schemaCache[tableName];
      console.log('[SQL Hints] Schema updated: DROP TABLE', tableName);
    }
  }
}

function handleAlterAddColumn(tableName, action) {
  if (!schemaCache[tableName]) schemaCache[tableName] = [];

  const match = action.match(
    /^ADD\s+(?:COLUMN\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(["`]?[\w]+["`]?)\s+(\S+(?:\s*\([^)]*\))?)/i
  );
  if (!match) return;

  const colName = unquoteIdentifier(match[1]);
  const colType = extractBaseType(match[2]);

  const exists = schemaCache[tableName].some(c => c.name.toLowerCase() === colName);
  if (!exists) {
    schemaCache[tableName].push({ name: colName, type: colType });
    console.log('[SQL Hints] Schema updated: ADD COLUMN', tableName + '.' + colName, colType);
  }
}

function handleAlterDropColumn(tableName, action) {
  if (!schemaCache[tableName]) return;

  const match = action.match(
    /^DROP\s+(?:COLUMN\s+)?(?:IF\s+EXISTS\s+)?(["`]?[\w]+["`]?)/i
  );
  if (!match) return;

  const colName = unquoteIdentifier(match[1]);
  schemaCache[tableName] = schemaCache[tableName].filter(
    c => c.name.toLowerCase() !== colName
  );
  console.log('[SQL Hints] Schema updated: DROP COLUMN', tableName + '.' + colName);
}

function handleAlterRename(tableName, action) {
  if (/^RENAME\s+TO\s+/i.test(action)) {
    const match = action.match(/^RENAME\s+TO\s+(["`]?[\w]+["`]?)/i);
    if (match && schemaCache[tableName]) {
      const newName = unquoteIdentifier(match[1]);
      schemaCache[newName] = schemaCache[tableName];
      delete schemaCache[tableName];
      console.log('[SQL Hints] Schema updated: RENAME TABLE', tableName, '->', newName);
    }
    return;
  }

  const colMatch = action.match(
    /^RENAME\s+(?:COLUMN\s+)?(["`]?[\w]+["`]?)\s+TO\s+(["`]?[\w]+["`]?)/i
  );
  if (colMatch && schemaCache[tableName]) {
    const oldName = unquoteIdentifier(colMatch[1]);
    const newName = unquoteIdentifier(colMatch[2]);
    for (const col of schemaCache[tableName]) {
      if (col.name.toLowerCase() === oldName) {
        col.name = newName;
        console.log('[SQL Hints] Schema updated: RENAME COLUMN', tableName + '.' + oldName, '->', newName);
        break;
      }
    }
  }
}

function handleAlterTable(sql) {
  const tableMatch = sql.match(
    /^ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?(["`]?[\w.]+["`]?)\s+(.+)$/i
  );
  if (!tableMatch) return;

  const tableName = unquoteIdentifier(tableMatch[1]);
  const action = tableMatch[2].trim();
  const upperAction = action.toUpperCase();

  if (upperAction.startsWith('ADD COLUMN') || upperAction.startsWith('ADD ')) {
    handleAlterAddColumn(tableName, action);
  } else if (upperAction.startsWith('DROP COLUMN') ||
             (upperAction.startsWith('DROP ') && !upperAction.startsWith('DROP CONSTRAINT'))) {
    handleAlterDropColumn(tableName, action);
  } else if (upperAction.startsWith('RENAME')) {
    handleAlterRename(tableName, action);
  }
}

function updateSchemaFromQuery(query) {
  if (!schemaCache) return;

  const normalized = query.trim().replace(/;\s*$/, '');
  const upper = normalized.toUpperCase();

  if (upper.startsWith('CREATE TABLE')) {
    handleCreateTable(normalized);
  } else if (upper.startsWith('DROP TABLE')) {
    handleDropTable(normalized);
  } else if (upper.startsWith('ALTER TABLE')) {
    handleAlterTable(normalized);
  }
}

// Export
window.SqlHints = {
  loadEditorHints,
  getCodeMirrorMode,
  getDbProvider,
  isSchemaLoaded,
  sqlCompletionSource,
  updateSchemaFromQuery
};
