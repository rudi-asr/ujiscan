// ============================================================================
// ujiscan Frontend API Client + Auth Manager
// ============================================================================

// Detect environment
const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
const API_BASE = isLocalhost ? 'http://localhost:8081' : 'https://eds-barrel-entity-sponsor.trycloudflare.com';

// ============================================================================
// 1. AUTH MANAGEMENT
// ============================================================================

class AuthManager {
    constructor() {
        this.tokenKey = 'ujiscan_token';
        this.userKey = 'ujiscan_user';
    }

    // Get stored token
    getToken() {
        return localStorage.getItem(this.tokenKey);
    }

    // Set token after login
    setToken(token) {
        localStorage.setItem(this.tokenKey, token);
    }

    // Get current user
    getUser() {
        const user = localStorage.getItem(this.userKey);
        return user ? JSON.parse(user) : null;
    }

    // Set current user
    setUser(user) {
        localStorage.setItem(this.userKey, JSON.stringify(user));
    }

    // Logout
    logout() {
        localStorage.removeItem(this.tokenKey);
        localStorage.removeItem(this.userKey);
        window.location.href = '/html/login.html';
    }

    // Check if logged in
    isLoggedIn() {
        return !!this.getToken();
    }

    // Login
    async login(email, password) {
        try {
            const res = await fetch(`${API_BASE}/auth/login`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });

            if (!res.ok) {
                throw new Error('Login failed');
            }

            const data = await res.json();
            this.setToken(data.token);
            this.setUser({ email, role: data.role });
            return true;
        } catch (err) {
            console.error('Login error:', err);
            return false;
        }
    }
}

const auth = new AuthManager();

// ============================================================================
// 2. API CLIENT
// ============================================================================

class APIClient {
    constructor(baseURL) {
        this.baseURL = baseURL;
    }

    // Helper: Make authenticated request
    async request(endpoint, options = {}) {
        const token = auth.getToken();
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const res = await fetch(`${this.baseURL}${endpoint}`, {
            ...options,
            headers
        });

        if (res.status === 401) {
            auth.logout();
        }

        return res;
    }

    // Auth endpoints
    async getMe() {
        const res = await this.request('/auth/me');
        return res.json();
    }

    // Engagement endpoints
    async listEngagements() {
        const res = await this.request('/api/engagements');
        return res.json();
    }

    async createEngagement(data) {
        const res = await this.request('/api/engagements', {
            method: 'POST',
            body: JSON.stringify(data)
        });
        return res.json();
    }

    async getEngagement(id) {
        const res = await this.request(`/api/engagements/${id}`);
        return res.json();
    }

    // Audit endpoints
    async getAuditLogs() {
        const res = await this.request('/api/audit/logs');
        return res.json();
    }

    // Dashboard endpoints
    async getDashboardTeam() {
        const res = await this.request('/api/dashboard/team');
        return res.json();
    }

    async getDashboardClient() {
        const res = await this.request('/api/dashboard/client');
        return res.json();
    }

    async getDashboardAdmin() {
        const res = await this.request('/api/dashboard/admin');
        return res.json();
    }

    // Notifications
    async getNotifications() {
        const res = await this.request('/api/notifications');
        return res.json();
    }

    // Scan endpoints (legacy)
    async listTools() {
        const res = await this.request('/api/tools');
        return res.json();
    }

    async startScan(target) {
        const res = await this.request('/api/scan', {
            method: 'POST',
            body: JSON.stringify({ target })
        });
        return res.json();
    }

    async getScan(id) {
        const res = await this.request(`/api/scan/${id}`);
        return res.json();
    }

    async getStatus() {
        const res = await this.request('/api/status');
        return res.json();
    }
}

const api = new APIClient(API_BASE);

// ============================================================================
// 3. UTILITY FUNCTIONS
// ============================================================================

// Format date
function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
}

// Format time ago
function formatTimeAgo(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now - date) / 1000);
    
    if (seconds < 60) return 'just now';
    if (seconds < 3600) return Math.floor(seconds / 60) + 'm ago';
    if (seconds < 86400) return Math.floor(seconds / 3600) + 'h ago';
    return Math.floor(seconds / 86400) + 'd ago';
}

// Show notification
function showNotification(message, type = 'info') {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.classList.add('show');
    }, 10);
    
    setTimeout(() => {
        notification.classList.remove('show');
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// ============================================================================
// 4. THEME MANAGEMENT
// ============================================================================

class ThemeManager {
    constructor() {
        this.themeKey = 'ujiscan_theme';
        this.darkTheme = 'dark';
        this.lightTheme = 'light';
    }

    init() {
        const saved = localStorage.getItem(this.themeKey);
        const theme = saved || this.darkTheme;
        this.setTheme(theme);
    }

    setTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem(this.themeKey, theme);
    }

    toggle() {
        const current = document.documentElement.getAttribute('data-theme');
        const next = current === this.darkTheme ? this.lightTheme : this.darkTheme;
        this.setTheme(next);
    }

    current() {
        return document.documentElement.getAttribute('data-theme');
    }
}

const theme = new ThemeManager();

// ============================================================================
// 5. PAGE INITIALIZATION
// ============================================================================

document.addEventListener('DOMContentLoaded', () => {
    theme.init();

    // Check if user is logged in
    if (!auth.isLoggedIn() && !window.location.pathname.includes('login.html')) {
        window.location.href = '/html/login.html';
        return;
    }

    // Initialize page-specific code if function exists
    if (typeof initPage === 'function') {
        initPage();
    }
});

// ============================================================================
// EXPORTS FOR GLOBAL USE
// ============================================================================

window.api = api;
window.auth = auth;
window.theme = theme;
window.formatDate = formatDate;
window.formatTimeAgo = formatTimeAgo;
window.showNotification = showNotification;
