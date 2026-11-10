import type { Account, VaultStatus, OTPPreviewRequest, OTPPreviewResponse, QRDecodeResponse, SyncInfoResponse, SyncPairInitResponse } from '../types/vault';

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const res = await fetch(endpoint, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

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

  // LAN Direct Sync
  getSyncInfo: () => request<SyncInfoResponse>('/api/sync/info'),
  initSyncPair: () => request<SyncPairInitResponse>('/api/sync/pair/init', { method: 'POST' }),
  downloadPairedVault: async (token: string): Promise<Blob> => {
    const res = await fetch(`/api/sync/pair/vault?token=${encodeURIComponent(token)}`);
    if (!res.ok) throw new Error('下载同步加密数据失败或配对码已失效');
    return res.blob();
  },
  uploadPairedVault: async (token: string, data: Blob): Promise<{ success: boolean }> => {
    const res = await fetch(`/api/sync/pair/vault?token=${encodeURIComponent(token)}`, {
      method: 'POST',
      body: data,
    });
    if (!res.ok) throw new Error('上传同步加密数据失败');
    return res.json();
  },

  previewOTP: (req: OTPPreviewRequest) => request<OTPPreviewResponse>('/api/otp/preview', { method: 'POST', body: JSON.stringify(req) }),
  decodeImageURL: (url: string) => request<QRDecodeResponse>('/api/qr/decode-url', { method: 'POST', body: JSON.stringify({ url }) }),
  decodeImageFile: async (file: File): Promise<QRDecodeResponse> => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch('/api/qr/decode-file', {
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
