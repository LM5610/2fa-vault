import type {
  Account,
  VaultStatus,
  OTPPreviewRequest,
  OTPPreviewResponse,
  QRDecodeResponse,
  SnapshotInfo,
  WebDAVConfig,
  TimeDriftResponse,
  AccountDTO,
} from '../types/vault';
import { localVault } from '../utils/localVault';
import { generateTOTP, generateNextTOTP, generateSteamOTP, generateNextSteamOTP, generateHOTP } from '../utils/localOTP';

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

/**
 * Checks if running in 100% offline pure local standalone mode
 * Returns true if no external server URL is configured (the default for iOS & standalone app)
 */
export const isLocalMode = (): boolean => {
  return !getApiBaseUrl();
};

async function remoteRequest<T>(endpoint: string, options?: RequestInit): Promise<T> {
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
    throw new Error(`无法连接到服务器 (${baseUrl}): ${err.message || '网络连接超时'}`);
  }

  const contentType = res.headers.get('content-type') || '';
  if (!contentType.includes('application/json')) {
    throw new Error(`服务器响应异常: 返回了非 JSON 数据 (${res.status} ${res.statusText})`);
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
  isLocalMode,

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

  getStatus: async (): Promise<VaultStatus> => {
    if (isLocalMode()) {
      return localVault.getStatus();
    }
    return remoteRequest<VaultStatus>('/api/status');
  },

  initVault: async (password: string): Promise<{ status: string }> => {
    if (isLocalMode()) {
      await localVault.init(password);
      return { status: 'initialized' };
    }
    return remoteRequest<{ status: string }>('/api/init', { method: 'POST', body: JSON.stringify({ password }) });
  },

  unlockVault: async (password: string): Promise<{ status: string }> => {
    if (isLocalMode()) {
      await localVault.unlock(password);
      return { status: 'unlocked' };
    }
    return remoteRequest<{ status: string }>('/api/unlock', { method: 'POST', body: JSON.stringify({ password }) });
  },

  lockVault: async (): Promise<{ status: string }> => {
    if (isLocalMode()) {
      localVault.lock();
      return { status: 'locked' };
    }
    return remoteRequest<{ status: string }>('/api/lock', { method: 'POST' });
  },

  setAutoLock: async (minutes: number): Promise<{ status: string }> => {
    if (isLocalMode()) {
      await localVault.setAutoLockMinutes(minutes);
      return { status: 'updated' };
    }
    return remoteRequest<{ status: string }>('/api/settings/autolock', { method: 'POST', body: JSON.stringify({ minutes }) });
  },

  changePassword: async (old_password: string, new_password: string): Promise<{ status: string }> => {
    if (isLocalMode()) {
      await localVault.changePassword(old_password, new_password);
      return { status: 'changed' };
    }
    return remoteRequest<{ status: string }>('/api/settings/password', {
      method: 'POST',
      body: JSON.stringify({ old_password, new_password }),
    });
  },

  getAccounts: async (query = '', tag = ''): Promise<AccountDTO[]> => {
    if (isLocalMode()) {
      return localVault.listAccounts(query, tag);
    }
    const params = new URLSearchParams();
    if (query) params.set('query', query);
    if (tag) params.set('tag', tag);
    return remoteRequest<AccountDTO[]>(`/api/accounts?${params.toString()}`);
  },

  getAccount: async (id: string): Promise<Account> => {
    if (isLocalMode()) {
      return localVault.getAccount(id);
    }
    return remoteRequest<Account>(`/api/accounts/${id}`);
  },

  createAccount: async (data: Partial<Account>): Promise<Account> => {
    if (isLocalMode()) {
      return localVault.addAccount(data);
    }
    return remoteRequest<Account>('/api/accounts', { method: 'POST', body: JSON.stringify(data) });
  },

  updateAccount: async (id: string, data: Partial<Account>): Promise<Account> => {
    if (isLocalMode()) {
      return localVault.updateAccount(id, data);
    }
    return remoteRequest<Account>(`/api/accounts/${id}`, { method: 'PUT', body: JSON.stringify(data) });
  },

  deleteAccount: async (id: string): Promise<{ status: string }> => {
    if (isLocalMode()) {
      await localVault.deleteAccount(id);
      return { status: 'deleted' };
    }
    return remoteRequest<{ status: string }>(`/api/accounts/${id}`, { method: 'DELETE' });
  },

  incrementHOTP: async (id: string): Promise<{ code: string; counter: number }> => {
    if (isLocalMode()) {
      return localVault.incrementHOTP(id);
    }
    return remoteRequest<{ code: string; counter: number }>(`/api/accounts/${id}/increment`, { method: 'POST' });
  },

  togglePin: async (id: string): Promise<Account> => {
    if (isLocalMode()) {
      return localVault.togglePin(id);
    }
    return remoteRequest<Account>(`/api/accounts/${id}/pin`, { method: 'POST' });
  },

  toggleRecoveryCode: async (id: string, index: number): Promise<Account> => {
    if (isLocalMode()) {
      return localVault.toggleRecoveryCode(id, index);
    }
    return remoteRequest<Account>(`/api/accounts/${id}/recovery`, { method: 'POST', body: JSON.stringify({ index }) });
  },

  reorderAccounts: async (ids: string[]): Promise<{ success: boolean }> => {
    if (isLocalMode()) {
      await localVault.reorderAccounts(ids);
      return { success: true };
    }
    return remoteRequest<{ success: boolean }>('/api/accounts/reorder', { method: 'POST', body: JSON.stringify({ ids }) });
  },

  getAccountQR: async (id: string): Promise<{ qr_data: string; otpauth_uri: string }> => {
    if (isLocalMode()) {
      return localVault.getAccountQR(id);
    }
    return remoteRequest<{ qr_data: string; otpauth_uri: string }>(`/api/accounts/${id}/qr`);
  },

  getTags: async (): Promise<string[]> => {
    if (isLocalMode()) {
      return localVault.getTags();
    }
    return remoteRequest<string[]>('/api/tags');
  },

  getExportData: async (format: 'json' | 'uri' = 'json'): Promise<{ data: string; format: string }> => {
    if (isLocalMode()) {
      return { data: localVault.getExportData(format), format };
    }
    return remoteRequest<{ data: string; format: string }>(`/api/export?format=${format}`);
  },

  importData: async (data: string): Promise<{ imported: number }> => {
    if (isLocalMode()) {
      const imported = await localVault.importData(data);
      return { imported };
    }
    return remoteRequest<{ imported: number }>('/api/import', { method: 'POST', body: JSON.stringify({ data }) });
  },

  // Snapshots & Rollback
  getSnapshots: async (): Promise<SnapshotInfo[]> => {
    if (isLocalMode()) {
      return localVault.getSnapshots();
    }
    return remoteRequest<SnapshotInfo[]>('/api/vault/snapshots');
  },

  rollbackSnapshot: async (filename: string): Promise<{ success: boolean; message: string }> => {
    if (isLocalMode()) {
      await localVault.rollbackSnapshot(filename);
      return { success: true, message: '快照已在本地成功回滚' };
    }
    return remoteRequest<{ success: boolean; message: string }>('/api/vault/snapshots/rollback', {
      method: 'POST',
      body: JSON.stringify({ filename }),
    });
  },

  // WebDAV Cloud Backup & Auto-Sync
  getWebDAVConfig: async (): Promise<WebDAVConfig> => {
    if (isLocalMode()) {
      return localVault.getWebDAVConfig();
    }
    return remoteRequest<WebDAVConfig>('/api/webdav/config');
  },

  saveWebDAVConfig: async (cfg: Partial<WebDAVConfig>): Promise<{ success: boolean; message: string; config?: WebDAVConfig }> => {
    if (isLocalMode()) {
      localVault.saveWebDAVConfig(cfg);
      return { success: true, message: 'WebDAV 本地配置已保存' };
    }
    return remoteRequest<{ success: boolean; message: string; config?: WebDAVConfig }>('/api/webdav/config', {
      method: 'POST',
      body: JSON.stringify(cfg),
    });
  },

  testWebDAV: async (data?: { url?: string; username?: string; password?: string }): Promise<{ success: boolean; message: string }> => {
    const cfg = isLocalMode() ? { ...localVault.getWebDAVConfig(), ...data } : data;
    if (!cfg || !cfg.url) throw new Error('缺少 WebDAV 地址');
    const authHeader = 'Basic ' + btoa(`${cfg.username}:${cfg.password}`);
    const res = await fetch(cfg.url, {
      method: 'PROPFIND',
      headers: { Authorization: authHeader, Depth: '0' },
    });
    if (res.ok || res.status === 207 || res.status === 405) {
      return { success: true, message: 'WebDAV 服务器连接成功！' };
    }
    throw new Error(`WebDAV 连接失败: HTTP ${res.status}`);
  },

  backupWebDAV: async (): Promise<{ success: boolean; message: string; last_sync?: string }> => {
    if (isLocalMode()) {
      const now = await localVault.backupToWebDAV();
      return { success: true, message: '已成功备份到 WebDAV 云端！', last_sync: now };
    }
    return remoteRequest<{ success: boolean; message: string; last_sync?: string }>('/api/webdav/backup', { method: 'POST' });
  },

  restoreWebDAV: async (): Promise<{ success: boolean; message: string; locked?: boolean }> => {
    if (isLocalMode()) {
      await localVault.restoreFromWebDAV();
      return { success: true, message: '已从 WebDAV 云端恢复数据，请使用原主密码解锁', locked: true };
    }
    return remoteRequest<{ success: boolean; message: string; locked?: boolean }>('/api/webdav/restore', { method: 'POST' });
  },

  previewOTP: async (req: OTPPreviewRequest): Promise<OTPPreviewResponse> => {
    if (isLocalMode()) {
      const now = Date.now();
      const period = req.period || 30;
      let code = '------';
      let nextCode = '------';

      if (req.type === 'steam') {
        code = await generateSteamOTP(req.secret, now);
        nextCode = await generateNextSteamOTP(req.secret, now);
      } else if (req.type === 'hotp') {
        code = await generateHOTP(req.secret, 0, req.digits || 6, req.algorithm || 'SHA1');
      } else {
        code = await generateTOTP(req.secret, period, req.digits || 6, req.algorithm || 'SHA1', now);
        nextCode = await generateNextTOTP(req.secret, period, req.digits || 6, req.algorithm || 'SHA1', now);
      }

      const remaining = period - (Math.floor(now / 1000) % period);
      const progress = ((Math.floor(now / 1000) % period) / period) * 100;
      return { code, next_code: nextCode, remaining, progress };
    }
    return remoteRequest<OTPPreviewResponse>('/api/otp/preview', { method: 'POST', body: JSON.stringify(req) });
  },

  getTimeDrift: async (): Promise<TimeDriftResponse> => {
    if (isLocalMode()) {
      return {
        system_time: new Date().toISOString(),
        network_time: new Date().toISOString(),
        drift_seconds: 0,
        is_drifted: false,
      };
    }
    return remoteRequest<TimeDriftResponse>('/api/time/drift');
  },

  decodeImageURL: async (url: string): Promise<QRDecodeResponse> => {
    if (isLocalMode()) {
      const res = await fetch(url);
      const blob = await res.blob();
      const decoded = await localVault.decodeImageFile(blob);
      return { success: true, text: decoded.uri, otpauth_uri: decoded.uri };
    }
    return remoteRequest<QRDecodeResponse>('/api/qr/decode-url', { method: 'POST', body: JSON.stringify({ url }) });
  },

  decodeImageFile: async (file: File | Blob): Promise<QRDecodeResponse> => {
    if (isLocalMode()) {
      const decoded = await localVault.decodeImageFile(file);
      return { success: true, text: decoded.uri, otpauth_uri: decoded.uri };
    }
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch(`${getApiBaseUrl()}/api/qr/decode-file`, {
      method: 'POST',
      body: formData,
    });
    if (!res.ok) throw new Error('二维码解析失败');
    return res.json();
  },
};
