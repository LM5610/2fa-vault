// iOS & iCloud Backup Module

import { showToast } from './ui.js';

// Check if device is iOS (iPhone/iPad)
export function isIOS() {
  return /iPad|iPhone|iPod/.test(navigator.userAgent) || 
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
}

// Check if native Share Sheet with File support is available
export function canShareFiles() {
  return typeof navigator.canShare === 'function';
}

/**
 * Backup to iCloud Drive or Local File.
 * On iOS devices, navigator.share with a file opens the native iOS Share Sheet,
 * allowing the user to tap "存储到‘文件’" -> "iCloud 云盘" directly.
 */
export async function backupVault(format = 'json') {
  try {
    showToast('正在生成备份数据...');
    const url = `/api/export?format=${format}`;
    const res = await fetch(url);
    if (!res.ok) throw new Error('导出数据失败');

    const content = await res.text();
    const dateStr = new Date().toISOString().slice(0, 10);
    const fileName = format === 'uri' 
      ? `2fa-vault-backup-${dateStr}.txt` 
      : `2fa-vault-backup-${dateStr}.json`;
    const mimeType = format === 'uri' ? 'text/plain' : 'application/json';

    const blob = new Blob([content], { type: mimeType });
    const file = new File([blob], fileName, { type: mimeType });

    // iOS Safari / PWA native share to iCloud Drive
    if (canShareFiles() && navigator.canShare({ files: [file] })) {
      try {
        await navigator.share({
          title: '2FA Vault 安全备份',
          text: '保存您的 2FA 令牌备份到 iCloud 云盘或本地文件',
          files: [file]
        });
        showToast('备份已调起，请在面板中选择“存储到‘文件’ ➔ iCloud 云盘”');
        return;
      } catch (shareErr) {
        if (shareErr.name === 'AbortError') {
          // User cancelled share sheet
          return;
        }
        // Fallback to direct download if share failed
        console.warn('Share sheet failed, falling back to download:', shareErr);
      }
    }

    // Standard download fallback
    const downloadUrl = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = downloadUrl;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(downloadUrl);

    if (isIOS()) {
      showToast('已开始下载，请在 Safari 下载项中选择“存储到‘文件’ ➔ iCloud 云盘”');
    } else {
      showToast('备份文件已保存到本地下载目录');
    }
  } catch (err) {
    showToast(`备份失败: ${err.message}`, true);
  }
}
