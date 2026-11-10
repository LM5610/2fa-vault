// Scanner & QR Code Parsing Module (Camera, Image URL, File)

import { state } from './state.js';
import { API } from './api.js';
import { showToast } from './ui.js';

export function populateFormFromParsed(parsed) {
  if (!parsed) return;
  if (parsed.issuer) document.getElementById('accIssuer').value = parsed.issuer;
  if (parsed.account_name) document.getElementById('accAccount').value = parsed.account_name;
  if (parsed.secret) document.getElementById('accSecret').value = parsed.secret;
  if (parsed.type) document.getElementById('accType').value = parsed.type;
  if (parsed.digits) document.getElementById('accDigits').value = parsed.digits;
  if (parsed.period) document.getElementById('accPeriod').value = parsed.period;
  if (parsed.algorithm) document.getElementById('accAlgo').value = parsed.algorithm;

  window.dispatchEvent(new CustomEvent('refreshModalPreview'));
}

export function populateFormFromURI(uriStr) {
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

    populateFormFromParsed({
      issuer,
      account_name: account,
      secret,
      type: type === 'hotp' ? 'hotp' : 'totp',
      digits,
      period,
      algorithm: ['SHA1', 'SHA256', 'SHA512'].includes(algo) ? algo : 'SHA1'
    });
  } catch (e) {
    console.warn('Parse URI error:', e);
  }
}

// Camera control
export async function startCamera() {
  stopCamera();
  const video = document.getElementById('scannerVideo');
  if (!video || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
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
          // ignore detection tick error
        }
      }, 300);
    }
  } catch (err) {
    console.warn('Camera access denied or unavailable:', err);
  }
}

export function stopCamera() {
  if (state.scanInterval) {
    clearInterval(state.scanInterval);
    state.scanInterval = null;
  }
  if (state.scanStream) {
    state.scanStream.getTracks().forEach(t => t.stop());
    state.scanStream = null;
  }
}

// Switch tabs inside Add Modal
export function switchAddTab(tabId) {
  document.querySelectorAll('#addTabs .tab-btn').forEach(btn => {
    btn.classList.toggle('active', btn.getAttribute('data-tab') === tabId);
  });
  document.querySelectorAll('.tab-content').forEach(c => {
    c.style.display = c.id === tabId ? 'block' : 'none';
  });

  if (tabId === 'tabScan') {
    startCamera();
  } else {
    stopCamera();
  }
}

// Parse Image from URL (Requirement 1)
export async function parseFromImageURL(url) {
  if (!url) {
    showToast('请输入图片链接', true);
    return;
  }

  const btn = document.getElementById('parseURLBtn');
  const previewContainer = document.getElementById('urlPreviewContainer');
  const previewImg = document.getElementById('urlPreviewImg');

  if (btn) btn.disabled = true;
  showToast('正在获取并解析图片二维码...');

  try {
    // Show image preview
    if (previewContainer && previewImg) {
      previewImg.src = url.startsWith('data:') ? url : `/api/proxy-image?url=${encodeURIComponent(url)}`;
      previewContainer.style.display = 'flex';
    }

    const res = await API.decodeImageURL(url);
    if (!res || !res.text) {
      throw new Error('未识别到有效的二维码');
    }

    if (res.parsed) {
      populateFormFromParsed(res.parsed);
    } else {
      populateFormFromURI(res.text);
    }

    switchAddTab('tabManual');
    stopCamera();
    showToast('图片二维码解析成功！已自动填充信息');
  } catch (err) {
    showToast(`解析失败: ${err.message}`, true);
  } finally {
    if (btn) btn.disabled = false;
  }
}

// Parse from uploaded file
export async function parseFromFile(file) {
  if (!file) return;

  showToast('正在识别二维码图片...');
  try {
    // 1. Try browser BarcodeDetector first
    if ('BarcodeDetector' in window) {
      try {
        const bmp = await createImageBitmap(file);
        const detector = new BarcodeDetector({ formats: ['qr_code'] });
        const barcodes = await detector.detect(bmp);
        if (barcodes.length > 0) {
          populateFormFromURI(barcodes[0].rawValue);
          switchAddTab('tabManual');
          stopCamera();
          showToast('二维码图片识别成功！');
          return;
        }
      } catch (e) {
        // Fallback to backend
      }
    }

    // 2. Fallback to robust backend decoding
    const res = await API.decodeImageFile(file);
    if (res.parsed) {
      populateFormFromParsed(res.parsed);
    } else if (res.text) {
      populateFormFromURI(res.text);
    }
    switchAddTab('tabManual');
    stopCamera();
    showToast('二维码图片识别成功！');
  } catch (err) {
    showToast(`识别失败: ${err.message}`, true);
  }
}
