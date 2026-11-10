// API Communication Module

export async function request(path, options = {}) {
  try {
    const res = await fetch(path, {
      headers: {
        'Content-Type': 'application/json',
        ...(options.headers || {})
      },
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

export const API = {
  getStatus: () => request('/api/vault/status'),
  initVault: (password) => request('/api/vault/init', { method: 'POST', body: JSON.stringify({ password }) }),
  unlockVault: (password) => request('/api/vault/unlock', { method: 'POST', body: JSON.stringify({ password }) }),
  lockVault: () => request('/api/vault/lock', { method: 'POST' }),
  changePassword: (old_password, new_password) => request('/api/vault/password', {
    method: 'POST',
    body: JSON.stringify({ old_password, new_password })
  }),
  setAutoLock: (minutes) => request('/api/vault/autolock', {
    method: 'POST',
    body: JSON.stringify({ minutes })
  }),
  getAccounts: (q = '', tag = '') => request(`/api/accounts?q=${encodeURIComponent(q)}&tag=${encodeURIComponent(tag)}`),
  createAccount: (data) => request('/api/accounts', { method: 'POST', body: JSON.stringify(data) }),
  updateAccount: (id, data) => request(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteAccount: (id) => request(`/api/accounts/${id}`, { method: 'DELETE' }),
  togglePin: (id) => request(`/api/accounts/${id}/pin`, { method: 'POST' }),
  incrementHOTP: (id) => request(`/api/accounts/${id}/hotp`, { method: 'POST' }),
  getAccountURI: (id) => request(`/api/accounts/${id}/uri`),
  getTags: () => request('/api/tags'),
  decodeImageURL: (url) => request('/api/qr/decode-url', { method: 'POST', body: JSON.stringify({ url }) }),
  previewOTP: (params) => {
    const sp = new URLSearchParams(params);
    return request(`/api/otp/preview?${sp.toString()}`);
  },
  decodeImageFile: async (file) => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch('/api/qr/decode-file', { method: 'POST', body: formData });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || '解码文件失败');
    return data;
  },
  importData: async (text) => {
    const res = await fetch('/api/import', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain; charset=utf-8' },
      body: text
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || '导入失败');
    return data;
  }
};
