// Pure Local Standalone Vault Engine
// 100% offline, zero network required, WebCrypto encrypted storage.

import type { Account, VaultStatus, SnapshotInfo, WebDAVConfig, AccountDTO } from '../types/vault';
import { encryptData, decryptData, type EncryptedPayload } from './localCrypto';
import {
  generateTOTP,
  generateNextTOTP,
  generateSteamOTP,
  generateNextSteamOTP,
  generateHOTP,
  buildOTPAuthURI,
  parseOTPAuthURI,
  parseGoogleMigrationURI,
} from './localOTP';
import QRCode from 'qrcode';
import jsQR from 'jsqr';

const STORAGE_KEY = 'vault_local_encrypted';
const SNAPSHOTS_KEY = 'vault_local_snapshots';
const WEBDAV_KEY = 'vault_local_webdav_config';

interface LocalVaultData {
  version: number;
  accounts: Account[];
  auto_lock_minutes: number;
  updated_at: string;
}

interface LocalSnapshot {
  id: string;
  name: string;
  timestamp: string;
  size: number;
  payload: EncryptedPayload;
}

class LocalVaultManager {
  private inMemoryData: LocalVaultData | null = null;
  private currentPassword: string | null = null;

  isInitialized(): boolean {
    return !!localStorage.getItem(STORAGE_KEY);
  }

  isLocked(): boolean {
    return this.inMemoryData === null;
  }

  getStatus(): VaultStatus {
    const initialized = this.isInitialized();
    const locked = this.isLocked();
    const auto_lock_minutes = this.inMemoryData?.auto_lock_minutes || 15;
    return { initialized, locked, auto_lock_minutes };
  }

  async init(password: string): Promise<void> {
    if (this.isInitialized()) {
      throw new Error('保险库已初始化');
    }
    if (password.length < 6) {
      throw new Error('主密码长度至少为 6 位');
    }

    const initialData: LocalVaultData = {
      version: 1,
      accounts: [],
      auto_lock_minutes: 15,
      updated_at: new Date().toISOString(),
    };

    const payload = await encryptData(initialData, password);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));

    this.inMemoryData = initialData;
    this.currentPassword = password;
    this.saveSnapshot(payload, '初始化保险库');
  }

  async unlock(password: string): Promise<void> {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      throw new Error('本地保险库未初始化');
    }

    const payload: EncryptedPayload = JSON.parse(raw);
    const data = await decryptData<LocalVaultData>(payload, password);

    this.inMemoryData = data;
    this.currentPassword = password;
  }

  lock(): void {
    this.inMemoryData = null;
    this.currentPassword = null;
  }

  async changePassword(oldPass: string, newPass: string): Promise<void> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    if (this.currentPassword !== oldPass) {
      throw new Error('原主密码不正确');
    }
    if (newPass.length < 6) {
      throw new Error('新主密码长度至少为 6 位');
    }

    this.currentPassword = newPass;
    await this.persist();
  }

  async setAutoLockMinutes(minutes: number): Promise<void> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    this.inMemoryData.auto_lock_minutes = minutes;
    await this.persist();
  }

  private async persist(): Promise<void> {
    if (!this.inMemoryData || !this.currentPassword) return;
    this.inMemoryData.updated_at = new Date().toISOString();

    const payload = await encryptData(this.inMemoryData, this.currentPassword);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
    this.saveSnapshot(payload, `变更数据 (${this.inMemoryData.accounts.length} 个令牌)`);
  }

  private saveSnapshot(payload: EncryptedPayload, name: string) {
    try {
      const existing: LocalSnapshot[] = JSON.parse(localStorage.getItem(SNAPSHOTS_KEY) || '[]');
      const snap: LocalSnapshot = {
        id: `snap_${Date.now()}`,
        name: `${name} - ${new Date().toLocaleTimeString()}`,
        timestamp: new Date().toISOString(),
        size: JSON.stringify(payload).length,
        payload,
      };
      existing.unshift(snap);
      if (existing.length > 10) existing.splice(10);
      localStorage.setItem(SNAPSHOTS_KEY, JSON.stringify(existing));
    } catch {}
  }

  getSnapshots(): SnapshotInfo[] {
    try {
      const existing: LocalSnapshot[] = JSON.parse(localStorage.getItem(SNAPSHOTS_KEY) || '[]');
      return existing.map(s => ({
        filename: s.id,
        created_at: s.timestamp,
        size: s.size,
      }));
    } catch {
      return [];
    }
  }

  async rollbackSnapshot(snapshotId: string): Promise<void> {
    const existing: LocalSnapshot[] = JSON.parse(localStorage.getItem(SNAPSHOTS_KEY) || '[]');
    const snap = existing.find(s => s.id === snapshotId);
    if (!snap) throw new Error('快照不存在');

    localStorage.setItem(STORAGE_KEY, JSON.stringify(snap.payload));
    if (this.currentPassword) {
      this.inMemoryData = await decryptData<LocalVaultData>(snap.payload, this.currentPassword);
    } else {
      this.inMemoryData = null;
    }
  }

  async listAccounts(query = '', tag = ''): Promise<AccountDTO[]> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');

    let list = [...this.inMemoryData.accounts];
    if (tag) {
      list = list.filter(a => a.tags && a.tags.includes(tag));
    }
    if (query.trim()) {
      const q = query.toLowerCase().trim();
      list = list.filter(
        a =>
          a.issuer.toLowerCase().includes(q) ||
          a.account_name.toLowerCase().includes(q) ||
          (a.tags && a.tags.some(t => t.toLowerCase().includes(q)))
      );
    }

    const now = Date.now();
    const dtos: AccountDTO[] = [];

    for (const acc of list) {
      let code = '------';
      let nextCode = '------';
      const period = acc.period || 30;
      let remaining = period - (Math.floor(now / 1000) % period);
      let progress = ((Math.floor(now / 1000) % period) / period) * 100;

      if (acc.type === 'steam') {
        code = await generateSteamOTP(acc.secret, now);
        nextCode = await generateNextSteamOTP(acc.secret, now);
      } else if (acc.type === 'hotp') {
        code = await generateHOTP(acc.secret, acc.counter || 0, acc.digits || 6, acc.algorithm || 'SHA1');
        remaining = 0;
        progress = 100;
      } else {
        code = await generateTOTP(acc.secret, period, acc.digits || 6, acc.algorithm || 'SHA1', now);
        nextCode = await generateNextTOTP(acc.secret, period, acc.digits || 6, acc.algorithm || 'SHA1', now);
      }

      dtos.push({
        ...acc,
        code,
        next_code: nextCode,
        remaining,
        progress,
      });
    }

    return dtos;
  }

  getAccount(id: string): Account {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const acc = this.inMemoryData.accounts.find(a => a.id === id);
    if (!acc) throw new Error('账号未找到');
    return { ...acc };
  }

  async addAccount(accData: Partial<Account>): Promise<Account> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const id = `acc_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;

    const newAcc: Account = {
      id,
      issuer: accData.issuer || '未知服务商',
      account_name: accData.account_name || '默认账号',
      secret: (accData.secret || '').toUpperCase().replace(/[\s\-_=]/g, ''),
      algorithm: accData.algorithm || 'SHA1',
      digits: accData.digits || 6,
      period: accData.period || 30,
      type: accData.type || 'totp',
      counter: accData.counter || 0,
      tags: accData.tags || [],
      notes: accData.notes || '',
      pinned: accData.pinned || false,
      recovery_codes: accData.recovery_codes || [],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    this.inMemoryData.accounts.unshift(newAcc);
    await this.persist();
    return newAcc;
  }

  async updateAccount(id: string, accData: Partial<Account>): Promise<Account> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const idx = this.inMemoryData.accounts.findIndex(a => a.id === id);
    if (idx === -1) throw new Error('账号未找到');

    const updated: Account = {
      ...this.inMemoryData.accounts[idx],
      ...accData,
      id,
      updated_at: new Date().toISOString(),
    };

    this.inMemoryData.accounts[idx] = updated;
    await this.persist();
    return updated;
  }

  async deleteAccount(id: string): Promise<void> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    this.inMemoryData.accounts = this.inMemoryData.accounts.filter(a => a.id !== id);
    await this.persist();
  }

  async incrementHOTP(id: string): Promise<{ code: string; counter: number }> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const acc = this.inMemoryData.accounts.find(a => a.id === id);
    if (!acc) throw new Error('账号未找到');

    acc.counter = (acc.counter || 0) + 1;
    acc.updated_at = new Date().toISOString();
    await this.persist();

    const code = await generateHOTP(acc.secret, acc.counter, acc.digits || 6, acc.algorithm || 'SHA1');
    return { code, counter: acc.counter };
  }

  async togglePin(id: string): Promise<Account> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const acc = this.inMemoryData.accounts.find(a => a.id === id);
    if (!acc) throw new Error('账号未找到');

    acc.pinned = !acc.pinned;
    acc.updated_at = new Date().toISOString();
    await this.persist();
    return { ...acc };
  }

  async toggleRecoveryCode(id: string, codeIndex: number): Promise<Account> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const acc = this.inMemoryData.accounts.find(a => a.id === id);
    if (!acc || !acc.recovery_codes || !acc.recovery_codes[codeIndex]) {
      throw new Error('未找到该备用码');
    }

    acc.recovery_codes[codeIndex].used = !acc.recovery_codes[codeIndex].used;
    acc.updated_at = new Date().toISOString();
    await this.persist();
    return { ...acc };
  }

  async reorderAccounts(ids: string[]): Promise<void> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const orderMap = new Map<string, number>();
    ids.forEach((id, index) => orderMap.set(id, index));

    this.inMemoryData.accounts.sort((a, b) => {
      const orderA = orderMap.has(a.id) ? orderMap.get(a.id)! : 9999;
      const orderB = orderMap.has(b.id) ? orderMap.get(b.id)! : 9999;
      return orderA - orderB;
    });

    await this.persist();
  }

  getTags(): string[] {
    if (!this.inMemoryData) return [];
    const set = new Set<string>();
    for (const a of this.inMemoryData.accounts) {
      if (a.tags) {
        a.tags.forEach(t => set.add(t));
      }
    }
    return Array.from(set);
  }

  async getAccountQR(id: string): Promise<{ qr_data: string; otpauth_uri: string }> {
    const acc = this.getAccount(id);
    const uri = buildOTPAuthURI(acc);
    const qrData = await QRCode.toDataURL(uri, {
      margin: 1,
      width: 256,
      errorCorrectionLevel: 'M',
    });
    return { qr_data: qrData, otpauth_uri: uri };
  }

  async decodeImageFile(file: File | Blob): Promise<{ uri: string }> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(file);
      img.onload = () => {
        URL.revokeObjectURL(url);
        const canvas = document.createElement('canvas');
        canvas.width = img.width;
        canvas.height = img.height;
        const ctx = canvas.getContext('2d');
        if (!ctx) return reject(new Error('无法创建图像上下文'));

        ctx.drawImage(img, 0, 0);
        const imgData = ctx.getImageData(0, 0, canvas.width, canvas.height);
        const code = jsQR(imgData.data, imgData.width, imgData.height);
        if (code && code.data) {
          resolve({ uri: code.data });
        } else {
          reject(new Error('未在图片中检测到有效的二维码'));
        }
      };
      img.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error('加载图片文件失败'));
      };
      img.src = url;
    });
  }

  async importData(text: string): Promise<number> {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    const str = text.trim();
    let importedCount = 0;

    // 1. Google Authenticator Migration URI: otpauth-migration://
    if (str.includes('otpauth-migration://')) {
      const lines = str.split('\n');
      for (const line of lines) {
        const trimmed = line.trim();
        if (trimmed.startsWith('otpauth-migration://')) {
          try {
            const list = parseGoogleMigrationURI(trimmed);
            for (const item of list) {
              await this.addAccount({
                issuer: item.issuer,
                account_name: item.account,
                secret: item.secret,
                algorithm: item.algorithm,
                digits: item.digits,
                period: item.period,
                type: item.type,
                counter: item.counter,
              });
              importedCount++;
            }
          } catch {}
        }
      }
      if (importedCount > 0) return importedCount;
    }

    // 2. Standard otpauth:// lines
    if (str.includes('otpauth://')) {
      const lines = str.split('\n');
      for (const line of lines) {
        const trimmed = line.trim();
        if (trimmed.startsWith('otpauth://')) {
          try {
            const p = parseOTPAuthURI(trimmed);
            await this.addAccount({
              issuer: p.issuer,
              account_name: p.account,
              secret: p.secret,
              algorithm: p.algorithm,
              digits: p.digits,
              period: p.period,
              type: p.type,
              counter: p.counter,
            });
            importedCount++;
          } catch {}
        }
      }
      if (importedCount > 0) return importedCount;
    }

    // 3. Aegis JSON export
    try {
      const json = JSON.parse(str);
      if (json && json.db && Array.isArray(json.db.entries)) {
        for (const e of json.db.entries) {
          const algo = (e.info?.algo || 'SHA1').toUpperCase();
          await this.addAccount({
            issuer: e.issuer || e.name || 'Aegis Import',
            account_name: e.name || 'Account',
            secret: e.info?.secret || '',
            algorithm: algo === 'SHA256' ? 'SHA256' : algo === 'SHA512' ? 'SHA512' : 'SHA1',
            digits: e.info?.digits || 6,
            period: e.info?.period || 30,
            type: e.type?.toLowerCase() === 'hotp' ? 'hotp' : e.type?.toLowerCase() === 'steam' ? 'steam' : 'totp',
            counter: e.info?.counter || 0,
            tags: e.group ? [e.group] : [],
            notes: e.note || '',
          });
          importedCount++;
        }
        return importedCount;
      }
    } catch {}

    throw new Error('未能识别的数据格式，支持 Google Authenticator 迁移码、otpauth:// 列表或 Aegis JSON');
  }

  getExportData(format: 'json' | 'uri' = 'json'): string {
    if (!this.inMemoryData) throw new Error('保险库已锁定');
    if (format === 'uri') {
      return this.inMemoryData.accounts.map(a => buildOTPAuthURI(a)).join('\n');
    }

    // Export Aegis compatible JSON
    const aegis = {
      version: 1,
      header: { slots: [] },
      db: {
        version: 1,
        entries: this.inMemoryData.accounts.map(a => ({
          type: a.type || 'totp',
          name: a.account_name,
          issuer: a.issuer,
          note: a.notes || '',
          group: a.tags && a.tags.length > 0 ? a.tags[0] : '',
          info: {
            secret: a.secret,
            algo: a.algorithm || 'SHA1',
            digits: a.digits || 6,
            period: a.period || 30,
            counter: a.counter || 0,
          },
        })),
      },
    };
    return JSON.stringify(aegis, null, 2);
  }

  // WebDAV direct sync
  getWebDAVConfig(): WebDAVConfig {
    const raw = localStorage.getItem(WEBDAV_KEY);
    return raw ? JSON.parse(raw) : { url: '', username: '', password: '', auto_sync: false, remote_dir: '/2FA-Vault' };
  }

  saveWebDAVConfig(cfg: Partial<WebDAVConfig>): void {
    const current = this.getWebDAVConfig();
    const updated = { ...current, ...cfg };
    localStorage.setItem(WEBDAV_KEY, JSON.stringify(updated));
  }

  async backupToWebDAV(): Promise<string> {
    const cfg = this.getWebDAVConfig();
    if (!cfg.url || !cfg.username || !cfg.password) throw new Error('未配置 WebDAV 信息');

    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) throw new Error('本地无数据');

    const cleanBase = cfg.url.replace(/\/+$/, '');
    const cleanDir = (cfg.remote_dir || '/2FA-Vault').replace(/^\/+/, '').replace(/\/+$/, '');
    const targetUrl = `${cleanBase}/${cleanDir}/vault.enc`;

    const authHeader = 'Basic ' + btoa(`${cfg.username}:${cfg.password}`);
    const res = await fetch(targetUrl, {
      method: 'PUT',
      headers: {
        Authorization: authHeader,
        'Content-Type': 'application/octet-stream',
      },
      body: raw,
    });

    if (!res.ok) throw new Error(`WebDAV 上传失败: HTTP ${res.status}`);
    const now = new Date().toISOString();
    this.saveWebDAVConfig({ last_sync: now });
    return now;
  }

  async restoreFromWebDAV(password?: string): Promise<void> {
    const cfg = this.getWebDAVConfig();
    if (!cfg.url || !cfg.username || !cfg.password) throw new Error('未配置 WebDAV 信息');

    const cleanBase = cfg.url.replace(/\/+$/, '');
    const cleanDir = (cfg.remote_dir || '/2FA-Vault').replace(/^\/+/, '').replace(/\/+$/, '');
    const targetUrl = `${cleanBase}/${cleanDir}/vault.enc`;

    const authHeader = 'Basic ' + btoa(`${cfg.username}:${cfg.password}`);
    const res = await fetch(targetUrl, {
      method: 'GET',
      headers: { Authorization: authHeader },
    });

    if (!res.ok) throw new Error(`WebDAV 下载失败: HTTP ${res.status}`);
    const rawText = await res.text();
    localStorage.setItem(STORAGE_KEY, rawText);

    if (password) {
      await this.unlock(password);
    } else {
      this.lock();
    }
  }
}

export const localVault = new LocalVaultManager();
