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
        if (typeof result === 'string') return '<span class="str">"' + escapeHtml(result) + '"</span>';

        if (Array.isArray(result)) {
            if (result.length === 0) return '(empty array)';
            let html = '';
            for (let i = 0; i < result.length; i++) {
                const item = result[i];
                let val;
                if (item === null) val = '<span class="nil">(nil)</span>';
                else if (typeof item === 'string') val = '<span class="str">"' + escapeHtml(item) + '"</span>';
                else if (typeof item === 'number') val = '<span class="integer">' + item + '</span>';
                else val = escapeHtml(String(item));
                html += '<span class="idx">' + (i + 1) + ')</span> ' + val + '\n';
            }
            return html.trim();
        }

        if (typeof result === 'object') {
            return escapeHtml(JSON.stringify(result, null, 2));
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
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
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
                '<div class="key-name"><span class="key-type ' + key.type + '">' + key.type + '</span><span>' + escapeHtml(key.key) + '</span></div>' + ttl + '</div>';
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
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
        }
    },

    renderKeyDetail(data) {
        const container = document.getElementById('keyDetail');
        if (!container) return;

        const ttlText = data.ttl === -1 ? 'No expiry' : data.ttl + 's';
        const ttlClass = data.ttl === -1 ? 'no-expiry' : '';

        container.innerHTML = '<div class="key-header">' +
            '<div class="key-info"><span class="key-type ' + data.type + '">' + data.type + '</span>' +
            '<h2>' + escapeHtml(data.key) + '</h2>' +
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
                    '<div class="value-display-body"><textarea class="value-editor" id="valueEditor">' + escapeHtml(data.value || '') + '</textarea></div></div>' +
                    '<div style="margin-top:12px"><button class="btn btn-success" onclick="RedisStudio.saveValue()">Save</button></div>';
            case 'list':
            case 'set':
                const items = Array.isArray(data.value) ? data.value : [];
                let html = '<div class="value-display"><div class="value-display-header"><span>' + data.type + ' (' + items.length + ' items)</span></div><div class="value-display-body"><div class="list-items">';
                items.forEach((item, i) => {
                    html += '<div class="list-item">' + (data.type === 'list' ? '<span class="list-item-index">' + i + '</span>' : '') + '<span class="list-item-value">' + escapeHtml(String(item)) + '</span></div>';
                });
                return html + '</div></div></div>';
            case 'zset':
                const zitems = Array.isArray(data.value) ? data.value : [];
                let zhtml = '<div class="value-display"><div class="value-display-header"><span>Sorted Set (' + zitems.length + ')</span></div><div class="value-display-body"><div class="list-items">';
                zitems.forEach(item => {
                    zhtml += '<div class="list-item"><span class="list-item-index">' + (item.score || 0) + '</span><span class="list-item-value">' + escapeHtml(String(item.member || '')) + '</span></div>';
                });
                return zhtml + '</div></div></div>';
            case 'hash':
                const entries = data.value ? Object.entries(data.value) : [];
                let hhtml = '<div class="value-display"><div class="value-display-header"><span>Hash (' + entries.length + ' fields)</span></div><div class="value-display-body"><div class="list-items">';
                entries.forEach(([k, v]) => {
                    hhtml += '<div class="hash-item"><span class="hash-item-key">' + escapeHtml(k) + '</span><span class="hash-item-value">' + escapeHtml(String(v)) + '</span></div>';
                });
                return hhtml + '</div></div></div>';
            default:
                return '<div class="value-display"><div class="value-display-body">' + escapeHtml(JSON.stringify(data.value)) + '</div></div>';
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
            showToast('Value saved', 'success');
            // Refresh the key to get updated metadata
            this.selectKey(this.currentKey);
        } catch (error) {
            showToast(error.message, 'error');
        }
    },

    copyValue() {
        const editor = document.getElementById('valueEditor');
        if (editor) { navigator.clipboard.writeText(editor.value); showToast('Copied', 'success'); }
    },

    copyKey() {
        if (this.currentKey) { navigator.clipboard.writeText(this.currentKey); showToast('Key copied', 'success'); }
    },

    promptTTL() {
        if (!this.currentKey) return;
        const currentTTL = this.currentKeyTTL > 0 ? this.currentKeyTTL : -1;
        const ttl = prompt(`Enter TTL in seconds (current: ${currentTTL === -1 ? 'no expiry' : currentTTL + 's'}). Use -1 to remove expiry:`);
        if (ttl === null) return;
        const n = parseInt(ttl);
        if (isNaN(n)) { showToast('Invalid TTL value', 'error'); return; }
        const cmd = n <= 0 ? 'PERSIST ' + this.currentKey : 'EXPIRE ' + this.currentKey + ' ' + n;
        this.runCommand(cmd).then(() => {
            // Update stored TTL
            this.currentKeyTTL = n <= 0 ? -1 : n;
            // Refresh both key detail and keys list to update TTL badges
            this.selectKey(this.currentKey);
            this.loadKeys();  // Refresh sidebar badges
            showToast(n <= 0 ? 'TTL removed' : `TTL set to ${n}s`, 'success');
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
            showToast('Deleted', 'success');
        } catch (error) {
            showToast(error.message, 'error');
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
            showToast('Switched to db' + db, 'info');
            // Save state after database change
            if (shouldSave) this.saveState();
        } catch (error) {
            showToast(error.message, 'error');
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
            showToast('Database purged', 'success');
        } catch (error) {
            showToast(error.message, 'error');
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

        // Load data for specific tabs
        if (tab === 'cli') setTimeout(() => this.focusTerminal(), 50);
        if (tab === 'stats') this.loadServerInfo();
        if (tab === 'slowlog') this.loadSlowLog();
        if (tab === 'config') this.loadConfig();
        if (tab === 'acl') this.loadACLUsers();
        if (tab === 'cluster') this.loadClusterInfo();
        if (tab === 'pubsub') this.loadChannels();

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
                '<div class="stat-card"><div class="stat-value">' + escapeHtml(d.version || 'N/A') + '</div><div class="stat-label">Version</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + escapeHtml(d.mode || 'standalone') + '</div><div class="stat-label">Mode</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + (d.connected_clients || 0) + '</div><div class="stat-label">Clients</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + escapeHtml(d.used_memory || 'N/A') + '</div><div class="stat-label">Used Memory</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + escapeHtml(d.peak_memory || 'N/A') + '</div><div class="stat-label">Peak Memory</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + escapeHtml(d.max_memory || 'N/A') + '</div><div class="stat-label">Max Memory</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + (d.total_keys || 0) + '</div><div class="stat-label">Keys</div></div>' +
                '<div class="stat-card"><div class="stat-value">' + this.formatUptime(d.uptime) + '</div><div class="stat-label">Uptime</div></div>' +
                '</div><div class="stats-info"><h4>OS</h4><p>' + escapeHtml(d.os || 'N/A') + '</p></div></div>';
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
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

        if (!keyInput?.value.trim()) { showToast('Key name required', 'error'); return; }

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
            showToast('Created', 'success');
            keyInput.value = '';
            if (valueInput) valueInput.value = '';
            if (ttlInput) ttlInput.value = '-1';
        } catch (error) {
            showToast(error.message, 'error');
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

    exportData: null,

    showExportModal() {
        document.getElementById('exportModal')?.classList.add('active');
        document.getElementById('exportPreview').innerHTML = '<p style="color: var(--text-tertiary);">Click "Generate" to preview export data</p>';
        this.exportData = null;
    },

    showImportModal() {
        document.getElementById('importModal')?.classList.add('active');
    },

    async generateExport() {
        const pattern = document.getElementById('exportPattern')?.value || '*';
        const preview = document.getElementById('exportPreview');
        preview.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/export?pattern=' + encodeURIComponent(pattern));
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            this.exportData = json.data;
            preview.innerHTML = '<pre style="max-height:300px;overflow:auto;">' +
                escapeHtml(JSON.stringify(json.data, null, 2)) + '</pre>' +
                '<p style="margin-top:8px;color:var(--text-secondary);">' + (json.data.count || 0) + ' keys to export</p>';
        } catch (error) {
            preview.innerHTML = '<p style="color:var(--error);">Error: ' + escapeHtml(error.message) + '</p>';
        }
    },

    downloadExport() {
        if (!this.exportData) {
            showToast('Generate export first', 'error');
            return;
        }
        const blob = new Blob([JSON.stringify(this.exportData, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'redis-export-' + new Date().toISOString().slice(0,10) + '.json';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        showToast('Downloaded', 'success');
    },

    async importKeys() {
        const dataInput = document.getElementById('importData');
        const overwrite = document.getElementById('importOverwrite')?.checked || false;

        if (!dataInput?.value.trim()) {
            showToast('Please enter JSON data', 'error');
            return;
        }

        try {
            const data = JSON.parse(dataInput.value);
            const keys = data.keys || data;

            const response = await fetch('/api/import', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ keys: keys, overwrite: overwrite })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            showToast(`Imported: ${json.imported}, Skipped: ${json.skipped}`, 'success');
            this.closeModal('importModal');
            this.loadKeys();
            dataInput.value = '';
        } catch (error) {
            showToast('Import failed: ' + error.message, 'error');
        }
    },

    showBulkTTLModal() {
        document.getElementById('bulkTTLModal')?.classList.add('active');
    },

    async applyBulkTTL() {
        const pattern = document.getElementById('bulkTTLPattern')?.value;
        const ttl = parseInt(document.getElementById('bulkTTLValue')?.value || '0');

        if (!pattern) {
            showToast('Pattern is required', 'error');
            return;
        }

        try {
            const response = await fetch('/api/bulk-ttl', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ pattern: pattern, ttl: ttl })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            showToast(`Updated TTL for ${json.updated} keys`, 'success');
            this.closeModal('bulkTTLModal');
            this.loadKeys();
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async loadMemoryStats() {
        const pattern = document.getElementById('memoryPattern')?.value || '*';
        const container = document.getElementById('memoryContent');
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const [statsRes, overviewRes] = await Promise.all([
                fetch('/api/memory/stats?pattern=' + encodeURIComponent(pattern) + '&limit=100'),
                fetch('/api/memory/overview')
            ]);
            const stats = await statsRes.json();
            const overview = await overviewRes.json();

            if (!stats.success) throw new Error(stats.message);

            const data = stats.data || {};
            const keys = data.keys || [];
            const typeStats = data.type_stats || {};
            const mem = overview.data || {};

            let html = '<div class="memory-overview">';
            html += '<div class="stats-grid">';
            html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(mem.used_memory_human || 'N/A') + '</div><div class="stat-label">Used Memory</div></div>';
            html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(mem.used_memory_peak_human || 'N/A') + '</div><div class="stat-label">Peak Memory</div></div>';
            html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(mem.used_memory_rss_human || 'N/A') + '</div><div class="stat-label">RSS Memory</div></div>';
            html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(mem.mem_fragmentation_ratio || 'N/A') + '</div><div class="stat-label">Fragmentation</div></div>';
            html += '</div></div>';

            html += '<h4 style="margin:20px 0 10px;">Memory by Type</h4>';
            html += '<div class="type-stats">';
            for (const [type, bytes] of Object.entries(typeStats)) {
                html += '<div class="type-stat"><span class="key-type ' + type + '">' + type + '</span><span>' + this.formatBytes(bytes) + '</span></div>';
            }
            html += '</div>';

            html += '<h4 style="margin:20px 0 10px;">Top Keys by Memory (' + keys.length + ')</h4>';
            html += '<table class="data-table"><thead><tr><th>Key</th><th>Type</th><th>Memory</th><th>TTL</th></tr></thead><tbody>';
            for (const key of keys) {
                html += '<tr><td>' + escapeHtml(key.key) + '</td><td><span class="key-type ' + key.type + '">' + key.type + '</span></td>';
                html += '<td>' + this.formatBytes(key.memory_used) + '</td>';
                html += '<td>' + (key.ttl === -1 ? '∞' : key.ttl + 's') + '</td></tr>';
            }
            html += '</tbody></table>';

            container.innerHTML = html;
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
        }
    },

    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    },

    async loadSlowLog() {
        const container = document.getElementById('slowlogContent');
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/slowlog?count=50');
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const entries = json.data || [];
            if (entries.length === 0) {
                container.innerHTML = '<div class="empty-state"><h3>No slow queries</h3><p>Slow queries will appear here when detected</p></div>';
                return;
            }

            let html = '<table class="data-table"><thead><tr><th>ID</th><th>Time</th><th>Duration</th><th>Command</th><th>Client</th></tr></thead><tbody>';
            for (const entry of entries) {
                const duration = entry.duration_us >= 1000 ? (entry.duration_us / 1000).toFixed(2) + ' ms' : entry.duration_us + ' µs';
                const cmd = Array.isArray(entry.command) ? entry.command.join(' ') : entry.command;
                html += '<tr><td>' + entry.id + '</td>';
                html += '<td>' + escapeHtml(entry.formatted_time || '') + '</td>';
                html += '<td class="duration">' + duration + '</td>';
                html += '<td class="command-cell"><code>' + escapeHtml(cmd.substring(0, 100)) + (cmd.length > 100 ? '...' : '') + '</code></td>';
                html += '<td>' + escapeHtml(entry.client_addr || '-') + '</td></tr>';
            }
            html += '</tbody></table>';

            container.innerHTML = html;
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
        }
    },

    async resetSlowLog() {
        if (!confirm('Clear the slow log?')) return;
        try {
            const response = await fetch('/api/slowlog', { method: 'DELETE' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            showToast('Slow log cleared', 'success');
            this.loadSlowLog();
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async executeScript() {
        const script = document.getElementById('scriptEditor')?.value;
        const keysStr = document.getElementById('scriptKeys')?.value || '';
        const argsStr = document.getElementById('scriptArgs')?.value || '';
        const resultBox = document.getElementById('scriptResult');

        if (!script) {
            showToast('Enter a script', 'error');
            return;
        }

        const keys = keysStr.split(',').map(k => k.trim()).filter(k => k);
        const args = argsStr.split(',').map(a => a.trim()).filter(a => a);

        resultBox.textContent = 'Executing...';

        try {
            const response = await fetch('/api/script/eval', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ script: script, keys: keys, args: args })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const data = json.data || {};
            if (data.error) {
                resultBox.textContent = 'Error: ' + data.error;
                resultBox.style.color = 'var(--error)';
            } else {
                resultBox.textContent = JSON.stringify(data.result, null, 2) + '\n\nDuration: ' + data.duration;
                resultBox.style.color = 'var(--text-primary)';
            }
        } catch (error) {
            resultBox.textContent = 'Error: ' + error.message;
            resultBox.style.color = 'var(--error)';
        }
    },

    async loadScript() {
        const script = document.getElementById('scriptEditor')?.value;
        const resultBox = document.getElementById('scriptResult');

        if (!script) {
            showToast('Enter a script', 'error');
            return;
        }

        try {
            const response = await fetch('/api/script/load', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ script: script })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            resultBox.textContent = 'Script loaded successfully!\nSHA: ' + json.sha;
            resultBox.style.color = 'var(--success)';
            showToast('Script loaded', 'success');
        } catch (error) {
            resultBox.textContent = 'Error: ' + error.message;
            resultBox.style.color = 'var(--error)';
        }
    },

    async flushScripts() {
        if (!confirm('Flush all loaded scripts?')) return;
        try {
            const response = await fetch('/api/scripts', { method: 'DELETE' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            showToast('Scripts flushed', 'success');
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async publishMessage() {
        const channel = document.getElementById('pubChannel')?.value;
        const message = document.getElementById('pubMessage')?.value;

        if (!channel) {
            showToast('Channel is required', 'error');
            return;
        }

        try {
            const response = await fetch('/api/pubsub/publish', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ channel: channel, message: message || '' })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            showToast(`Published to ${json.receivers} subscribers`, 'success');
            document.getElementById('pubMessage').value = '';
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async loadChannels() {
        const container = document.getElementById('channelsList');
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/pubsub/channels');
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const data = json.data || {};
            const channels = data.channels || [];

            if (channels.length === 0) {
                container.innerHTML = '<div class="empty-state"><p>No active channels</p><small style="color:var(--text-tertiary);">Channels only appear when subscribers are connected</small></div>';
                return;
            }

            let html = '<div class="channels-grid">';
            for (const ch of channels) {
                html += '<div class="channel-item"><span class="channel-name">' + escapeHtml(ch.channel) + '</span>';
                html += '<span class="channel-subs">' + ch.subscribers + ' subs</span></div>';
            }
            html += '</div>';
            if (data.pattern_subscribers > 0) {
                html += '<p style="margin-top:10px;color:var(--text-secondary);">Pattern subscribers: ' + data.pattern_subscribers + '</p>';
            }

            container.innerHTML = html;
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
        }
    },

    async loadConfig() {
        const pattern = document.getElementById('configPattern')?.value || '*';
        const container = document.getElementById('configContent');
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/config?pattern=' + encodeURIComponent(pattern));
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const config = json.data || {};
            const entries = Object.entries(config);

            if (entries.length === 0) {
                container.innerHTML = '<div class="empty-state"><p>No configuration found</p></div>';
                return;
            }

            let html = '<table class="data-table config-table"><thead><tr><th>Parameter</th><th>Value</th><th></th></tr></thead><tbody>';
            for (const [key, value] of entries.sort((a, b) => a[0].localeCompare(b[0]))) {
                html += '<tr data-key="' + escapeHtml(key) + '">';
                html += '<td class="config-key">' + escapeHtml(key) + '</td>';
                html += '<td><input type="text" class="config-value-input" value="' + escapeHtml(value) + '" data-original="' + escapeHtml(value) + '"></td>';
                html += '<td><button class="btn btn-sm" onclick="RedisStudio.saveConfigValue(this)">Save</button></td>';
                html += '</tr>';
            }
            html += '</tbody></table>';

            container.innerHTML = html;
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
        }
    },

    async saveConfigValue(btn) {
        const row = btn.closest('tr');
        const key = row.dataset.key;
        const input = row.querySelector('.config-value-input');
        const value = input.value;

        try {
            const response = await fetch('/api/config', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ key: key, value: value })
            });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            input.dataset.original = value;
            showToast('Config updated', 'success');
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async rewriteConfig() {
        if (!confirm('Rewrite Redis configuration file?')) return;
        try {
            const response = await fetch('/api/config/rewrite', { method: 'POST' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            showToast('Config rewritten', 'success');
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async resetStats() {
        if (!confirm('Reset server statistics?')) return;
        try {
            const response = await fetch('/api/config/resetstat', { method: 'POST' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            showToast('Stats reset', 'success');
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async loadACLUsers() {
        const container = document.getElementById('aclContent');
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/acl/users');
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const users = json.data || [];

            if (users.length === 0) {
                container.innerHTML = '<div class="empty-state"><p>No ACL users (Redis 6+ required)</p></div>';
                return;
            }

            let html = '<table class="data-table"><thead><tr><th>User Rules</th></tr></thead><tbody>';
            for (const user of users) {
                html += '<tr><td><code>' + escapeHtml(user) + '</code></td></tr>';
            }
            html += '</tbody></table>';

            container.innerHTML = html;
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '<br><small>ACL requires Redis 6.0+</small></p></div>';
        }
    },

    showACLLogModal() {
        document.getElementById('aclLogModal')?.classList.add('active');
        this.loadACLLog();
    },

    async loadACLLog() {
        const container = document.getElementById('aclLogContent');
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const response = await fetch('/api/acl/log?count=20');
            const json = await response.json();
            if (!json.success) throw new Error(json.message);

            const logs = json.data || [];

            if (logs.length === 0) {
                container.innerHTML = '<div class="empty-state"><p>No ACL log entries</p></div>';
                return;
            }

            let html = '<table class="data-table"><thead><tr><th>Reason</th><th>Context</th><th>Object</th><th>Username</th><th>Age</th></tr></thead><tbody>';
            for (const log of logs) {
                html += '<tr>';
                html += '<td>' + escapeHtml(log.reason || '') + '</td>';
                html += '<td>' + escapeHtml(log.context || '') + '</td>';
                html += '<td>' + escapeHtml(log.object || '') + '</td>';
                html += '<td>' + escapeHtml(log.username || '') + '</td>';
                html += '<td>' + (log.age_seconds || 0) + 's</td>';
                html += '</tr>';
            }
            html += '</tbody></table>';

            container.innerHTML = html;
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
        }
    },

    async resetACLLog() {
        try {
            const response = await fetch('/api/acl/log', { method: 'DELETE' });
            const json = await response.json();
            if (!json.success) throw new Error(json.message);
            showToast('ACL log cleared', 'success');
            this.loadACLLog();
        } catch (error) {
            showToast('Failed: ' + error.message, 'error');
        }
    },

    async loadClusterInfo() {
        const container = document.getElementById('clusterContent');
        container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

        try {
            const [replRes, clusterRes] = await Promise.all([
                fetch('/api/replication'),
                fetch('/api/cluster')
            ]);
            const repl = await replRes.json();
            const cluster = await clusterRes.json();

            let html = '<div class="cluster-info">';

            // Replication Info
            html += '<h4>Replication</h4>';
            if (repl.success && repl.data) {
                const r = repl.data;
                html += '<div class="stats-grid">';
                html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(r.role || 'unknown') + '</div><div class="stat-label">Role</div></div>';
                html += '<div class="stat-card"><div class="stat-value">' + (r.connected_slaves || 0) + '</div><div class="stat-label">Connected Slaves</div></div>';
                if (r.master_host) {
                    html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(r.master_host + ':' + r.master_port) + '</div><div class="stat-label">Master</div></div>';
                    html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(r.master_link_status || 'unknown') + '</div><div class="stat-label">Link Status</div></div>';
                }
                html += '</div>';

                if (r.slaves && r.slaves.length > 0) {
                    html += '<h5 style="margin-top:15px;">Slaves</h5>';
                    html += '<table class="data-table"><thead><tr><th>Index</th><th>IP</th><th>Port</th><th>State</th><th>Offset</th></tr></thead><tbody>';
                    for (const slave of r.slaves) {
                        html += '<tr>';
                        html += '<td>' + escapeHtml(slave.index || '') + '</td>';
                        html += '<td>' + escapeHtml(slave.ip || '') + '</td>';
                        html += '<td>' + escapeHtml(slave.port || '') + '</td>';
                        html += '<td>' + escapeHtml(slave.state || '') + '</td>';
                        html += '<td>' + escapeHtml(slave.offset || '') + '</td>';
                        html += '</tr>';
                    }
                    html += '</tbody></table>';
                }
            } else {
                html += '<p style="color:var(--text-tertiary);">Replication info not available</p>';
            }

            // Cluster Info
            html += '<h4 style="margin-top:25px;">Cluster</h4>';
            if (cluster.success && cluster.data) {
                const c = cluster.data;
                if (!c.enabled) {
                    html += '<p style="color:var(--text-tertiary);">Cluster mode is not enabled</p>';
                } else {
                    html += '<div class="stats-grid">';
                    html += '<div class="stat-card"><div class="stat-value">' + escapeHtml(c.state || 'unknown') + '</div><div class="stat-label">State</div></div>';
                    html += '<div class="stat-card"><div class="stat-value">' + (c.known_nodes || 0) + '</div><div class="stat-label">Nodes</div></div>';
                    html += '<div class="stat-card"><div class="stat-value">' + (c.slots_ok || 0) + '</div><div class="stat-label">Slots OK</div></div>';
                    html += '<div class="stat-card"><div class="stat-value">' + (c.size || 0) + '</div><div class="stat-label">Size</div></div>';
                    html += '</div>';

                    if (c.nodes && c.nodes.length > 0) {
                        html += '<h5 style="margin-top:15px;">Nodes</h5>';
                        html += '<table class="data-table"><thead><tr><th>ID</th><th>Address</th><th>Flags</th><th>State</th><th>Slots</th></tr></thead><tbody>';
                        for (const node of c.nodes) {
                            html += '<tr>';
                            html += '<td title="' + escapeHtml(node.id || '') + '">' + escapeHtml((node.id || '').substring(0, 8)) + '...</td>';
                            html += '<td>' + escapeHtml(node.addr || '') + '</td>';
                            html += '<td>' + escapeHtml(node.flags || '') + '</td>';
                            html += '<td>' + escapeHtml(node.state || '') + '</td>';
                            html += '<td>' + escapeHtml(node.slots || '') + '</td>';
                            html += '</tr>';
                        }
                        html += '</tbody></table>';
                    }
                }
            } else {
                html += '<p style="color:var(--text-tertiary);">Cluster info not available</p>';
            }

            html += '</div>';
            container.innerHTML = html;
        } catch (error) {
            container.innerHTML = '<div class="empty-state"><p>Error: ' + escapeHtml(error.message) + '</p></div>';
        }
    }
};

document.addEventListener('DOMContentLoaded', () => RedisStudio.init());
