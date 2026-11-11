import type { Account, VaultStatus, OTPPreviewRequest, OTPPreviewResponse, QRDecodeResponse, SyncInfoResponse, SyncPairInitResponse, SnapshotInfo, WebDAVConfig, TimeDriftResponse } from '../types/vault';

export const getApiBaseUrl = (): string => {
  return localStorage.getItem('vault_api_server') || '';
};

export const setApiBaseUrl = (url: string) => {
  if (url) {
    localStorage.setItem('vault_api_server', url.replace(/\/+$/, ''));
  } else {
    localStorage.removeItem('vault_api_server');
  }
};

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const fullUrl = endpoint.startsWith('http') ? endpoint : `${baseUrl}${endpoint}`;

  let res: Response;
  try {
    res = await fetch(fullUrl, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });
  } catch (err: any) {
    throw new Error(`无法连接到服务器 (${baseUrl || '本地服务'}): ${err.message || '网络连接超时'}`);
  }

  // Prevent WebKit "The string did not match the expected pattern" error when Capacitor serves index.html for unknown /api routes
  const contentType = res.headers.get('content-type') || '';
  if (!contentType.includes('application/json')) {
    if (!baseUrl && (fullUrl.startsWith('capacitor://') || (typeof window !== 'undefined' && window.location.protocol === 'capacitor:'))) {
      throw new Error('未连接到后端服务：请在下方点击【设置服务器】填入电脑服务端的局域网地址 (如 http://192.168.1.X:2233)');
    }
    throw new Error(`服务器响应异常: 返回了非 JSON 数据 (${res.status} ${res.statusText})，请检查服务器地址`);
  }

  if (!res.ok) {
    let errorMsg = `HTTP Error ${res.status}`;
    try {
      const errData = await res.json();
      if (errData && errData.error) errorMsg = errData.error;
    } catch {}
    throw new Error(errorMsg);
  }

  return res.json() as Promise<T>;
}

export const API = {
  testServerUrl: async (url: string): Promise<VaultStatus> => {
    const cleanUrl = url.replace(/\/+$/, '');
    const res = await fetch(`${cleanUrl}/api/status`, { headers: { 'Content-Type': 'application/json' } });
    const contentType = res.headers.get('content-type') || '';
    if (!contentType.includes('application/json')) {
      throw new Error('目标地址未运行 2FA Vault 后端服务 (返回非 JSON 响应)');
    }
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },
  getStatus: () => request<VaultStatus>('/api/status'),
  initVault: (password: string) => request<{ status: string }>('/api/init', { method: 'POST', body: JSON.stringify({ password }) }),
  unlockVault: (password: string) => request<{ status: string }>('/api/unlock', { method: 'POST', body: JSON.stringify({ password }) }),
  lockVault: () => request<{ status: string }>('/api/lock', { method: 'POST' }),
  setAutoLock: (minutes: number) => request<{ status: string }>('/api/settings/autolock', { method: 'POST', body: JSON.stringify({ minutes }) }),
  changePassword: (old_password: string, new_password: string) => request<{ status: string }>('/api/settings/password', { method: 'POST', body: JSON.stringify({ old_password, new_password }) }),

  getAccounts: (query = '', tag = '') => {
    const params = new URLSearchParams();
    if (query) params.set('query', query);
    if (tag) params.set('tag', tag);
    return request<Account[]>(`/api/accounts?${params.toString()}`);
  },
  getAccount: (id: string) => request<Account>(`/api/accounts/${id}`),
  createAccount: (data: Partial<Account>) => request<Account>('/api/accounts', { method: 'POST', body: JSON.stringify(data) }),
  updateAccount: (id: string, data: Partial<Account>) => request<Account>(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteAccount: (id: string) => request<{ status: string }>(`/api/accounts/${id}`, { method: 'DELETE' }),
  incrementHOTP: (id: string) => request<{ code: string; counter: number }>(`/api/accounts/${id}/increment`, { method: 'POST' }),
  togglePin: (id: string) => request<Account>(`/api/accounts/${id}/pin`, { method: 'POST' }),
  toggleRecoveryCode: (id: string, index: number) => request<Account>(`/api/accounts/${id}/recovery`, { method: 'POST', body: JSON.stringify({ index }) }),
  reorderAccounts: (ids: string[]) => request<{ success: boolean }>('/api/accounts/reorder', { method: 'POST', body: JSON.stringify({ ids }) }),
  getAccountQR: (id: string) => request<{ qr_data: string; otpauth_uri: string }>(`/api/accounts/${id}/qr`),

  getTags: () => request<string[]>('/api/tags'),
  getExportData: (format: 'json' | 'uri' = 'json') => request<{ data: string; format: string }>(`/api/export?format=${format}`),
  importData: (data: string) => request<{ imported: number }>('/api/import', { method: 'POST', body: JSON.stringify({ data }) }),

  // Snapshots & Rollback
  getSnapshots: () => request<SnapshotInfo[]>('/api/vault/snapshots'),
  rollbackSnapshot: (filename: string) => request<{ success: boolean; message: string }>('/api/vault/snapshots/rollback', {
    method: 'POST',
    body: JSON.stringify({ filename }),
  }),

  // WebDAV Cloud Backup & Auto-Sync
  getWebDAVConfig: () => request<WebDAVConfig>('/api/webdav/config'),
  saveWebDAVConfig: (cfg: Partial<WebDAVConfig>) => request<{ success: boolean; message: string; config?: WebDAVConfig }>('/api/webdav/config', {
    method: 'POST',
    body: JSON.stringify(cfg),
  }),
  testWebDAV: (data?: { url?: string; username?: string; password?: string }) => request<{ success: boolean; message: string }>('/api/webdav/test', {
    method: 'POST',
    body: JSON.stringify(data || {}),
  }),
  backupWebDAV: () => request<{ success: boolean; message: string; last_sync?: string }>('/api/webdav/backup', {
    method: 'POST',
  }),
  restoreWebDAV: () => request<{ success: boolean; message: string; locked?: boolean }>('/api/webdav/restore', {
    method: 'POST',
  }),

  // LAN Direct Sync
  getSyncInfo: () => request<SyncInfoResponse>('/api/sync/info'),
  initSyncPair: () => request<SyncPairInitResponse>('/api/sync/pair/init', { method: 'POST' }),
  downloadPairedVault: async (token: string): Promise<Blob> => {
    const res = await fetch(`${getApiBaseUrl()}/api/sync/pair/vault?token=${encodeURIComponent(token)}`);
    if (!res.ok) throw new Error('下载同步加密数据失败或配对码已失效');
    return res.blob();
  },
  uploadPairedVault: async (token: string, data: Blob): Promise<{ success: boolean }> => {
    const res = await fetch(`${getApiBaseUrl()}/api/sync/pair/vault?token=${encodeURIComponent(token)}`, {
      method: 'POST',
      body: data,
    });
    if (!res.ok) throw new Error('上传同步加密数据失败');
    return res.json();
  },

  previewOTP: (req: OTPPreviewRequest) => request<OTPPreviewResponse>('/api/otp/preview', { method: 'POST', body: JSON.stringify(req) }),
  getTimeDrift: () => request<TimeDriftResponse>('/api/time/drift'),
  decodeImageURL: (url: string) => request<QRDecodeResponse>('/api/qr/decode-url', { method: 'POST', body: JSON.stringify({ url }) }),
  decodeImageFile: async (file: File): Promise<QRDecodeResponse> => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch(`${getApiBaseUrl()}/api/qr/decode-file`, {
      method: 'POST',
      body: formData,
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || 'Failed to decode QR code');
    }
    return res.json();
  },
};
