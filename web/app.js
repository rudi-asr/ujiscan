// Initialize
document.addEventListener('DOMContentLoaded', () => {
    checkAPIStatus();
    setupEventListeners();
});

// Check API status
async function checkAPIStatus() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        document.getElementById('apiStatus').textContent = `API: ${data.status}`;
    } catch (err) {
        document.getElementById('apiStatus').textContent = 'API: offline';
    }
}

// Setup form submission
function setupEventListeners() {
    document.getElementById('scanForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const target = document.getElementById('target').value;
        if (!target.trim()) return;

        addLog(`Starting scan for target: ${target}`, 'info');
        document.getElementById('status').textContent = 'Running...';
        
        try {
            const res = await fetch('/api/scan', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ target })
            });
            
            if (res.ok) {
                const data = await res.json();
                addLog(`Scan ID: ${data.id}`, 'success');
                document.getElementById('status').textContent = 'In progress...';
            } else {
                addLog(`Error: ${res.statusText}`, 'error');
            }
        } catch (err) {
            addLog(`Error: ${err.message}`, 'error');
        }
    });
}

// Log utility
function addLog(message, level = 'info') {
    const logsContainer = document.getElementById('logs');
    const entry = document.createElement('div');
    entry.className = `log-entry ${level}`;
    entry.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
    logsContainer.appendChild(entry);
    logsContainer.scrollTop = logsContainer.scrollHeight;
}
