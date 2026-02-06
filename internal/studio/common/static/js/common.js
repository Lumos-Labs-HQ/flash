// Shared utility functions for all FlashORM Studios

// DOM helpers
function $(selector) { return document.querySelector(selector); }
function $$(selector) { return document.querySelectorAll(selector); }

// Escape HTML to prevent XSS
function escapeHtml(text) {
    if (text == null) return '';
    const div = document.createElement('div');
    div.textContent = String(text);
    return div.innerHTML;
}

// Escape for use in HTML attributes
function escapeHtmlAttr(text) {
    if (text == null) return '';
    return String(text).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// Show a toast notification
function showToast(message, type, duration) {
    type = type || 'info';
    duration = duration || 3000;

    // Remove existing toasts
    var existing = document.querySelectorAll('.studio-toast');
    existing.forEach(function(el) { el.remove(); });

    var toast = document.createElement('div');
    toast.className = 'studio-toast studio-toast-' + type;

    var icons = { success: '\u2713', error: '\u2717', warning: '\u26A0', info: '\u2139' };
    toast.innerHTML = '<span class="studio-toast-icon">' + (icons[type] || icons.info) + '</span>' +
        '<span class="studio-toast-msg">' + escapeHtml(message) + '</span>';

    document.body.appendChild(toast);

    requestAnimationFrame(function() {
        toast.classList.add('show');
    });

    setTimeout(function() {
        toast.classList.remove('show');
        setTimeout(function() { toast.remove(); }, 300);
    }, duration);
}

// Fetch wrapper with JSON parsing and error handling
async function apiCall(url, options) {
    options = options || {};
    try {
        var resp = await fetch(url, options);
        var json = await resp.json();
        if (!resp.ok) {
            throw new Error((json && json.message) || 'Request failed');
        }
        return json;
    } catch (err) {
        throw err;
    }
}

// Session storage helpers
var sessionState = {
    save: function(key, value) {
        try { sessionStorage.setItem(key, JSON.stringify(value)); } catch(e) {}
    },
    get: function(key, defaultValue) {
        try {
            var v = sessionStorage.getItem(key);
            return v !== null ? JSON.parse(v) : (defaultValue !== undefined ? defaultValue : null);
        } catch(e) { return defaultValue !== undefined ? defaultValue : null; }
    },
    remove: function(key) {
        try { sessionStorage.removeItem(key); } catch(e) {}
    }
};
