// Event Listeners and User Interaction Binding

import { state } from './state.js';
import { API } from './api.js';
import { showToast, renderAccounts, renderTagPills } from './ui.js';
import { startCamera, stopCamera, switchAddTab, parseFromImageURL, parseFromFile, populateFormFromURI } from './scanner.js';
import { backupVault } from './icloud.js';

export function setupEventListeners() {
  // Lock / Unlock / Init Form
  const lockForm = document.getElementById('lockForm');
  if (lockForm) {
    lockForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const pwd = document.getElementById('masterPasswordInput').value;
      if (!pwd) return;

      try {
        if (!state.initialized) {
          await API.initVault(pwd);
          showToast('保险库已成功初始化');
        } else {
          await API.unlockVault(pwd);
          showToast('保险库已解锁');
        }
        document.getElementById('masterPasswordInput').value = '';
        await refreshApp();
      } catch (err) {
        showToast(err.message, true);
      }
    });
  }

  // Lock button
  const lockBtn = document.getElementById('lockBtn');
  if (lockBtn) {
    lockBtn.addEventListener('click', async () => {
      try {
        await API.lockVault();
        showToast('保险库已安全锁定');
        await refreshApp();
      } catch (err) {
        showToast(err.message, true);
      }
    });
  }

  // Search input
  let searchTimer = null;
  const searchInput = document.getElementById('searchInput');
  if (searchInput) {
    searchInput.addEventListener('input', (e) => {
      clearTimeout(searchTimer);
      state.searchQuery = e.target.value;
      searchTimer = setTimeout(async () => {
        state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
        renderAccounts();
      }, 200);
    });
  }

  // Tag pills delegation
  const tagPills = document.getElementById('tagPills');
  if (tagPills) {
    tagPills.addEventListener('click', async (e) => {
      const pill = e.target.closest('.tag-pill');
      if (!pill) return;
      state.activeTag = pill.getAttribute('data-tag') || '';
      document.querySelectorAll('.tag-pill').forEach(p => p.classList.remove('active'));
      pill.classList.add('active');
      state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
      renderAccounts();
    });
  }

  // Accounts Grid delegation
  const accountsGrid = document.getElementById('accountsGrid');
  if (accountsGrid) {
    accountsGrid.addEventListener('click', async (e) => {
      const card = e.target.closest('.account-card');
      if (!card) return;
      const id = card.getAttribute('data-id');

      // Click on Code Box -> Copy
      const codeBox = e.target.closest('.code-box');
      const hotpBtn = e.target.closest('.hotp-btn');
      if (hotpBtn) {
        e.stopPropagation();
        try {
          const res = await API.incrementHOTP(id);
          showToast(`已生成新验证码: ${res.code}`);
          state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
          renderAccounts();
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
          await API.togglePin(id);
          state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
          renderAccounts();
        } catch (err) {
          showToast(err.message, true);
        }
        return;
      }

      // QR button
      if (e.target.closest('.qr-btn')) {
        openQRModal(id);
        return;
      }

      // Edit button
      if (e.target.closest('.edit-btn')) {
        openEditModal(id);
        return;
      }

      // Delete button
      if (e.target.closest('.del-btn')) {
        const acc = state.accounts.find(a => a.id === id);
        const name = acc ? `${acc.issuer} (${acc.account_name})` : '此令牌';
        if (confirm(`确定要删除 ${name} 吗？此操作无法撤销。`)) {
          try {
            await API.deleteAccount(id);
            showToast('令牌已删除');
            state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
            state.tags = await API.getTags();
            renderAccounts();
            renderTagPills();
          } catch (err) {
            showToast(err.message, true);
          }
        }
        return;
      }
    });
  }

  // Modal close buttons
  document.querySelectorAll('.close-modal').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.modal-backdrop').forEach(m => m.classList.remove('active'));
      stopCamera();
    });
  });

  // Add button
  const addBtn = document.getElementById('addBtn');
  if (addBtn) {
    addBtn.addEventListener('click', () => {
      state.editingId = null;
      document.getElementById('modalTitle').textContent = '添加 2FA 令牌';
      document.getElementById('accountForm').reset();
      document.getElementById('editAccountId').value = '';
      document.getElementById('addModal').classList.add('active');
      switchAddTab('tabScan');
      startCamera();
    });
  }

  // Add Modal Tabs
  const addTabs = document.getElementById('addTabs');
  if (addTabs) {
    addTabs.addEventListener('click', (e) => {
      const btn = e.target.closest('.tab-btn');
      if (!btn) return;
      const tab = btn.getAttribute('data-tab');
      switchAddTab(tab);
    });
  }

  // Image URL Parse Button (Requirement 1)
  const parseURLBtn = document.getElementById('parseURLBtn');
  if (parseURLBtn) {
    parseURLBtn.addEventListener('click', () => {
      const url = document.getElementById('qrURLInput').value.trim();
      parseFromImageURL(url);
    });
  }

  // Image File Input
  const qrFileInput = document.getElementById('qrFileInput');
  if (qrFileInput) {
    qrFileInput.addEventListener('change', (e) => {
      const file = e.target.files[0];
      if (file) parseFromFile(file);
    });
  }

  // Parse URI Button
  const parseURIBtn = document.getElementById('parseURIBtn');
  if (parseURIBtn) {
    parseURIBtn.addEventListener('click', () => {
      const uriStr = document.getElementById('otpURIInput').value.trim();
      if (!uriStr) {
        showToast('请先粘贴 otpauth:// 链接', true);
        return;
      }
      populateFormFromURI(uriStr);
      switchAddTab('tabManual');
    });
  }

  // Save Account Button
  const saveAccountBtn = document.getElementById('saveAccountBtn');
  if (saveAccountBtn) {
    saveAccountBtn.addEventListener('click', async () => {
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
          await API.updateAccount(id, payload);
          showToast('令牌更新成功');
        } else {
          await API.createAccount(payload);
          showToast('令牌添加成功');
        }
        document.getElementById('addModal').classList.remove('active');
        stopCamera();
        state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
        state.tags = await API.getTags();
        renderAccounts();
        renderTagPills();
      } catch (err) {
        showToast(err.message, true);
      }
    });
  }

  // Import / Export Modal
  const importExportBtn = document.getElementById('importExportBtn');
  if (importExportBtn) {
    importExportBtn.addEventListener('click', () => {
      document.getElementById('importExportModal').classList.add('active');
    });
  }

  // Export buttons
  const exportJSONBtn = document.getElementById('exportJSONBtn');
  if (exportJSONBtn) {
    exportJSONBtn.addEventListener('click', () => backupVault('json'));
  }

  const exportURIsBtn = document.getElementById('exportURIsBtn');
  if (exportURIsBtn) {
    exportURIsBtn.addEventListener('click', () => backupVault('uri'));
  }

  const exportICloudBtn = document.getElementById('exportICloudBtn');
  if (exportICloudBtn) {
    exportICloudBtn.addEventListener('click', () => backupVault('json'));
  }

  // Import button
  const doImportBtn = document.getElementById('doImportBtn');
  if (doImportBtn) {
    doImportBtn.addEventListener('click', async () => {
      const text = document.getElementById('importTextInput').value.trim();
      if (!text) {
        showToast('请先输入或选择要导入的内容', true);
        return;
      }
      try {
        const data = await API.importData(text);
        showToast(`成功导入 ${data.imported} 个令牌`);
        document.getElementById('importExportModal').classList.remove('active');
        document.getElementById('importTextInput').value = '';
        state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
        state.tags = await API.getTags();
        renderAccounts();
        renderTagPills();
      } catch (err) {
        showToast(err.message, true);
      }
    });
  }

  const importFileInput = document.getElementById('importFileInput');
  if (importFileInput) {
    importFileInput.addEventListener('change', (e) => {
      const file = e.target.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = (evt) => {
        document.getElementById('importTextInput').value = evt.target.result;
      };
      reader.readAsText(file);
    });
  }

  // Settings Modal
  const settingsBtn = document.getElementById('settingsBtn');
  if (settingsBtn) {
    settingsBtn.addEventListener('click', () => {
      document.getElementById('settingsModal').classList.add('active');
    });
  }

  const autoLockSelect = document.getElementById('autoLockSelect');
  if (autoLockSelect) {
    autoLockSelect.addEventListener('change', async (e) => {
      const mins = parseInt(e.target.value);
      try {
        await API.setAutoLock(mins);
        showToast('自动锁定时间已更新');
      } catch (err) {
        showToast(err.message, true);
      }
    });
  }

  const changePasswordForm = document.getElementById('changePasswordForm');
  if (changePasswordForm) {
    changePasswordForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const oldPass = document.getElementById('oldMasterPassword').value;
      const newPass = document.getElementById('newMasterPassword').value;
      try {
        await API.changePassword(oldPass, newPass);
        showToast('主密码修改成功');
        changePasswordForm.reset();
        document.getElementById('settingsModal').classList.remove('active');
      } catch (err) {
        showToast(err.message, true);
      }
    });
  }

  // Global Keyboard Shortcuts (Ctrl+F to search, Ctrl+L to lock, Esc to close modals)
  window.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
      e.preventDefault();
      const searchBox = document.getElementById('searchInput');
      if (searchBox) searchBox.focus();
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'l') {
      e.preventDefault();
      const lkBtn = document.getElementById('lockBtn');
      if (lkBtn) lkBtn.click();
    } else if (e.key === 'Escape') {
      document.querySelectorAll('.modal-backdrop').forEach(m => m.classList.remove('active'));
      stopCamera();
    }
  });
}

async function openQRModal(id) {
  try {
    const res = await API.getAccountURI(id);
    const uri = res.uri;
    const qrImg = document.getElementById('qrImage');
    qrImg.src = `/api/qr?size=300&data=${encodeURIComponent(uri)}`;
    document.getElementById('qrURIText').textContent = uri;

    const copyBtn = document.getElementById('copyURIBtn');
    if (copyBtn) {
      copyBtn.onclick = () => {
        navigator.clipboard.writeText(uri).then(() => {
          showToast('已复制 URI 链接');
        });
      };
    }

    document.getElementById('qrModal').classList.add('active');
  } catch (err) {
    showToast(err.message, true);
  }
}

function openEditModal(id) {
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
  document.getElementById('addModal').classList.add('active');
}

export async function refreshApp() {
  try {
    const status = await API.getStatus();
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
      state.accounts = await API.getAccounts(state.searchQuery, state.activeTag);
      state.tags = await API.getTags();
      renderAccounts();
      renderTagPills();
    }
  } catch (err) {
    showToast('连接服务器失败，请确保后台正在运行', true);
  }
}
