// 2FA Vault - Modern Frontend Application Logic

(function () {
  'use strict';

  // State
  let state = {
    initialized: false,
    locked: true,
    autoLockMinutes: 15,
    accounts: [],
    tags: [],
    activeTag: '',
    searchQuery: '',
    editingId: null,
    scanStream: null,
    scanInterval: null
  };

  // Color generator based on issuer name
  const AVATAR_COLORS = [
    ['#4f46e5', '#7c3aed'],
    ['#2563eb', '#38bdf8'],
    ['#059669', '#34d399'],
    ['#d97706', '#fbbf24'],
    ['#dc2626', '#f87171'],
    ['#7c3aed', '#ec4899'],
    ['#0891b2', '#06b6d4'],
    ['#475569', '#64748b']
  ];

  function getAvatarGradient(name) {
    if (!name) return 'linear-gradient(135deg, #4f46e5, #7c3aed)';
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    const idx = Math.abs(hash) % AVATAR_COLORS.length;
    const [c1, c2] = AVATAR_COLORS[idx];
    return `linear-gradient(135deg, ${c1}, ${c2})`;
  }

  // Toast notifications
  function showToast(message, isError = false) {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast ${isError ? 'error' : ''}`;
    toast.innerHTML = `
      <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
        ${isError 
          ? '<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>'
          : '<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>'}
      </svg>
      <span>${escapeHTML(message)}</span>
    `;
    container.appendChild(toast);

    if (navigator.vibrate) {
      navigator.vibrate(isError ? [50, 50, 50] : [40]);
    }

    setTimeout(() => {
      toast.style.transition = 'opacity 0.25s, transform 0.25s';
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(10px) scale(0.95)';
      setTimeout(() => toast.remove(), 250);
    }, 2800);
  }

  function escapeHTML(str) {
    if (!str) return '';
    return str.replace(/[&<>'"]/g, 
      tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag));
  }

  // API Client helpers
  async function api(path, options = {}) {
    try {
      const res = await fetch(path, {
        headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
        ...options
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data.error || `HTTP ${res.status}`);
      }
      return data;
    } catch (err) {
      console.error(`API Error on ${path}:`, err);
      throw err;
    }
  }

  // Initialize App
  async function init() {
    setupEventListeners();
    setupTheme();
    setupPWA();
    await checkStatus();
    startCountdownLoop();
  }

  async function checkStatus() {
    try {
      const status = await api('/api/vault/status');
      state.initialized = status.initialized;
      state.locked = status.locked;
      state.autoLockMinutes = status.auto_lock_minutes || 15;

      const lockScreen = document.getElementById('lockScreen');
      const lockTitle = document.getElementById('lockTitle');
      const lockSubtitle = document.getElementById('lockSubtitle');
      const unlockBtn = document.getElementById('unlockBtn');
      const autoLockSelect = document.getElementById('autoLockSelect');
      if (autoLockSelect) autoLockSelect.value = state.autoLockMinutes;

      if (!state.initialized) {
        lockScreen.style.display = 'flex';
        lockTitle.textContent = '欢迎使用 2FA Vault';
        lockSubtitle.textContent = '首次启动，请设置主密码以初始化您的加密保险库';
        unlockBtn.textContent = '创建主密码并初始化';
      } else if (state.locked) {
        lockScreen.style.display = 'flex';
        lockTitle.textContent = '保险库已锁定';
        lockSubtitle.textContent = '请输入主密码以解密 2FA 令牌';
        unlockBtn.textContent = '解锁保险库';
      } else {
        lockScreen.style.display = 'none';
        await loadAccounts();
        await loadTags();
      }
    } catch (err) {
      showToast('连接服务器失败，请确保后台正在运行', true);
    }
  }

  async function loadAccounts() {
    try {
      const q = encodeURIComponent(state.searchQuery || '');
      const tag = encodeURIComponent(state.activeTag || '');
      const accounts = await api(`/api/accounts?q=${q}&tag=${tag}`);
      state.accounts = accounts || [];
      renderAccounts();
    } catch (err) {
      if (err.message.includes('locked')) {
        state.locked = true;
        checkStatus();
      }
    }
  }

  async function loadTags() {
    try {
      const tags = await api('/api/tags');
      state.tags = tags || [];
      renderTagPills();
    } catch (err) {
      console.warn('Load tags error:', err);
    }
  }

  function renderTagPills() {
    const container = document.getElementById('tagPills');
    let html = `<span class="tag-pill ${state.activeTag === '' ? 'active' : ''}" data-tag="">全部</span>`;
    for (const t of state.tags) {
      html += `<span class="tag-pill ${state.activeTag === t ? 'active' : ''}" data-tag="${escapeHTML(t)}">${escapeHTML(t)}</span>`;
    }
    container.innerHTML = html;
  }

  function renderAccounts() {
    const grid = document.getElementById('accountsGrid');
    if (!state.accounts || state.accounts.length === 0) {
      grid.innerHTML = `
        <div class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24"><path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/></svg>
          <h3>暂无 2FA 验证令牌</h3>
          <p style="margin-top: 6px; font-size: 0.9rem;">点击上方“添加令牌”开始扫描二维码或手动录入</p>
        </div>
      `;
      return;
    }

    const now = Math.floor(Date.now() / 1000);

    grid.innerHTML = state.accounts.map(acc => {
      const initial = (acc.issuer || acc.account_name || '?').charAt(0).toUpperCase();
      const gradient = getAvatarGradient(acc.issuer || acc.account_name);
      const isSteam = acc.type === 'steam';
      const isHOTP = acc.type === 'hotp';

      const period = acc.period || 30;
      const remaining = isHOTP ? 0 : (period - (now % period));
      const progress = isHOTP ? 0 : (now % period) / period;

      // Format code with space (e.g. 123 456)
      const rawCode = acc.code || '------';
      let formattedCode = rawCode;
      if (rawCode.length === 6) {
        formattedCode = `${rawCode.slice(0, 3)} ${rawCode.slice(3)}`;
      } else if (rawCode.length === 8) {
        formattedCode = `${rawCode.slice(0, 4)} ${rawCode.slice(4)}`;
      }

      // Ring circumference = 2 * PI * 16 ≈ 100.53
      const dashoffset = 100.53 * progress;
      let strokeColor = 'var(--success)';
      if (remaining <= 5) strokeColor = 'var(--danger)';
      else if (remaining <= 10) strokeColor = 'var(--warning)';

      return `
        <div class="account-card" data-id="${acc.id}">
          <div class="card-header">
            <div class="card-meta">
              <div class="issuer-avatar" style="background: ${gradient};">${initial}</div>
              <div class="issuer-info">
                <div class="issuer-title">${escapeHTML(acc.issuer || '未知服务')}</div>
                <div class="account-label">${escapeHTML(acc.account_name || '无用户名')}</div>
              </div>
            </div>
            <div class="card-actions">
              <button class="btn-icon-sm pin-btn ${acc.pinned ? 'pinned' : ''}" title="${acc.pinned ? '取消置顶' : '置顶'}">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/></svg>
              </button>
              <button class="btn-icon-sm qr-btn" title="查看二维码">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M3 3h8v8H3zm2 2v4h4V5zm8-2h8v8h-8zm2 2v4h4V5zM3 13h8v8H3zm2 2v4h4v-4zm13-2h3v2h-3zm-3 2h2v3h-2zm3 3h3v3h-3zm-3 2h2v2h-2z"/></svg>
              </button>
              <button class="btn-icon-sm edit-btn" title="编辑">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/></svg>
              </button>
              <button class="btn-icon-sm del-btn" title="删除">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/></svg>
              </button>
            </div>
          </div>

          <div class="code-box" data-code="${rawCode}" title="点击复制验证码">
            <div class="otp-code">${formattedCode}</div>
            ${isHOTP ? `
              <button class="btn btn-ghost btn-icon hotp-btn" title="生成下一个验证码 (当前计数: ${acc.counter})">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"/></svg>
              </button>
            ` : `
              <div class="timer-container">
                <svg class="timer-svg" viewBox="0 0 38 38">
                  <circle class="timer-bg" cx="19" cy="19" r="16"></circle>
                  <circle class="timer-circle" cx="19" cy="19" r="16" 
                          style="stroke-dashoffset: ${dashoffset}; stroke: ${strokeColor};"></circle>
                </svg>
                <span class="timer-text">${remaining}s</span>
              </div>
            `}
          </div>

          <div class="card-footer">
            <div style="display: flex; gap: 6px; flex-wrap: wrap;">
              ${isSteam ? '<span class="badge badge-steam">Steam</span>' : ''}
              ${(acc.tags || []).map(t => `<span class="badge">${escapeHTML(t)}</span>`).join('')}
            </div>
            <span>${acc.algorithm || 'SHA1'} · ${acc.digits || 6}位</span>
          </div>
        </div>
      `;
    }).join('');
  }

  // Real-time Countdown Animation Loop
  function startCountdownLoop() {
    let lastSecond = -1;

    setInterval(() => {
      if (state.locked || !state.accounts || state.accounts.length === 0) return;

      const now = Math.floor(Date.now() / 1000);
      const currentSecond = now % 30;

      // Update circular meters smoothly
      document.querySelectorAll('.account-card').forEach(card => {
        const id = card.getAttribute('data-id');
        const acc = state.accounts.find(a => a.id === id);
        if (!acc || acc.type === 'hotp') return;

        const period = acc.period || 30;
        const remaining = period - (now % period);
        const progress = (now % period) / period;
        const dashoffset = 100.53 * progress;

        const circle = card.querySelector('.timer-circle');
        const text = card.querySelector('.timer-text');

        if (circle) {
          circle.style.strokeDashoffset = dashoffset;
          if (remaining <= 5) {
            circle.style.stroke = 'var(--danger)';
          } else if (remaining <= 10) {
            circle.style.stroke = 'var(--warning)';
          } else {
            circle.style.stroke = 'var(--success)';
          }
        }
        if (text) {
          text.textContent = `${remaining}s`;
        }
      });

      // When hitting period boundary, refresh account codes from server
      if (currentSecond === 0 && lastSecond !== 0) {
        loadAccounts();
      }
      lastSecond = currentSecond;
    }, 1000);
  }

  // Setup Event Listeners
  function setupEventListeners() {
    // Unlock / Init Form
    document.getElementById('lockForm').addEventListener('submit', async (e) => {
      e.preventDefault();
      const pwd = document.getElementById('masterPasswordInput').value;
      if (!pwd) return;

      try {
        if (!state.initialized) {
          await api('/api/vault/init', { method: 'POST', body: JSON.stringify({ password: pwd }) });
          showToast('保险库已成功初始化');
        } else {
          await api('/api/vault/unlock', { method: 'POST', body: JSON.stringify({ password: pwd }) });
          showToast('保险库已解锁');
        }
        document.getElementById('masterPasswordInput').value = '';
        await checkStatus();
      } catch (err) {
        showToast(err.message, true);
      }
    });

    // Lock button
    document.getElementById('lockBtn').addEventListener('click', async () => {
      try {
        await api('/api/vault/lock', { method: 'POST' });
        showToast('保险库已安全锁定');
        await checkStatus();
      } catch (err) {
        showToast(err.message, true);
      }
    });

    // Search input
    let searchTimer = null;
    document.getElementById('searchInput').addEventListener('input', (e) => {
      clearTimeout(searchTimer);
      state.searchQuery = e.target.value;
      searchTimer = setTimeout(loadAccounts, 200);
    });

    // Tag pills delegation
    document.getElementById('tagPills').addEventListener('click', (e) => {
      const pill = e.target.closest('.tag-pill');
      if (!pill) return;
      state.activeTag = pill.getAttribute('data-tag') || '';
      document.querySelectorAll('.tag-pill').forEach(p => p.classList.remove('active'));
      pill.classList.add('active');
      loadAccounts();
    });

    // Accounts Grid delegation (Copy, Pin, QR, Edit, Delete, HOTP)
    document.getElementById('accountsGrid').addEventListener('click', async (e) => {
      const card = e.target.closest('.account-card');
      if (!card) return;
      const id = card.getAttribute('data-id');

      // Click on Code Box -> Copy
      const codeBox = e.target.closest('.code-box');
      const hotpBtn = e.target.closest('.hotp-btn');
      if (hotpBtn) {
        e.stopPropagation();
        try {
          const res = await api(`/api/accounts/${id}/hotp`, { method: 'POST' });
          showToast(`已生成新验证码: ${res.code}`);
          await loadAccounts();
        } catch (err) {
          showToast(err.message, true);
        }
        return;
      }

      if (codeBox) {
        const code = codeBox.getAttribute('data-code');
        if (code && code !== '------' && code !== 'ERR') {
          navigator.clipboard.writeText(code).then(() => {
            showToast(`已复制验证码: ${code}`);
          }).catch(() => {
            showToast('复制失败，请手动长按复制', true);
          });
        }
        return;
      }

      // Pin button
      if (e.target.closest('.pin-btn')) {
        try {
          await api(`/api/accounts/${id}/pin`, { method: 'POST' });
          await loadAccounts();
        } catch (err) {
          showToast(err.message, true);
        }
        return;
      }

      // QR button
      if (e.target.closest('.qr-btn')) {
        showQRModal(id);
        return;
      }

      // Edit button
      if (e.target.closest('.edit-btn')) {
        showEditModal(id);
        return;
      }

      // Delete button
      if (e.target.closest('.del-btn')) {
        const acc = state.accounts.find(a => a.id === id);
        const name = acc ? `${acc.issuer} (${acc.account_name})` : '此令牌';
        if (confirm(`确定要删除 ${name} 吗？此操作无法撤销。`)) {
          try {
            await api(`/api/accounts/${id}`, { method: 'DELETE' });
            showToast('令牌已删除');
            await loadAccounts();
            await loadTags();
          } catch (err) {
            showToast(err.message, true);
          }
        }
        return;
      }
    });

    // Modal close buttons
    document.querySelectorAll('.close-modal').forEach(btn => {
      btn.addEventListener('click', () => {
        document.querySelectorAll('.modal-backdrop').forEach(m => m.classList.remove('active'));
        stopCamera();
      });
    });

    // Add button
    document.getElementById('addBtn').addEventListener('click', () => {
      state.editingId = null;
      document.getElementById('modalTitle').textContent = '添加 2FA 令牌';
      document.getElementById('accountForm').reset();
      document.getElementById('editAccountId').value = '';
      document.getElementById('addModal').classList.add('active');
      switchAddTab('tabScan');
      startCamera();
    });

    // Tabs inside Add Modal
    document.getElementById('addTabs').addEventListener('click', (e) => {
      const btn = e.target.closest('.tab-btn');
      if (!btn) return;
      const tab = btn.getAttribute('data-tab');
      switchAddTab(tab);
      if (tab === 'tabScan') {
        startCamera();
      } else {
        stopCamera();
      }
    });

    // Parse URI Button
    document.getElementById('parseURIBtn').addEventListener('click', () => {
      const uriStr = document.getElementById('otpURIInput').value.trim();
      if (!uriStr) {
        showToast('请先粘贴 otpauth:// 链接', true);
        return;
      }
      populateFormFromURI(uriStr);
      switchAddTab('tabManual');
      stopCamera();
    });

    // Save Account Button
    document.getElementById('saveAccountBtn').addEventListener('click', async () => {
      const id = document.getElementById('editAccountId').value;
      const issuer = document.getElementById('accIssuer').value.trim();
      const accountName = document.getElementById('accAccount').value.trim();
      const secret = document.getElementById('accSecret').value.trim();
      const type = document.getElementById('accType').value;
      const digits = parseInt(document.getElementById('accDigits').value) || 6;
      const period = parseInt(document.getElementById('accPeriod').value) || 30;
      const algo = document.getElementById('accAlgo').value;
      const tagsStr = document.getElementById('accTags').value.trim();
      const notes = document.getElementById('accNotes').value.trim();

      if (!secret) {
        showToast('密钥不能为空', true);
        return;
      }

      const tags = tagsStr ? tagsStr.split(/[,，]/).map(t => t.trim()).filter(Boolean) : [];

      const payload = {
        issuer,
        account_name: accountName,
        secret,
        type,
        digits,
        period,
        algorithm: algo,
        tags,
        notes
      };

      try {
        if (id) {
          payload.id = id;
          await api(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(payload) });
          showToast('令牌更新成功');
        } else {
          await api('/api/accounts', { method: 'POST', body: JSON.stringify(payload) });
          showToast('令牌添加成功');
        }
        document.getElementById('addModal').classList.remove('active');
        stopCamera();
        await loadAccounts();
        await loadTags();
      } catch (err) {
        showToast(err.message, true);
      }
    });

    // QR Image File Input
    document.getElementById('qrFileInput').addEventListener('change', async (e) => {
      const file = e.target.files[0];
      if (!file) return;

      try {
        const bmp = await createImageBitmap(file);
        if ('BarcodeDetector' in window) {
          const detector = new BarcodeDetector({ formats: ['qr_code'] });
          const barcodes = await detector.detect(bmp);
          if (barcodes.length > 0) {
            populateFormFromURI(barcodes[0].rawValue);
            switchAddTab('tabManual');
            stopCamera();
            showToast('二维码解析成功');
            return;
          }
        }
        showToast('未能识别图片中的二维码，请尝试手动输入密钥', true);
      } catch (err) {
        showToast('读取二维码图片失败: ' + err.message, true);
      }
    });

    // Import / Export Modal
    document.getElementById('importExportBtn').addEventListener('click', () => {
      document.getElementById('importExportModal').classList.add('active');
    });

    document.getElementById('exportJSONBtn').addEventListener('click', () => {
      window.open('/api/export?format=json', '_blank');
    });

    document.getElementById('exportURIsBtn').addEventListener('click', () => {
      window.open('/api/export?format=uri', '_blank');
    });

    document.getElementById('doImportBtn').addEventListener('click', async () => {
      const text = document.getElementById('importTextInput').value.trim();
      if (!text) {
        showToast('请先输入或选择要导入的内容', true);
        return;
      }
      try {
        const res = await fetch('/api/import', {
          method: 'POST',
          headers: { 'Content-Type': 'text/plain; charset=utf-8' },
          body: text
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || '导入失败');
        showToast(`成功导入 ${data.imported} 个令牌`);
        document.getElementById('importExportModal').classList.remove('active');
        document.getElementById('importTextInput').value = '';
        await loadAccounts();
        await loadTags();
      } catch (err) {
        showToast(err.message, true);
      }
    });

    document.getElementById('importFileInput').addEventListener('change', (e) => {
      const file = e.target.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (evt) => {
        document.getElementById('importTextInput').value = evt.target.result;
      };
      reader.readAsText(file);
    });

    // Settings Modal
    document.getElementById('settingsBtn').addEventListener('click', () => {
      document.getElementById('settingsModal').classList.add('active');
    });

    document.getElementById('autoLockSelect').addEventListener('change', async (e) => {
      const mins = parseInt(e.target.value);
      try {
        await api('/api/vault/autolock', { method: 'POST', body: JSON.stringify({ minutes: mins }) });
        showToast('自动锁定时间已更新');
      } catch (err) {
        showToast(err.message, true);
      }
    });

    document.getElementById('changePasswordForm').addEventListener('submit', async (e) => {
      e.preventDefault();
      const oldPass = document.getElementById('oldMasterPassword').value;
      const newPass = document.getElementById('newMasterPassword').value;
      try {
        await api('/api/vault/password', {
          method: 'POST',
          body: JSON.stringify({ old_password: oldPass, new_password: newPass })
        });
        showToast('主密码修改成功');
        document.getElementById('changePasswordForm').reset();
        document.getElementById('settingsModal').classList.remove('active');
      } catch (err) {
        showToast(err.message, true);
      }
    });

    // Keyboard shortcuts: Ctrl+F to search, Ctrl+L to lock, Esc to close modals
    window.addEventListener('keydown', (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
        e.preventDefault();
        document.getElementById('searchInput').focus();
      } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'l') {
        e.preventDefault();
        document.getElementById('lockBtn').click();
      } else if (e.key === 'Escape') {
        document.querySelectorAll('.modal-backdrop').forEach(m => m.classList.remove('active'));
        stopCamera();
      }
    });
  }

  // Camera Scanner implementation
  async function startCamera() {
    stopCamera();
    const video = document.getElementById('scannerVideo');
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      console.warn('Camera API not supported');
      return;
    }

    try {
      state.scanStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment' }
      });
      video.srcObject = state.scanStream;
      await video.play();

      if ('BarcodeDetector' in window) {
        const detector = new BarcodeDetector({ formats: ['qr_code'] });
        state.scanInterval = setInterval(async () => {
          if (video.readyState < 2) return;
          try {
            const barcodes = await detector.detect(video);
            if (barcodes.length > 0) {
              const val = barcodes[0].rawValue;
              stopCamera();
              populateFormFromURI(val);
              switchAddTab('tabManual');
              showToast('二维码扫描成功！');
            }
          } catch (e) {
            // frame detect pass
          }
        }, 300);
      }
    } catch (err) {
      console.warn('Camera access error:', err);
    }
  }

  function stopCamera() {
    if (state.scanInterval) {
      clearInterval(state.scanInterval);
      state.scanInterval = null;
    }
    if (state.scanStream) {
      state.scanStream.getTracks().forEach(t => t.stop());
      state.scanStream = null;
    }
  }

  function switchAddTab(tabId) {
    document.querySelectorAll('#addTabs .tab-btn').forEach(btn => {
      btn.classList.toggle('active', btn.getAttribute('data-tab') === tabId);
    });
    document.querySelectorAll('.tab-content').forEach(c => {
      c.style.display = c.id === tabId ? 'block' : 'none';
    });
  }

  function populateFormFromURI(uriStr) {
    try {
      const u = new URL(uriStr);
      if (u.protocol !== 'otpauth:') return;

      const type = u.host.toLowerCase();
      let label = decodeURIComponent(u.pathname.replace(/^\//, ''));
      let issuer = '';
      let account = label;

      if (label.includes(':')) {
        const parts = label.split(':');
        issuer = parts[0].trim();
        account = parts.slice(1).join(':').trim();
      }

      const q = u.searchParams;
      if (q.get('issuer')) issuer = q.get('issuer');
      const secret = q.get('secret') || '';
      const algo = (q.get('algorithm') || 'SHA1').toUpperCase();
      const digits = q.get('digits') || '6';
      const period = q.get('period') || '30';

      document.getElementById('accIssuer').value = issuer;
      document.getElementById('accAccount').value = account;
      document.getElementById('accSecret').value = secret;
      document.getElementById('accType').value = type === 'hotp' ? 'hotp' : 'totp';
      document.getElementById('accDigits').value = digits;
      document.getElementById('accPeriod').value = period;
      document.getElementById('accAlgo').value = ['SHA1', 'SHA256', 'SHA512'].includes(algo) ? algo : 'SHA1';
    } catch (e) {
      console.warn('Parse URI error:', e);
    }
  }

  async function showQRModal(id) {
    try {
      const res = await api(`/api/accounts/${id}/uri`);
      const uri = res.uri;
      const qrImg = document.getElementById('qrImage');
      qrImg.src = `/api/qr?size=300&data=${encodeURIComponent(uri)}`;
      document.getElementById('qrURIText').textContent = uri;

      document.getElementById('copyURIBtn').onclick = () => {
        navigator.clipboard.writeText(uri).then(() => {
          showToast('已复制 URI 链接');
        });
      };

      document.getElementById('qrModal').classList.add('active');
    } catch (err) {
      showToast(err.message, true);
    }
  }

  async function showEditModal(id) {
    const acc = state.accounts.find(a => a.id === id);
    if (!acc) return;

    state.editingId = id;
    document.getElementById('modalTitle').textContent = '编辑 2FA 令牌';
    document.getElementById('editAccountId').value = acc.id;
    document.getElementById('accIssuer').value = acc.issuer || '';
    document.getElementById('accAccount').value = acc.account_name || '';
    document.getElementById('accSecret').value = acc.secret || '';
    document.getElementById('accType').value = acc.type || 'totp';
    document.getElementById('accDigits').value = acc.digits || 6;
    document.getElementById('accPeriod').value = acc.period || 30;
    document.getElementById('accAlgo').value = acc.algorithm || 'SHA1';
    document.getElementById('accTags').value = (acc.tags || []).join(', ');
    document.getElementById('accNotes').value = acc.notes || '';

    switchAddTab('tabManual');
    stopCamera();
    document.getElementById('addModal').classList.add('active');
  }

  // Theme management
  function setupTheme() {
    const savedTheme = localStorage.getItem('2fa_theme') || 'dark';
    applyTheme(savedTheme);

    document.getElementById('themeToggleBtn').addEventListener('click', () => {
      const nextTheme = document.documentElement.classList.contains('dark') ? 'light' : 'dark';
      applyTheme(nextTheme);
    });
  }

  function applyTheme(theme) {
    if (theme === 'light') {
      document.documentElement.classList.remove('dark');
      document.documentElement.classList.add('light');
      document.getElementById('themeToggleText').textContent = '☀️ 切换为暗色模式';
    } else {
      document.documentElement.classList.remove('light');
      document.documentElement.classList.add('dark');
      document.getElementById('themeToggleText').textContent = '🌙 切换为亮色模式';
    }
    localStorage.setItem('2fa_theme', theme);
  }

  // PWA Service Worker Registration
  function setupPWA() {
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/service-worker.js').catch(() => {});
    }
  }

  // Boot
  document.addEventListener('DOMContentLoaded', init);
})();
