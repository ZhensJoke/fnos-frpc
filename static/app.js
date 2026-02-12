// === State ===
let authToken = '';
let servers = [];
let selectedServerId = null;
let editingServerId = null;
let editingProxyId = null;
let frpcInstalled = false;

// === API Helper ===
async function api(method, path, body = null) {
    const opts = {
        method,
        headers: { 'Content-Type': 'application/json' },
    };
    if (authToken) {
        opts.headers['X-Auth-Token'] = authToken;
    }
    if (body) {
        opts.body = JSON.stringify(body);
    }
    const res = await fetch(`/api${path}`, opts);
    const data = await res.json();
    if (!res.ok) {
        throw new Error(data.error || `HTTP ${res.status}`);
    }
    return data;
}

// === Toast ===
function toast(msg, type = 'info') {
    const container = document.getElementById('toast-container');
    const el = document.createElement('div');
    el.className = `toast toast-${type}`;
    el.textContent = msg;
    container.appendChild(el);
    setTimeout(() => el.remove(), 3000);
}

// === Page Navigation ===
function showPage(id) {
    document.querySelectorAll('.page').forEach(p => p.classList.add('hidden'));
    document.getElementById(id).classList.remove('hidden');
}

// === Modal ===
function openModal(id) {
    document.getElementById(id).classList.remove('hidden');
}

function closeModal(id) {
    document.getElementById(id).classList.add('hidden');
}

// === Init ===
async function init() {
    try {
        const status = await api('GET', '/auth/status');
        if (status.needSetup) {
            showPage('setup-page');
            return;
        }

        // Try restoring saved session
        const savedToken = localStorage.getItem('authToken');
        if (savedToken) {
            authToken = savedToken;
            try {
                await api('GET', '/servers');
                enterApp();
                return;
            } catch (e) {
                // Token expired or invalid
                authToken = '';
                localStorage.removeItem('authToken');
            }
        }

        showPage('login-page');
    } catch (e) {
        toast('无法连接到服务器', 'error');
    }
}

// === Auth: Setup ===
document.getElementById('setup-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const pw = document.getElementById('setup-password').value;
    const confirm = document.getElementById('setup-confirm').value;
    const errEl = document.getElementById('setup-error');

    if (pw !== confirm) {
        errEl.textContent = '两次输入的密码不一致';
        errEl.classList.remove('hidden');
        return;
    }
    if (pw.length < 6) {
        errEl.textContent = '密码至少需要6位';
        errEl.classList.remove('hidden');
        return;
    }

    try {
        const res = await api('POST', '/auth/setup', { password: pw });
        authToken = res.token;
        localStorage.setItem('authToken', authToken);
        toast('密码设置成功', 'success');
        enterApp();
    } catch (e) {
        errEl.textContent = e.message;
        errEl.classList.remove('hidden');
    }
});

// === Auth: Login ===
document.getElementById('login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const pw = document.getElementById('login-password').value;
    const errEl = document.getElementById('login-error');

    try {
        const res = await api('POST', '/auth/login', { password: pw });
        authToken = res.token;
        localStorage.setItem('authToken', authToken);
        enterApp();
    } catch (e) {
        errEl.textContent = '密码错误';
        errEl.classList.remove('hidden');
    }
});

// === Enter App ===
async function enterApp() {
    showPage('main-page');
    await loadServers();
    await checkFrpcVersion();

    // Auto-open version modal if frpc not installed
    if (!frpcInstalled) {
        document.getElementById('btn-frpc-manage').click();
    }
}

// === Logout ===
document.getElementById('btn-logout').addEventListener('click', () => {
    authToken = '';
    localStorage.removeItem('authToken');
    selectedServerId = null;
    showPage('login-page');
    toast('\u5df2\u9000\u51fa\u767b\u5f55', 'info');
});

// === Load Servers ===
async function loadServers() {
    try {
        servers = await api('GET', '/servers');
        renderServerList();
    } catch (e) {
        toast('加载服务器列表失败: ' + e.message, 'error');
    }
}

function renderServerList() {
    const list = document.getElementById('server-list');

    if (servers.length === 0) {
        list.innerHTML = '<div class="empty-state"><p>暂无服务器</p><p class="text-muted">点击 + 添加第一个</p></div>';
        return;
    }

    list.innerHTML = servers.map(s => `
        <div class="server-item ${s.id === selectedServerId ? 'active' : ''}" data-id="${s.id}" onclick="selectServer('${s.id}')">
            <span class="status-indicator ${s.running ? 'running' : 'stopped'}"></span>
            <div>
                <div class="server-name">${escapeHtml(s.name)}</div>
                <div class="server-addr">${escapeHtml(s.serverAddr)}:${s.serverPort}</div>
            </div>
        </div>
    `).join('');
}

// === Select Server ===
function selectServer(id) {
    selectedServerId = id;
    renderServerList();
    renderServerDetail();
}

function renderServerDetail() {
    const server = servers.find(s => s.id === selectedServerId);
    if (!server) {
        document.getElementById('no-selection').classList.remove('hidden');
        document.getElementById('server-detail').classList.add('hidden');
        return;
    }

    document.getElementById('no-selection').classList.add('hidden');
    document.getElementById('server-detail').classList.remove('hidden');

    // Header
    document.getElementById('server-detail-name').textContent = server.name;

    // Running status
    const dot = document.getElementById('server-running-dot');
    const toggleText = document.getElementById('btn-toggle-text');
    const toggleBtn = document.getElementById('btn-toggle-server');

    if (server.running) {
        dot.className = 'status-indicator running';
        toggleText.textContent = '停止';
        toggleBtn.classList.add('btn-danger');
    } else {
        dot.className = 'status-indicator stopped';
        toggleText.textContent = '启动';
        toggleBtn.classList.remove('btn-danger');
    }

    // Config grid
    const grid = document.getElementById('server-config-grid');
    grid.innerHTML = `
        <div class="config-item">
            <div class="label">服务器地址</div>
            <div class="value">${escapeHtml(server.serverAddr)}</div>
        </div>
        <div class="config-item">
            <div class="label">端口</div>
            <div class="value">${server.serverPort}</div>
        </div>
        <div class="config-item">
            <div class="label">Token</div>
            <div class="value">${server.authToken ? '••••••••' : '未设置'}</div>
        </div>
        <div class="config-item">
            <div class="label">TLS</div>
            <div class="value">${server.tlsEnable ? '已启用' : '未启用'}</div>
        </div>
        ${server.user ? `<div class="config-item"><div class="label">用户名</div><div class="value">${escapeHtml(server.user)}</div></div>` : ''}
    `;

    // Proxies
    renderProxyTable(server.proxies || []);

    // Logs
    refreshLogs();
}

// === Proxy Table ===
function renderProxyTable(proxies) {
    const tbody = document.getElementById('proxy-table-body');
    const emptyEl = document.getElementById('proxy-empty');
    const tableEl = document.getElementById('proxy-table');

    if (!proxies || proxies.length === 0) {
        tableEl.classList.add('hidden');
        emptyEl.classList.remove('hidden');
        return;
    }

    tableEl.classList.remove('hidden');
    emptyEl.classList.add('hidden');

    tbody.innerHTML = proxies.map(p => {
        let remote = '';
        if (p.type === 'tcp' || p.type === 'udp') {
            remote = p.remotePort ? `:${p.remotePort}` : '-';
        } else {
            const parts = [];
            if (p.customDomains && p.customDomains.length) parts.push(p.customDomains.join(', '));
            if (p.subdomain) parts.push(`sub: ${p.subdomain}`);
            remote = parts.join('; ') || '-';
        }

        return `
            <tr>
                <td>${escapeHtml(p.name)}</td>
                <td><span class="type-badge type-${p.type}">${p.type}</span></td>
                <td>${escapeHtml(p.localIP || '127.0.0.1')}:${p.localPort}</td>
                <td>${escapeHtml(remote)}</td>
                <td>
                    <button class="btn btn-sm btn-ghost" onclick="editProxy('${p.id}')" title="编辑">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.12 2.12 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                    </button>
                    <button class="btn btn-sm btn-ghost btn-danger" onclick="deleteProxy('${p.id}')" title="删除">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                    </button>
                </td>
            </tr>
        `;
    }).join('');
}

// === Server CRUD ===
document.getElementById('btn-add-server').addEventListener('click', () => {
    editingServerId = null;
    document.getElementById('modal-server-title').textContent = '添加服务器';
    document.getElementById('server-form').reset();
    openModal('modal-server');
});

document.getElementById('btn-edit-server').addEventListener('click', () => {
    const server = servers.find(s => s.id === selectedServerId);
    if (!server) return;

    editingServerId = server.id;
    document.getElementById('modal-server-title').textContent = '编辑服务器';
    document.getElementById('sf-name').value = server.name;
    document.getElementById('sf-addr').value = server.serverAddr;
    document.getElementById('sf-port').value = server.serverPort;
    document.getElementById('sf-token').value = server.authToken || '';
    document.getElementById('sf-user').value = server.user || '';
    document.getElementById('sf-tls').checked = server.tlsEnable || false;
    openModal('modal-server');
});

document.getElementById('server-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = {
        name: document.getElementById('sf-name').value,
        serverAddr: document.getElementById('sf-addr').value,
        serverPort: parseInt(document.getElementById('sf-port').value),
        authToken: document.getElementById('sf-token').value,
        user: document.getElementById('sf-user').value,
        tlsEnable: document.getElementById('sf-tls').checked,
    };

    try {
        if (editingServerId) {
            await api('PUT', `/servers/${editingServerId}`, data);
            toast('服务器已更新', 'success');
        } else {
            await api('POST', '/servers', data);
            toast('服务器已添加', 'success');
        }
        closeModal('modal-server');
        await loadServers();
        if (editingServerId) selectServer(editingServerId);
    } catch (e) {
        toast(e.message, 'error');
    }
});

document.getElementById('btn-delete-server').addEventListener('click', async () => {
    if (!selectedServerId) return;
    const server = servers.find(s => s.id === selectedServerId);
    if (!confirm(`确定删除服务器 "${server.name}" 吗？`)) return;

    try {
        await api('DELETE', `/servers/${selectedServerId}`);
        toast('服务器已删除', 'success');
        selectedServerId = null;
        await loadServers();
        document.getElementById('no-selection').classList.remove('hidden');
        document.getElementById('server-detail').classList.add('hidden');
    } catch (e) {
        toast(e.message, 'error');
    }
});

// === Start/Stop Server ===
document.getElementById('btn-toggle-server').addEventListener('click', async () => {
    const server = servers.find(s => s.id === selectedServerId);
    if (!server) return;

    try {
        if (server.running) {
            await api('POST', `/servers/${selectedServerId}/stop`);
            toast('已停止', 'success');
        } else {
            await api('POST', `/servers/${selectedServerId}/start`);
            toast('已启动', 'success');
        }
        await loadServers();
        renderServerDetail();
    } catch (e) {
        toast(e.message, 'error');
    }
});

// === Proxy CRUD ===
document.getElementById('btn-add-proxy').addEventListener('click', () => {
    editingProxyId = null;
    document.getElementById('modal-proxy-title').textContent = '添加代理规则';
    document.getElementById('proxy-form').reset();
    document.getElementById('pf-local-ip').value = '127.0.0.1';
    document.getElementById('pf-type').value = 'tcp';
    toggleProxyFields();
    openModal('modal-proxy');
});

document.getElementById('pf-type').addEventListener('change', toggleProxyFields);

function toggleProxyFields() {
    const type = document.getElementById('pf-type').value;
    const tcpFields = document.getElementById('proxy-tcp-fields');
    const httpFields = document.getElementById('proxy-http-fields');

    if (type === 'http' || type === 'https') {
        tcpFields.classList.add('hidden');
        httpFields.classList.remove('hidden');
    } else {
        tcpFields.classList.remove('hidden');
        httpFields.classList.add('hidden');
    }
}

function editProxy(proxyId) {
    const server = servers.find(s => s.id === selectedServerId);
    if (!server) return;
    const proxy = server.proxies.find(p => p.id === proxyId);
    if (!proxy) return;

    editingProxyId = proxyId;
    document.getElementById('modal-proxy-title').textContent = '编辑代理规则';
    document.getElementById('pf-name').value = proxy.name;
    document.getElementById('pf-type').value = proxy.type;
    document.getElementById('pf-local-ip').value = proxy.localIP || '127.0.0.1';
    document.getElementById('pf-local-port').value = proxy.localPort;
    document.getElementById('pf-remote-port').value = proxy.remotePort || '';
    document.getElementById('pf-domains').value = (proxy.customDomains || []).join(', ');
    document.getElementById('pf-subdomain').value = proxy.subdomain || '';
    toggleProxyFields();
    openModal('modal-proxy');
}

async function deleteProxy(proxyId) {
    if (!confirm('确定删除此代理规则吗？')) return;
    try {
        await api('DELETE', `/servers/${selectedServerId}/proxies/${proxyId}`);
        toast('代理规则已删除', 'success');
        await loadServers();
        renderServerDetail();
    } catch (e) {
        toast(e.message, 'error');
    }
}

document.getElementById('proxy-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const type = document.getElementById('pf-type').value;
    const domainsRaw = document.getElementById('pf-domains').value.trim();

    const data = {
        name: document.getElementById('pf-name').value,
        type: type,
        localIP: document.getElementById('pf-local-ip').value || '127.0.0.1',
        localPort: parseInt(document.getElementById('pf-local-port').value),
    };

    if (type === 'tcp' || type === 'udp') {
        const rp = document.getElementById('pf-remote-port').value;
        if (rp) data.remotePort = parseInt(rp);
    } else {
        if (domainsRaw) data.customDomains = domainsRaw.split(',').map(d => d.trim()).filter(Boolean);
        const sub = document.getElementById('pf-subdomain').value.trim();
        if (sub) data.subdomain = sub;
    }

    try {
        if (editingProxyId) {
            await api('PUT', `/servers/${selectedServerId}/proxies/${editingProxyId}`, data);
            toast('代理规则已更新', 'success');
        } else {
            await api('POST', `/servers/${selectedServerId}/proxies`, data);
            toast('代理规则已添加', 'success');
        }
        closeModal('modal-proxy');
        await loadServers();
        renderServerDetail();
    } catch (e) {
        toast(e.message, 'error');
    }
});

// === Logs ===
document.getElementById('btn-refresh-logs').addEventListener('click', refreshLogs);

async function refreshLogs() {
    if (!selectedServerId) return;
    try {
        const data = await api('GET', `/servers/${selectedServerId}/logs`);
        const viewer = document.getElementById('log-viewer');
        viewer.textContent = data.logs || '暂无日志';
        viewer.scrollTop = viewer.scrollHeight;
    } catch (e) {
        // ignore
    }
}

// === FRPC Version ===
async function checkFrpcVersion() {
    try {
        const data = await api('GET', '/frpc/version');
        const badge = document.getElementById('frpc-version-badge');
        const text = document.getElementById('frpc-version-text');

        if (data.installed) {
            text.textContent = `frpc ${data.version}`;
            badge.querySelector('.dot').className = 'dot dot-green';
            frpcInstalled = true;
        } else {
            text.textContent = 'frpc 未安装';
            badge.querySelector('.dot').className = 'dot dot-red';
            frpcInstalled = false;
        }
    } catch (e) {
        document.getElementById('frpc-version-text').textContent = '检测失败';
    }
}

// === Close Version Modal with Reminder ===
function closeVersionModal() {
    closeModal('modal-version');

    // If frpc is still not installed, show reminder tooltip
    if (!frpcInstalled) {
        const tooltip = document.getElementById('frpc-tooltip');
        const manageBtn = document.getElementById('btn-frpc-manage');
        tooltip.classList.remove('hidden');
        manageBtn.classList.add('pulse-glow');

        // Auto-dismiss after 6 seconds
        setTimeout(() => {
            tooltip.classList.add('hidden');
            manageBtn.classList.remove('pulse-glow');
        }, 6000);
    }
}

document.getElementById('btn-frpc-manage').addEventListener('click', async () => {
    openModal('modal-version');

    // Current version
    try {
        const data = await api('GET', '/frpc/version');
        document.getElementById('ver-current').textContent = data.installed ? data.version : '未安装';
    } catch (e) {
        document.getElementById('ver-current').textContent = '获取失败';
    }

    // Latest version
    checkLatestVersion();
});

document.getElementById('btn-check-latest').addEventListener('click', checkLatestVersion);

async function checkLatestVersion() {
    document.getElementById('ver-latest').textContent = '获取中...';
    try {
        const data = await api('GET', '/frpc/latest');
        document.getElementById('ver-latest').textContent = data.version;
    } catch (e) {
        document.getElementById('ver-latest').textContent = '获取失败';
    }
}

// Online install
document.getElementById('btn-install-online').addEventListener('click', async () => {
    const progressEl = document.getElementById('version-progress');
    const statusEl = document.getElementById('version-status');
    const btn = document.getElementById('btn-install-online');

    btn.disabled = true;
    progressEl.classList.remove('hidden');
    statusEl.textContent = '正在下载并安装 frpc...';

    try {
        const data = await api('POST', '/frpc/install');
        statusEl.textContent = `安装成功: ${data.version}`;
        document.getElementById('ver-current').textContent = data.version;
        toast(`frpc ${data.version} 安装成功`, 'success');
        checkFrpcVersion();
    } catch (e) {
        statusEl.textContent = `安装失败: ${e.message}`;
        toast('安装失败: ' + e.message, 'error');
    } finally {
        btn.disabled = false;
        setTimeout(() => progressEl.classList.add('hidden'), 3000);
    }
});

// Offline upload
const uploadArea = document.getElementById('upload-area');
const uploadFile = document.getElementById('upload-file');

uploadArea.addEventListener('click', () => uploadFile.click());

uploadArea.addEventListener('dragover', (e) => {
    e.preventDefault();
    uploadArea.style.borderColor = 'var(--accent)';
});

uploadArea.addEventListener('dragleave', () => {
    uploadArea.style.borderColor = '';
});

uploadArea.addEventListener('drop', (e) => {
    e.preventDefault();
    uploadArea.style.borderColor = '';
    if (e.dataTransfer.files.length) {
        handleUpload(e.dataTransfer.files[0]);
    }
});

uploadFile.addEventListener('change', () => {
    if (uploadFile.files.length) {
        handleUpload(uploadFile.files[0]);
    }
});

async function handleUpload(file) {
    const progressEl = document.getElementById('version-progress');
    const statusEl = document.getElementById('version-status');

    progressEl.classList.remove('hidden');
    statusEl.textContent = `正在上传 ${file.name}...`;

    const formData = new FormData();
    formData.append('file', file);

    try {
        const res = await fetch('/api/frpc/upload', {
            method: 'POST',
            headers: authToken ? { 'X-Auth-Token': authToken } : {},
            body: formData,
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error);

        statusEl.textContent = `安装成功: ${data.version}`;
        document.getElementById('ver-current').textContent = data.version;
        toast(`frpc ${data.version} 安装成功`, 'success');
        checkFrpcVersion();
    } catch (e) {
        statusEl.textContent = `安装失败: ${e.message}`;
        toast('上传安装失败: ' + e.message, 'error');
    } finally {
        setTimeout(() => progressEl.classList.add('hidden'), 3000);
    }
}

// === Utilities ===
function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// === Auto refresh ===
setInterval(async () => {
    if (authToken && document.getElementById('main-page').classList.contains('hidden') === false) {
        await loadServers();
        if (selectedServerId) {
            renderServerDetail();
        }
    }
}, 10000);

// === Theme Management ===
function getAutoTheme() {
    // 1. Check system preference
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
        return 'light';
    }
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        return 'dark';
    }
    // 2. Fallback: time-based (6:00-18:00 = light)
    const hour = new Date().getHours();
    return (hour >= 6 && hour < 18) ? 'light' : 'dark';
}

function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    updateThemeIcon(theme);
}

function updateThemeIcon(theme) {
    const icon = document.getElementById('theme-icon');
    if (!icon) return;
    if (theme === 'light') {
        // Sun icon
        icon.innerHTML = '<circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>';
    } else {
        // Moon icon
        icon.innerHTML = '<path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/>';
    }
}

// Initialize theme immediately
(function () {
    const saved = localStorage.getItem('theme');
    const theme = saved || getAutoTheme();
    applyTheme(theme);
})();

// Theme toggle button
document.getElementById('btn-theme-toggle').addEventListener('click', () => {
    const current = document.documentElement.getAttribute('data-theme') || 'dark';
    const next = current === 'dark' ? 'light' : 'dark';
    localStorage.setItem('theme', next);
    applyTheme(next);
});

// Listen for system theme changes
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    if (!localStorage.getItem('theme')) {
        applyTheme(e.matches ? 'dark' : 'light');
    }
});

// === Start ===
init();

