// Redis Studio JavaScript
const REDIS_STORAGE_KEY = 'flashorm_redis_studio_state';

const RedisStudio = {
    currentDb: 0,
    currentKey: null,
    currentKeyTTL: -1,  // Store current key's TTL for preservation
    currentKeyType: 'string',  // Store current key's type
    keys: [],
    commandHistory: [],
    historyIndex: -1,
    currentTab: 'browser',
    terminalInput: null,

    // Save state to sessionStorage
    saveState() {
        const state = {
            currentDb: this.currentDb,
            currentKey: this.currentKey,
            currentTab: this.currentTab,
            commandHistory: this.commandHistory
        };
        try {
            sessionStorage.setItem(REDIS_STORAGE_KEY, JSON.stringify(state));
        } catch (e) {
            console.warn('Failed to save Redis state:', e);
        }
    },

    // Restore state from sessionStorage
    restoreState() {
        try {
            const saved = sessionStorage.getItem(REDIS_STORAGE_KEY);
            if (saved) {
                const state = JSON.parse(saved);
                if (typeof state.currentDb === 'number') {
                    this.currentDb = state.currentDb;
                    const dbSelect = document.getElementById('dbSelect');
                    if (dbSelect) dbSelect.value = state.currentDb;
                }
                if (state.commandHistory) {
                    this.commandHistory = state.commandHistory;
                }
                if (state.currentTab) {
                    this.currentTab = state.currentTab;
                }
                return state;
            }
        } catch (e) {
            console.warn('Failed to restore Redis state:', e);
        }
        return null;
    },

    init() {
        this.bindEvents();

        // Restore previous state
        const savedState = this.restoreState();

        // If we have a saved database, select it first
        if (savedState && savedState.currentDb !== 0) {
            this.selectDatabase(savedState.currentDb, false);  // false = don't save again
        }

        this.loadKeys().then(() => {
            // If we had a selected key, restore it after keys load
            if (savedState && savedState.currentKey) {
                this.selectKey(savedState.currentKey);
            }
        });

        // Switch to saved tab
        if (savedState && savedState.currentTab) {
            this.switchTab(savedState.currentTab);
        }

        this.loadServerInfo();
        this.initTerminal();

        // Save state on navigation
        window.addEventListener('beforeunload', () => this.saveState());
        document.querySelectorAll('a[href]').forEach(link => {
            link.addEventListener('click', () => this.saveState());
        });
    },

    bindEvents() {
        document.getElementById('searchInput')?.addEventListener('input', (e) => this.filterKeys(e.target.value));
        document.getElementById('dbSelect')?.addEventListener('change', (e) => this.selectDatabase(parseInt(e.target.value)));
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => this.switchTab(tab.dataset.tab));
        });
    },

    // ===== REAL TERMINAL =====
    initTerminal() {
        const scroll = document.getElementById('terminalScroll');
        if (!scroll) return;

        // Welcome message
        scroll.innerHTML = '<div class="terminal-welcome">' +
            '<span class="ascii">╦═╗┌─┐┌┬┐┬┌─┐  ╔═╗╦  ╦</span>' +
            '<span class="ascii">╠╦╝├┤  │││└─┐  ║  ║  ║</span>' +
            '<span class="ascii">╩╚═└─┘─┴┘┴└─┘  ╚═╝╩═╝╩</span>' +
            '<div style="margin-top:8px">Type commands and press Enter. Use ↑↓ for history.</div>' +
            '</div>';

        this.createInputLine();
    },

    createInputLine() {
        const scroll = document.getElementById('terminalScroll');
        if (!scroll) return;

        const line = document.createElement('div');
        line.className = 'terminal-line input-line';
        line.innerHTML = '<span class="terminal-prompt">redis[' + this.currentDb + ']&gt;&nbsp;</span>';

        const input = document.createElement('input');
        input.type = 'text';
        input.className = 'terminal-input';
        input.autocomplete = 'off';
        input.spellcheck = false;

        input.addEventListener('keydown', (e) => this.handleTerminalKey(e));

        line.appendChild(input);
        scroll.appendChild(line);

        this.terminalInput = input;
        scroll.scrollTop = scroll.scrollHeight;
    },

    handleTerminalKey(e) {
        if (e.key === 'Enter') {
            const cmd = e.target.value.trim();
            e.target.disabled = true;

            if (cmd) {
                this.commandHistory.push(cmd);
                this.historyIndex = this.commandHistory.length;
                this.executeTerminalCommand(cmd);
            } else {
                this.createInputLine();
                this.focusTerminal();
            }
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (this.historyIndex > 0) {
                this.historyIndex--;
                e.target.value = this.commandHistory[this.historyIndex] || '';
            }
        } else if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (this.historyIndex < this.commandHistory.length - 1) {
                this.historyIndex++;
                e.target.value = this.commandHistory[this.historyIndex] || '';
            } else {
                this.historyIndex = this.commandHistory.length;
                e.target.value = '';
            }
        } else if (e.key === 'l' && e.ctrlKey) {
            e.preventDefault();
            this.clearTerminal();
        }
    },

    async executeTerminalCommand(cmd) {
        // Handle local commands
        if (cmd.toLowerCase() === 'clear' || cmd.toLowerCase() === 'cls') {
            this.clearTerminal();
            return;
        }

        if (cmd.toLowerCase() === 'help') {
            this.appendOutput('Available commands:\n' +
                '  SET key value     - Set a string value\n' +
                '  GET key           - Get value of key\n' +
                '  DEL key           - Delete a key\n' +
                '  KEYS pattern      - Find keys matching pattern\n' +
                '  TTL key           - Get time to live\n' +
                '  EXPIRE key sec    - Set expiration\n' +
                '  LPUSH/RPUSH       - Add to list\n' +
                '  HSET/HGET         - Hash operations\n' +
                '  INFO              - Server info\n' +
                '  FLUSHDB           - Delete all keys in current DB\n' +
                '  clear             - Clear terminal\n' +
                '\nSee https://redis.io/commands for full list', false);
            this.createInputLine();
            this.focusTerminal();
            return;
        }

        try {
            const response = await fetch('/api/cli', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: cmd })
            });

            const json = await response.json();

            if (!json.success) {
                this.appendOutput('(error) ' + (json.message || 'Unknown error'), true);
            } else {
                const data = json.data || {};
                if (data.error) {
                    this.appendOutput('(error) ' + data.error, true);
                } else {
                    this.appendOutput(this.formatResult(data.result), false);
                }
            }
        } catch (error) {
            this.appendOutput('(error) ' + error.message, true);
        }

        this.createInputLine();
        this.focusTerminal();

        // Refresh keys if command modifies data
        const firstWord = cmd.toLowerCase().split(' ')[0];
        if (['set', 'del', 'expire', 'lpush', 'rpush', 'sadd', 'zadd', 'hset', 'hdel', 'flushdb', 'persist', 'rename'].includes(firstWord)) {
            setTimeout(() => this.loadKeys(), 100);
        }
    },

    appendOutput(text, isError) {
        const scroll = document.getElementById('terminalScroll');
        if (!scroll) return;

        const output = document.createElement('div');
        output.className = 'terminal-line terminal-output' + (isError ? ' error' : '');
        output.innerHTML = text;
        scroll.appendChild(output);
        scroll.scrollTop = scroll.scrollHeight;
    },

    formatResult(result) {
        if (result === null || result === undefined) return '<span class="nil">(nil)</span>';
        if (result === 'OK') return '<span class="ok">OK</span>';
        if (typeof result === 'number') return '<span class="integer">(integer) ' + result + '</span>';
        if (typeof result === 'string') return '<span class="str">"' + this.escapeHtml(result) + '"</span>';

        if (Array.isArray(result)) {
            if (result.length === 0) return '(empty array)';
            let html = '';
            for (let i = 0; i < result.length; i++) {
                const item = result[i];
                let val;
                if (item === null) val = '<span class="nil">(nil)</span>';
                else if (typeof item === 'string') val = '<span class="str">"' + this.escapeHtml(item) + '"</span>';
                else if (typeof item === 'number') val = '<span class="integer">' + item + '</span>';
                else val = this.escapeHtml(String(item));
                html += '<span class="idx">' + (i + 1) + ')</span> ' + val + '\n';
            }
            return html.trim();
        }

        if (typeof result === 'object') {
            return this.escapeHtml(JSON.stringify(result, null, 2));
        }

        return String(result);
    },

    clearTerminal() {
        this.initTerminal();
        this.focusTerminal();
    },

    focusTerminal() {
        if (this.terminalInput) {
            this.terminalInput.focus();
        }
    },

    // ===== KEY BROWSER =====
    async loadKeys(pattern) {
        pattern = pattern || '*';
        const container = document.getElementById('keysList');
        if (!container) return;

        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/keys?pattern=' + encodeURIComponent(pattern) + '&count=100');
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const data = json.data || {};
            this.keys = data.keys || [];

            document.getElementById('keysCount').textContent = this.keys.length + ' keys';
            this.renderKeys();
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + this.escapeHtml(error.message) + '</p></div>';
        }
    },

    renderKeys() {
        const container = document.getElementById('keysList');
        if (!container) return;

        if (this.keys.length === 0) {
            container.innerHTML = '<div class="empty-state"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg><h3>No keys</h3><p>Create a new key to get started</p></div>';
            return;
        }

        let html = '';
        for (let i = 0; i < this.keys.length; i++) {
            const key = this.keys[i];
            const isActive = this.currentKey === key.key ? 'active' : '';
            const ttl = key.ttl > 0 ? '<span class="ttl-badge">' + this.formatTTL(key.ttl) + '</span>' : '';
            html += '<div class="key-item ' + isActive + '" onclick="RedisStudio.selectKeyByIndex(' + i + ')">' +
                '<div class="key-name"><span class="key-type ' + key.type + '">' + key.type + '</span><span>' + this.escapeHtml(key.key) + '</span></div>' + ttl + '</div>';
        }
        container.innerHTML = html;
    },

    selectKeyByIndex(index) {
        if (index >= 0 && index < this.keys.length && this.keys[index]) {
            this.selectKey(this.keys[index].key);
        }
    },

    filterKeys(search) {
        this.loadKeys(search ? '*' + search + '*' : '*');
    },

    async selectKey(key) {
        this.currentKey = key;
        this.renderKeys();

        const container = document.getElementById('keyDetail');
        if (!container) return;

        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/key?key=' + encodeURIComponent(key));
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            // Store key metadata for later use (e.g., preserving TTL on save)
            this.currentKeyTTL = json.data.ttl || -1;
            this.currentKeyType = json.data.type || 'string';
            this.renderKeyDetail(json.data);
            // Save state after key selection
            this.saveState();
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + this.escapeHtml(error.message) + '</p></div>';
        }
    },

    renderKeyDetail(data) {
        const container = document.getElementById('keyDetail');
        if (!container) return;

        const ttlText = data.ttl === -1 ? 'No expiry' : data.ttl + 's';
        const ttlClass = data.ttl === -1 ? 'no-expiry' : '';

        container.innerHTML = '<div class="key-header">' +
            '<div class="key-info"><span class="key-type ' + data.type + '">' + data.type + '</span>' +
            '<h2>' + this.escapeHtml(data.key) + '</h2>' +
            '<span class="ttl-badge ' + ttlClass + '">' + ttlText + '</span></div>' +
            '<div class="key-actions">' +
            '<button class="btn btn-sm" onclick="RedisStudio.copyKey()">Copy Key</button>' +
            '<button class="btn btn-sm" onclick="RedisStudio.promptTTL()">Set TTL</button>' +
            '<button class="btn btn-sm btn-danger" onclick="RedisStudio.deleteCurrentKey()">Delete</button>' +
            '</div></div>' +
            '<div class="key-value-container">' + this.renderValue(data) + '</div>';
    },

    renderValue(data) {
        switch (data.type) {
            case 'string':
                return '<div class="value-display"><div class="value-display-header"><span>String (' + (data.value?.length || 0) + ' bytes)</span>' +
                    '<button class="copy-btn" onclick="RedisStudio.copyValue()">Copy</button></div>' +
                    '<div class="value-display-body"><textarea class="value-editor" id="valueEditor">' + this.escapeHtml(data.value || '') + '</textarea></div></div>' +
                    '<div style="margin-top:12px"><button class="btn btn-success" onclick="RedisStudio.saveValue()">Save</button></div>';
            case 'list':
            case 'set':
                const items = Array.isArray(data.value) ? data.value : [];
                let html = '<div class="value-display"><div class="value-display-header"><span>' + data.type + ' (' + items.length + ' items)</span></div><div class="value-display-body"><div class="list-items">';
                items.forEach((item, i) => {
                    html += '<div class="list-item">' + (data.type === 'list' ? '<span class="list-item-index">' + i + '</span>' : '') + '<span class="list-item-value">' + this.escapeHtml(String(item)) + '</span></div>';
                });
                return html + '</div></div></div>';
            case 'zset':
                const zitems = Array.isArray(data.value) ? data.value : [];
                let zhtml = '<div class="value-display"><div class="value-display-header"><span>Sorted Set (' + zitems.length + ')</span></div><div class="value-display-body"><div class="list-items">';
                zitems.forEach(item => {
                    zhtml += '<div class="list-item"><span class="list-item-index">' + (item.score || 0) + '</span><span class="list-item-value">' + this.escapeHtml(String(item.member || '')) + '</span></div>';
                });
                return zhtml + '</div></div></div>';
            case 'hash':
                const entries = data.value ? Object.entries(data.value) : [];
                let hhtml = '<div class="value-display"><div class="value-display-header"><span>Hash (' + entries.length + ' fields)</span></div><div class="value-display-body"><div class="list-items">';
                entries.forEach(([k, v]) => {
                    hhtml += '<div class="hash-item"><span class="hash-item-key">' + this.escapeHtml(k) + '</span><span class="hash-item-value">' + this.escapeHtml(String(v)) + '</span></div>';
                });
                return hhtml + '</div></div></div>';
            default:
                return '<div class="value-display"><div class="value-display-body">' + this.escapeHtml(JSON.stringify(data.value)) + '</div></div>';
        }
    },

    async saveValue() {
        if (!this.currentKey) return;
        const editor = document.getElementById('valueEditor');
        if (!editor) return;

        try {
            // Include TTL in the request to preserve it
            const ttl = this.currentKeyTTL > 0 ? this.currentKeyTTL : 0;

            const response = await fetch('/api/key?key=' + encodeURIComponent(this.currentKey), {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    value: editor.value,
                    type: this.currentKeyType || 'string',
                    ttl: ttl  // Preserve existing TTL
                })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            this.showToast('Value saved', 'success');
            // Refresh the key to get updated metadata
            this.selectKey(this.currentKey);
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    copyValue() {
        const editor = document.getElementById('valueEditor');
        if (editor) { navigator.clipboard.writeText(editor.value); this.showToast('Copied', 'success'); }
    },

    copyKey() {
        if (this.currentKey) { navigator.clipboard.writeText(this.currentKey); this.showToast('Key copied', 'success'); }
    },

    promptTTL() {
        if (!this.currentKey) return;
        const currentTTL = this.currentKeyTTL > 0 ? this.currentKeyTTL : -1;
        const ttl = prompt(`Enter TTL in seconds (current: ${currentTTL === -1 ? 'no expiry' : currentTTL + 's'}). Use -1 to remove expiry:`);
        if (ttl === null) return;
        const n = parseInt(ttl);
        if (isNaN(n)) { this.showToast('Invalid TTL value', 'error'); return; }
        const cmd = n <= 0 ? 'PERSIST ' + this.currentKey : 'EXPIRE ' + this.currentKey + ' ' + n;
        this.runCommand(cmd).then(() => {
            // Update stored TTL
            this.currentKeyTTL = n <= 0 ? -1 : n;
            // Refresh both key detail and keys list to update TTL badges
            this.selectKey(this.currentKey);
            this.loadKeys();  // Refresh sidebar badges
            this.showToast(n <= 0 ? 'TTL removed' : `TTL set to ${n}s`, 'success');
        });
    },

    deleteCurrentKey() {
        if (!this.currentKey || !confirm('Delete "' + this.currentKey + '"?')) return;
        this.deleteKey(this.currentKey);
    },

    async deleteKey(key) {
        try {
            const response = await fetch('/api/key?key=' + encodeURIComponent(key), { method: 'DELETE' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            this.currentKey = null;
            this.loadKeys();
            document.getElementById('keyDetail').innerHTML = '<div class="empty-state"><h3>Select a key</h3></div>';
            this.showToast('Deleted', 'success');
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    async runCommand(cmd) {
        try {
            const response = await fetch('/api/cli', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: cmd })
            });
            return await response.json();
        } catch (e) {
            return { success: false };
        }
    },

    // ===== DATABASE & PURGE =====
    async selectDatabase(db, shouldSave = true) {
        try {
            const response = await fetch('/api/databases/' + db, { method: 'POST' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            this.currentDb = db;
            this.currentKey = null;
            this.loadKeys();
            document.getElementById('keyDetail').innerHTML = '<div class="empty-state"><h3>Select a key</h3></div>';
            // Reinit terminal with new DB
            this.initTerminal();
            this.showToast('Switched to db' + db, 'info');
            // Save state after database change
            if (shouldSave) this.saveState();
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    async flushDatabase() {
        if (!confirm('⚠️ DELETE ALL KEYS in current database (db' + this.currentDb + ')?\n\nThis cannot be undone!')) return;

        try {
            const response = await fetch('/api/flush', { method: 'POST' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            this.currentKey = null;
            this.loadKeys();
            document.getElementById('keyDetail').innerHTML = '<div class="empty-state"><h3>Database purged</h3></div>';
            this.showToast('Database purged', 'success');
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    // ===== TABS =====
    switchTab(tab) {
        this.currentTab = tab;
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        document.querySelector('.tab[data-tab="' + tab + '"]')?.classList.add('active');
        document.querySelectorAll('.tab-content').forEach(c => c.style.display = 'none');
        const el = document.getElementById(tab + 'Tab');
        if (el) el.style.display = 'flex';

        if (tab === 'cli') setTimeout(() => this.focusTerminal(), 50);
        if (tab === 'stats') this.loadServerInfo();

        // Save state after tab change
        this.saveState();
    },

    // ===== SERVER INFO =====
    async loadServerInfo() {
        const container = document.getElementById('statsTab');
        if (!container) return;

        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/info');
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const d = json.data || {};
            container.innerHTML = '<div class="stats-container"><div class="stats-grid">' +
                '<div class="stat-card"><div class="stat-value">' + this.escapeHtml(d.version || 'N/A') + '</div><div class="stat-label">Version</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + this.escapeHtml(d.mode || 'standalone') + '</div><div class="stat-label">Mode</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + (d.connected_clients || 0) + '</div><div class="stat-label">Clients</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + this.escapeHtml(d.used_memory || 'N/A') + '</div><div class="stat-label">Used Memory</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + this.escapeHtml(d.peak_memory || 'N/A') + '</div><div class="stat-label">Peak Memory</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + this.escapeHtml(d.max_memory || 'N/A') + '</div><div class="stat-label">Max Memory</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + (d.total_keys || 0) + '</div><div class="stat-label">Keys</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + this.formatUptime(d.uptime) + '</div><div class="stat-label">Uptime</div></div>' +
                '</div><div class="stats-info"><h4>OS</h4><p>' + this.escapeHtml(d.os || 'N/A') + '</p></div></div>';
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + this.escapeHtml(error.message) + '</p></div>';
        }
    },

    // ===== MODAL =====
    showNewKeyModal() {
        document.getElementById('newKeyModal')?.classList.add('active');
    },

    closeModal(id) {
        document.getElementById(id)?.classList.remove('active');
    },

    async createNewKey() {
        const keyInput = document.getElementById('newKeyName');
        const typeSelect = document.getElementById('newKeyType');
        const valueInput = document.getElementById('newKeyValue');
        const ttlInput = document.getElementById('newKeyTTL');

        if (!keyInput?.value.trim()) { this.showToast('Key name required', 'error'); return; }

        const keyName = keyInput.value.trim();
        const ttl = parseInt(ttlInput?.value || '-1');

        try {
            const response = await fetch('/api/keys', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    key: keyName,
                    value: valueInput?.value || '',
                    type: typeSelect?.value || 'string',
                    ttl: isNaN(ttl) || ttl < 0 ? 0 : ttl
                })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            this.closeModal('newKeyModal');
            this.loadKeys();
            this.selectKey(keyName);
            this.showToast('Created', 'success');
            keyInput.value = '';
            if (valueInput) valueInput.value = '';
            if (ttlInput) ttlInput.value = '-1';
        } catch (error) {
            this.showToast(error.message, 'error');
        }
    },

    // ===== UTILS =====
    formatTTL(s) {
        if (s === -1) return '∞';
        if (s < 60) return s + 's';
        if (s < 3600) return Math.floor(s / 60) + 'm';
        if (s < 86400) return Math.floor(s / 3600) + 'h';
        return Math.floor(s / 86400) + 'd';
    },

    formatUptime(s) {
        if (!s) return 'N/A';
        const d = Math.floor(s / 86400);
        const h = Math.floor((s % 86400) / 3600);
        if (d > 0) return d + 'd ' + h + 'h';
        const m = Math.floor((s % 3600) / 60);
        if (h > 0) return h + 'h ' + m + 'm';
        return m + 'm';
    },

    escapeHtml(str) {
        if (str == null) return '';
        const div = document.createElement('div');
        div.textContent = String(str);
        return div.innerHTML;
    },

    showToast(msg, type) {
        let toast = document.getElementById('toast');
        if (!toast) { toast = document.createElement('div'); toast.id = 'toast'; document.body.appendChild(toast); }
        toast.className = 'toast toast-' + (type || 'info');
        toast.textContent = msg;
        toast.classList.add('show');
        setTimeout(() => toast.classList.remove('show'), 3000);
    }
};

document.addEventListener('DOMContentLoaded', () => RedisStudio.init());
