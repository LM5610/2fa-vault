// UI Helpers and DOM Rendering

import { state } from './state.js';

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

export function getAvatarGradient(name) {
  if (!name) return 'linear-gradient(135deg, #4f46e5, #7c3aed)';
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  const idx = Math.abs(hash) % AVATAR_COLORS.length;
  const [c1, c2] = AVATAR_COLORS[idx];
  return `linear-gradient(135deg, ${c1}, ${c2})`;
}

export function escapeHTML(str) {
  if (!str) return '';
  return String(str).replace(/[&<>'"]/g, 
    tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag));
}

export function showToast(message, isError = false) {
  const container = document.getElementById('toastContainer');
  if (!container) return;
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

export function setupTheme() {
  const savedTheme = localStorage.getItem('2fa_theme') || 'dark';
  applyTheme(savedTheme);

  const btn = document.getElementById('themeToggleBtn');
  if (btn) {
    btn.addEventListener('click', () => {
      const nextTheme = document.documentElement.classList.contains('dark') ? 'light' : 'dark';
      applyTheme(nextTheme);
    });
  }
}

export function applyTheme(theme) {
  const toggleText = document.getElementById('themeToggleText');
  if (theme === 'light') {
    document.documentElement.classList.remove('dark');
    document.documentElement.classList.add('light');
    if (toggleText) toggleText.textContent = '☀️ 切换为暗色模式';
  } else {
    document.documentElement.classList.remove('light');
    document.documentElement.classList.add('dark');
    if (toggleText) toggleText.textContent = '🌙 切换为亮色模式';
  }
  localStorage.setItem('2fa_theme', theme);
}

export function renderTagPills() {
  const container = document.getElementById('tagPills');
  if (!container) return;
  let html = `<span class="tag-pill ${state.activeTag === '' ? 'active' : ''}" data-tag="">全部</span>`;
  for (const t of state.tags) {
    html += `<span class="tag-pill ${state.activeTag === t ? 'active' : ''}" data-tag="${escapeHTML(t)}">${escapeHTML(t)}</span>`;
  }
  container.innerHTML = html;
}

export function renderAccounts() {
  const grid = document.getElementById('accountsGrid');
  if (!grid) return;

  if (!state.accounts || state.accounts.length === 0) {
    grid.innerHTML = `
      <div class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24"><path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/></svg>
        <h3>暂无 2FA 验证令牌</h3>
        <p style="margin-top: 6px; font-size: 0.9rem;">点击上方“添加令牌”开始扫描二维码、粘贴图片链接或手动录入</p>
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

    const rawCode = acc.code || '------';
    let formattedCode = rawCode;
    if (rawCode.length === 6) {
      formattedCode = `${rawCode.slice(0, 3)} ${rawCode.slice(3)}`;
    } else if (rawCode.length === 8) {
      formattedCode = `${rawCode.slice(0, 4)} ${rawCode.slice(4)}`;
    }

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
