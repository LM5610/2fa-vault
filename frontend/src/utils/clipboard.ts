// Secure Clipboard Helper with Auto-Clear Security Protection

let autoClearTimer: number | null = null;
let lastCopiedText = '';

export function getAutoClearSeconds(): number {
  const saved = localStorage.getItem('2fa_auto_clear_seconds');
  if (saved === null) return 30; // Default 30s
  const parsed = parseInt(saved, 10);
  return isNaN(parsed) ? 30 : parsed;
}

export function setAutoClearSeconds(seconds: number): void {
  localStorage.setItem('2fa_auto_clear_seconds', seconds.toString());
}

export async function copyWithAutoClear(
  text: string,
  label: string,
  showToast: (msg: string, isError?: boolean) => void
): Promise<boolean> {
  if (!text || text === '------') return false;

  try {
    await navigator.clipboard.writeText(text);
  } catch {
    // Fallback for older browsers
    const input = document.createElement('input');
    input.value = text;
    document.body.appendChild(input);
    input.select();
    document.execCommand('copy');
    document.body.removeChild(input);
  }

  lastCopiedText = text;

  // Clear previous timer
  if (autoClearTimer !== null) {
    window.clearTimeout(autoClearTimer);
    autoClearTimer = null;
  }

  const seconds = getAutoClearSeconds();
  if (seconds > 0) {
    showToast(`已复制 ${label} · 将在 ${seconds}s 后自动清空剪贴板`);

    autoClearTimer = window.setTimeout(async () => {
      try {
        let currentText = '';
        try {
          currentText = await navigator.clipboard.readText();
        } catch {
          currentText = lastCopiedText; // If read permission not granted, assume it's still there
        }

        if (currentText === lastCopiedText) {
          await navigator.clipboard.writeText('');
          showToast('已自动擦除剪贴板中的敏感验证码，防止泄露');
        }
      } catch {
        // Silently ignore clipboard write issues in background
      } finally {
        autoClearTimer = null;
        lastCopiedText = '';
      }
    }, seconds * 1000);
  } else {
    showToast(`已复制 ${label}`);
  }

  return true;
}
