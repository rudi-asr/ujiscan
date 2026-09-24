// Global state
let currentScanID = null;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    checkAPIStatus();
    loadTools();
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
        addLog('API connection failed', 'error');
    }
}

// Load available tools
async function loadTools() {
    try {
        const res = await fetch('/api/tools');
        const tools = await res.json();
        
        const toolsList = document.getElementById('toolsList');
        if (toolsList) {
            toolsList.innerHTML = '';
            tools.forEach(tool => {
                const badge = document.createElement('span');
                badge.className = `tool-badge ${tool.available ? 'available' : 'unavailable'}`;
                badge.textContent = tool.name;
                toolsList.appendChild(badge);
            });
        }

        addLog(`${tools.length} tools loaded`, 'info');
    } catch (err) {
        addLog(`Failed to load tools: ${err.message}`, 'error');
    }
}

// Setup form submission
function setupEventListeners() {
    document.getElementById('scanForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const target = document.getElementById('target').value;
        if (!target.trim()) return;

        const button = e.target.querySelector('button');
        button.disabled = true;
        button.textContent = 'Starting...';

        addLog(`Starting scan for target: ${target}`, 'info');
        document.getElementById('status').innerHTML = '<span class="status-running">Running...</span>';
        
        try {
            const res = await fetch('/api/scan', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ target })
            });
            
            if (res.ok) {
                const data = await res.json();
                currentScanID = data.id;
                addLog(`Scan started: ${data.id}`, 'success');
                document.getElementById('target').value = '';
                
                // Poll for results
                pollScanResults(data.id);
            } else {
                const error = await res.text();
                addLog(`Error: ${error}`, 'error');
                document.getElementById('status').textContent = 'Failed';
            }
        } catch (err) {
            addLog(`Error: ${err.message}`, 'error');
            document.getElementById('status').textContent = 'Error';
        } finally {
            button.disabled = false;
            button.textContent = 'Start Scan';
        }
    });
}

// Poll for scan results
async function pollScanResults(scanID) {
    const maxAttempts = 60; // 60 seconds max (1 sec per attempt)
    let attempt = 0;

    const poll = async () => {
        try {
            const res = await fetch(`/api/scan/${scanID}`);
            const scan = await res.json();

            document.getElementById('status').innerHTML = 
                `<span class="status-${scan.status}">${scan.status.toUpperCase()}</span>`;

            // Display results
            if (scan.results && scan.results.length > 0) {
                displayResults(scan);
            }

            // If completed or failed, stop polling
            if (scan.status === 'completed' || scan.status === 'failed') {
                addLog(`Scan ${scan.status}`, scan.status === 'failed' ? 'error' : 'success');
                
                if (scan.error) {
                    addLog(`Error: ${scan.error}`, 'error');
                }
                
                return;
            }

            // Continue polling
            if (attempt < maxAttempts) {
                attempt++;
                setTimeout(poll, 1000); // Poll every 1 second
            } else {
                // Max timeout reached
                addLog(`Scan timeout after ${maxAttempts} seconds`, 'error');
                document.getElementById('status').innerHTML = 
                    `<span class="status-timeout">TIMEOUT</span>`;
            }
        } catch (err) {
            addLog(`Poll error: ${err.message}`, 'error');
            if (attempt < maxAttempts) {
                attempt++;
                setTimeout(poll, 2000);
            }
        }
    };

    poll();
}

// Display scan results
function displayResults(scan) {
    const resultsDiv = document.getElementById('results');
    if (!resultsDiv) return;

    let html = `<div class="results-header">
        <h3>Results (${scan.results.length} tools)</h3>
        <div class="result-summary">
            Target: <strong>${scan.target}</strong> | 
            Started: <strong>${new Date(scan.started_at).toLocaleString()}</strong>
        </div>
    </div>`;

    scan.results.forEach((result, idx) => {
        const statusClass = result.success ? 'success' : 'error';
        html += `
        <details class="result-item result-${statusClass}">
            <summary>
                <span class="tool-name">${result.tool_name}</span>
                <span class="phase-badge">${result.phase}</span>
                <span class="status-badge">${result.success ? 'OK' : 'FAILED'}</span>
                <span class="duration">${result.duration_ms}ms</span>
            </summary>
            <div class="result-details">
                <div class="command">
                    <strong>Command:</strong> ${result.command} ${result.args.join(' ')}
                </div>
                <div class="exit-code">
                    <strong>Exit Code:</strong> ${result.exit_code}
                </div>
                ${result.stdout ? `<div class="output">
                    <strong>Output:</strong>
                    <pre>${escapeHtml(result.stdout.substring(0, 500))}${result.stdout.length > 500 ? '...' : ''}</pre>
                </div>` : ''}
                ${result.error ? `<div class="error">
                    <strong>Error:</strong> ${result.error}
                </div>` : ''}
            </div>
        </details>`;
    });

    resultsDiv.innerHTML = html;
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

// HTML escape utility
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
