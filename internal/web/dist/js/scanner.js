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
export async function toggleCamera() {
  const vp = document.getElementById('scannerViewport');
  const txt = document.getElementById('toggleCameraText');
  if (!vp) return;

  if (vp.style.display === 'none') {
    vp.style.display = 'flex';
    if (txt) txt.textContent = '关闭摄像头扫描';
    await startCamera();
  } else {
    vp.style.display = 'none';
    if (txt) txt.textContent = '开启摄像头扫描';
    stopCamera();
  }
}

export async function startCamera() {
  stopCamera();
  const video = document.getElementById('scannerVideo');
  if (!video || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
    showToast('当前环境不支持或未授权摄像头访问', true);
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
            const vp = document.getElementById('scannerViewport');
            const txt = document.getElementById('toggleCameraText');
            if (vp) vp.style.display = 'none';
            if (txt) txt.textContent = '开启摄像头扫描';
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
    showToast('无法启动摄像头: ' + err.message, true);
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

  // Always reset camera viewport when switching tabs
  const vp = document.getElementById('scannerViewport');
  const txt = document.getElementById('toggleCameraText');
  if (vp) vp.style.display = 'none';
  if (txt) txt.textContent = '开启摄像头扫描';
  stopCamera();
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

export function openAddModalForImport() {
  const addModal = document.getElementById('addModal');
  if (addModal) {
    state.editingId = null;
    document.getElementById('modalTitle').textContent = '添加 2FA 令牌';
    document.getElementById('accountForm').reset();
    document.getElementById('editAccountId').value = '';
    addModal.classList.add('active');
  }
}

export function setupDragDropAndPaste() {
  // 1. Dropzone click to select file
  const dropzone = document.getElementById('qrDropzone');
  const fileInput = document.getElementById('qrFileInput');
  if (dropzone && fileInput) {
    dropzone.addEventListener('click', (e) => {
      if (e.target.closest('#toggleCameraBtn') || e.target.closest('label')) return;
      fileInput.click();
    });
  }

  // 2. Drag & Drop on the in-modal QR Dropzone
  if (dropzone) {
    dropzone.addEventListener('dragover', (e) => {
      e.preventDefault();
      e.stopPropagation();
      dropzone.classList.add('dragover');
    });
    dropzone.addEventListener('dragleave', (e) => {
      e.preventDefault();
      e.stopPropagation();
      dropzone.classList.remove('dragover');
    });
    dropzone.addEventListener('drop', (e) => {
      e.preventDefault();
      e.stopPropagation();
      dropzone.classList.remove('dragover');
      handleDroppedData(e.dataTransfer);
    });
  }

  // 3. Global Drag & Drop over Window
  let dragCounter = 0;
  window.addEventListener('dragenter', (e) => {
    e.preventDefault();
    dragCounter++;
    if (e.dataTransfer && e.dataTransfer.types) {
      document.body.classList.add('is-dragging');
    }
  });

  window.addEventListener('dragover', (e) => {
    e.preventDefault();
  });

  window.addEventListener('dragleave', (e) => {
    e.preventDefault();
    dragCounter--;
    if (dragCounter <= 0) {
      dragCounter = 0;
      document.body.classList.remove('is-dragging');
    }
  });

  window.addEventListener('drop', (e) => {
    e.preventDefault();
    dragCounter = 0;
    document.body.classList.remove('is-dragging');
    handleDroppedData(e.dataTransfer);
  });

  // 4. Global Clipboard Paste (Ctrl+V)
  window.addEventListener('paste', async (e) => {
    const activeEl = document.activeElement;
    const isTextInput = activeEl && (activeEl.tagName === 'INPUT' || activeEl.tagName === 'TEXTAREA');

    const items = e.clipboardData?.items;
    if (items) {
      for (const item of items) {
        if (item.type.indexOf('image') !== -1) {
          const file = item.getAsFile();
          if (file) {
            e.preventDefault();
            openAddModalForImport();
            showToast('已捕获剪贴板截图，正在解析二维码...');
            await parseFromFile(file);
            return;
          }
        }
      }
    }

    // If not typing in an input, check if text is an otpauth:// URI or image URL
    if (!isTextInput) {
      const text = e.clipboardData?.getData('text/plain')?.trim();
      if (text) {
        if (text.startsWith('otpauth://')) {
          e.preventDefault();
          openAddModalForImport();
          populateFormFromURI(text);
          switchAddTab('tabManual');
          showToast('已识别剪贴板中的 2FA URI 链接');
        } else if (text.startsWith('http://') || text.startsWith('https://')) {
          e.preventDefault();
          openAddModalForImport();
          showToast('正在从剪贴板链接解析二维码...');
          parseFromImageURL(text);
        }
      }
    }
  });
}

async function handleDroppedData(dataTransfer) {
  if (!dataTransfer) return;

  // Case A: Dropped file (from desktop, file explorer, or browser drag)
  if (dataTransfer.files && dataTransfer.files.length > 0) {
    const file = dataTransfer.files[0];
    if (file.type.startsWith('image/')) {
      openAddModalForImport();
      showToast('已接收拖入的二维码图片，正在识别...');
      await parseFromFile(file);
      return;
    }
  }

  // Case B: Dropped image URL from other web page (URI-list)
  const uri = dataTransfer.getData('text/uri-list');
  if (uri && (uri.startsWith('http://') || uri.startsWith('https://') || uri.startsWith('data:image/'))) {
    openAddModalForImport();
    showToast('已接收拖拽的图片链接，正在解析...');
    await parseFromImageURL(uri.trim());
    return;
  }

  // Case C: Dropped HTML with <img> tag
  const html = dataTransfer.getData('text/html');
  if (html) {
    const match = html.match(/<img[^>]+src=["']([^"']+)["']/i);
    if (match && match[1]) {
      openAddModalForImport();
      showToast('已识别拖拽的网页图片，正在解析...');
      await parseFromImageURL(match[1]);
      return;
    }
  }

  // Case D: Dropped Plain Text (URI or link)
  const text = dataTransfer.getData('text/plain')?.trim();
  if (text) {
    if (text.startsWith('otpauth://')) {
      openAddModalForImport();
      populateFormFromURI(text);
      switchAddTab('tabManual');
      showToast('已识别拖入的 2FA URI 链接');
      return;
    } else if (text.startsWith('http://') || text.startsWith('https://')) {
      openAddModalForImport();
      showToast('已接收拖入的图片链接，正在解析...');
      await parseFromImageURL(text);
      return;
    }
  }
}
