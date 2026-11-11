<template>
  <div v-if="modelValue" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/70 backdrop-blur-sm">
    <div class="w-full max-w-xl max-h-[90vh] flex flex-col rounded-3xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
      <!-- Modal Header -->
      <div class="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between bg-slate-50/50 dark:bg-slate-900/50">
        <h3 class="font-bold text-lg text-slate-900 dark:text-white">
          {{ accountToEdit ? '编辑 2FA 令牌' : '添加 2FA 令牌' }}
        </h3>
        <button
          @click="close"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          <X :size="18" />
        </button>
      </div>

      <!-- Navigation Tabs (Only shown when adding new account) -->
      <div v-if="!accountToEdit" class="flex border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950/40 text-sm font-semibold">
        <button
          v-for="t in tabs"
          :key="t.id"
          @click="activeTab = t.id"
          class="flex-1 py-3 text-center transition-all border-b-2"
          :class="activeTab === t.id
            ? 'border-brand-500 text-brand-600 dark:text-brand-400 bg-brand-50/50 dark:bg-brand-500/10'
            : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'"
        >
          {{ t.label }}
        </button>
      </div>

      <!-- Modal Body (Scrollable) -->
      <div class="p-6 overflow-y-auto space-y-5">
        <!-- Live OTP Preview Widget (Prominently pinned at top) -->
        <LivePreview
          :secret="form.secret"
          :digits="form.digits"
          :period="form.period"
          :algorithm="form.algorithm"
          :type="form.type"
        />

        <!-- Tab 1: QR Recognition (Dropzone, Paste, Camera) -->
        <div v-if="!accountToEdit && activeTab === 'tabScan'" class="space-y-4">
          <!-- Dropzone -->
          <div
            @dragover.prevent="isDropOver = true"
            @dragleave.prevent="isDropOver = false"
            @drop.prevent="onDrop"
            class="border-2 border-dashed rounded-2xl p-6 text-center transition-all duration-200 flex flex-col items-center justify-center gap-3 cursor-pointer select-none"
            :class="isDropOver
              ? 'border-brand-500 bg-brand-500/10 scale-[1.01]'
              : 'border-slate-300 dark:border-slate-700 hover:border-brand-500/60 bg-slate-50/50 dark:bg-slate-950/30'"
            @click="triggerFileInput"
          >
            <div class="w-12 h-12 rounded-2xl bg-brand-50 dark:bg-brand-500/10 text-brand-500 flex items-center justify-center">
              <QrCode :size="24" />
            </div>
            <div>
              <div class="font-bold text-slate-900 dark:text-white text-base">
                拖拽二维码图片到此处
              </div>
              <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
                支持从其他网页直接拖入图片，或截图后按 <kbd class="px-1.5 py-0.5 rounded text-[10px] font-mono bg-slate-200 dark:bg-slate-800 text-brand-500 font-bold border border-slate-300 dark:border-slate-700">Ctrl+V</kbd> 快速粘贴
              </p>
            </div>

            <div class="flex items-center gap-2 mt-1">
              <label
                @click.stop
                class="px-3.5 py-1.5 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-semibold text-xs shadow-sm cursor-pointer transition-colors"
              >
                <span>选择本地图片</span>
                <input
                  ref="fileInputRef"
                  type="file"
                  accept="image/*"
                  class="hidden"
                  @change="onFileSelected"
                />
              </label>

              <button
                type="button"
                @click.stop="toggleCamera"
                class="px-3.5 py-1.5 rounded-xl border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-200 font-semibold text-xs transition-colors flex items-center gap-1.5"
              >
                <Camera :size="14" />
                <span>{{ isCameraActive ? '关闭摄像头' : '开启摄像头扫描' }}</span>
              </button>
            </div>
          </div>

          <!-- Collapsible Camera Viewport -->
          <div
            v-show="isCameraActive"
            class="relative w-full h-52 rounded-2xl overflow-hidden bg-black flex items-center justify-center"
          >
            <video ref="videoRef" playsinline class="w-full h-full object-cover"></video>
            <div class="absolute w-36 h-36 border-2 border-dashed border-brand-400 rounded-xl shadow-[0_0_0_9999px_rgba(0,0,0,0.5)] pointer-events-none"></div>
          </div>
        </div>

        <!-- Tab 2: Image URL Parser -->
        <div v-if="!accountToEdit && activeTab === 'tabURL'" class="space-y-3">
          <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300">
            二维码图片链接 (HTTP / HTTPS / Data URI)
          </label>
          <div class="flex gap-2">
            <input
              v-model="imageUrlInput"
              type="url"
              placeholder="例如: https://example.com/2fa-qrcode.png"
              class="flex-1 px-3.5 py-2.5 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
            />
            <button
              @click="decodeURL"
              :disabled="isDecodingURL || !imageUrlInput"
              class="px-4 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-semibold text-xs transition-colors disabled:opacity-50 flex items-center gap-1.5"
            >
              <RefreshCw v-if="isDecodingURL" :size="14" class="animate-spin" />
              <span>解析图片</span>
            </button>
          </div>

          <!-- URL Image Preview -->
          <div v-if="imageUrlPreview" class="flex justify-center p-3 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950/40">
            <img :src="imageUrlPreview" alt="二维码预览" class="max-h-40 object-contain rounded-lg shadow-sm" />
          </div>
        </div>

        <!-- Tab 3: Manual Input Form (or active when editing) -->
        <div v-show="accountToEdit || activeTab === 'tabManual'" class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                服务提供商 (Issuer)
              </label>
              <input
                v-model="form.issuer"
                type="text"
                placeholder="例如: GitHub, Google, Binance"
                class="w-full px-3.5 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                账号 / 邮箱 (Account)
              </label>
              <input
                v-model="form.account_name"
                type="text"
                placeholder="例如: user@example.com"
                class="w-full px-3.5 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              />
            </div>
          </div>

          <div>
            <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
              密钥 (Secret Key / Base32) <span class="text-rose-500">*</span>
            </label>
            <input
              v-model="form.secret"
              type="text"
              placeholder="例如: JBSWY3DPEHPK3PXP"
              required
              class="w-full px-3.5 py-2 rounded-xl text-sm font-mono uppercase bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
            />
          </div>

          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                类型
              </label>
              <select
                v-model="form.type"
                class="w-full px-3 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              >
                <option value="totp">基于时间 (TOTP)</option>
                <option value="hotp">基于计数器 (HOTP)</option>
                <option value="steam">Steam Guard</option>
              </select>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                位数
              </label>
              <select
                v-model="form.digits"
                class="w-full px-3 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              >
                <option :value="6">6 位数</option>
                <option :value="8">8 位数</option>
                <option :value="5">5 位 (Steam)</option>
              </select>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                周期 (秒)
              </label>
              <input
                v-model.number="form.period"
                type="number"
                min="10"
                max="300"
                class="w-full px-3 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              />
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                算法
              </label>
              <select
                v-model="form.algorithm"
                class="w-full px-3 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              >
                <option value="SHA1">SHA1</option>
                <option value="SHA256">SHA256</option>
                <option value="SHA512">SHA512</option>
              </select>
            </div>
          </div>

          <div>
            <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
              分类标签 (逗号分隔)
            </label>
            <input
              v-model="tagsInput"
              type="text"
              placeholder="例如: 个人, 工作, 游戏"
              class="w-full px-3.5 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
            />
          </div>

          <div>
            <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
              备注
            </label>
            <input
              v-model="form.notes"
              type="text"
              placeholder="可选备注信息"
              class="w-full px-3.5 py-2 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
            />
          </div>

          <div>
            <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5 flex items-center justify-between">
              <span>备用应急恢复码 (Recovery Codes，可选)</span>
              <span class="text-[11px] text-slate-400">支持批量粘贴多行</span>
            </label>
            <textarea
              v-model="recoveryCodesInput"
              rows="3"
              placeholder="从服务商安全设置中复制的备用恢复码 (每行一个，例如 a1b2-c3d4)"
              class="w-full px-3.5 py-2 rounded-xl text-xs font-mono bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
            ></textarea>
          </div>
        </div>

        <!-- Tab 4: Paste URI -->
        <div v-if="!accountToEdit && activeTab === 'tabURI'" class="space-y-3">
          <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300">
            粘贴完整 otpauth:// 协议链接
          </label>
          <textarea
            v-model="uriInput"
            rows="3"
            placeholder="例如: otpauth://totp/GitHub:user?secret=JBSWY3DPEHPK3PXP&issuer=GitHub"
            class="w-full px-3.5 py-2.5 rounded-xl text-sm font-mono bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
          ></textarea>
          <button
            @click="parseURI"
            :disabled="!uriInput.trim()"
            class="w-full py-2.5 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-semibold text-xs transition-colors disabled:opacity-50"
          >
            解析并填充
          </button>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="px-6 py-4 border-t border-slate-200 dark:border-slate-800 flex items-center justify-end gap-3 bg-slate-50/50 dark:bg-slate-900/50">
        <button
          type="button"
          @click="close"
          class="px-4 py-2 rounded-xl border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300 text-sm font-semibold transition-colors"
        >
          取消
        </button>
        <button
          type="button"
          @click="handleSave"
          :disabled="isSaving || !form.secret"
          class="px-5 py-2 rounded-xl bg-brand-600 hover:bg-brand-500 text-white text-sm font-semibold shadow-md shadow-brand-500/25 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
        >
          <span v-if="isSaving" class="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
          <span>{{ accountToEdit ? '保存更改' : '保存令牌' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted, onUnmounted } from 'vue';
import { X, QrCode, Camera, RefreshCw } from 'lucide-vue-next';
import LivePreview from './LivePreview.vue';
import type { Account, OTPType, OTPAlgorithm } from '../types/vault';
import { API } from '../api/client';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';

const props = defineProps<{
  modelValue: boolean;
  accountToEdit?: Account | null;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
  (e: 'saved'): void;
}>();

const { refreshAccounts, refreshTags } = useVault();
const { showToast } = useToast();

const tabs = [
  { id: 'tabScan', label: '二维码识别' },
  { id: 'tabURL', label: '图片链接解析' },
  { id: 'tabManual', label: '手动输入' },
  { id: 'tabURI', label: '粘贴 URI' },
];

const activeTab = ref('tabScan');
const isDropOver = ref(false);
const isCameraActive = ref(false);
const isDecodingURL = ref(false);
const isSaving = ref(false);

const imageUrlInput = ref('');
const imageUrlPreview = ref('');
const uriInput = ref('');
const tagsInput = ref('');
const recoveryCodesInput = ref('');
const fileInputRef = ref<HTMLInputElement | null>(null);
const videoRef = ref<HTMLVideoElement | null>(null);

let cameraStream: MediaStream | null = null;
let scanInterval: number | null = null;

const form = reactive({
  issuer: '',
  account_name: '',
  secret: '',
  type: 'totp' as OTPType,
  digits: 6,
  period: 30,
  algorithm: 'SHA1' as OTPAlgorithm,
  notes: '',
});

// Watch accountToEdit to prefill form
watch(
  () => props.accountToEdit,
  (acc) => {
    if (acc) {
      form.issuer = acc.issuer || '';
      form.account_name = acc.account_name || '';
      form.secret = acc.secret || '';
      form.type = acc.type || 'totp';
      form.digits = acc.digits || 6;
      form.period = acc.period || 30;
      form.algorithm = acc.algorithm || 'SHA1';
      form.notes = acc.notes || '';
      tagsInput.value = (acc.tags || []).join(', ');
      recoveryCodesInput.value = (acc.recovery_codes || []).map((r) => r.code).join('\n');
      activeTab.value = 'tabManual';
    } else {
      resetForm();
    }
  },
  { immediate: true }
);

function resetForm() {
  form.issuer = '';
  form.account_name = '';
  form.secret = '';
  form.type = 'totp';
  form.digits = 6;
  form.period = 30;
  form.algorithm = 'SHA1';
  form.notes = '';
  tagsInput.value = '';
  recoveryCodesInput.value = '';
  imageUrlInput.value = '';
  imageUrlPreview.value = '';
  uriInput.value = '';
  activeTab.value = 'tabScan';
  stopCamera();
}

function close() {
  stopCamera();
  emit('update:modelValue', false);
}

function triggerFileInput() {
  fileInputRef.value?.click();
}

function applyParsedData(parsed: any) {
  if (!parsed) return;
  if (parsed.issuer) form.issuer = parsed.issuer;
  if (parsed.account_name) form.account_name = parsed.account_name;
  if (parsed.secret) form.secret = parsed.secret;
  if (parsed.type) form.type = parsed.type;
  if (parsed.digits) form.digits = parsed.digits;
  if (parsed.period) form.period = parsed.period;
  if (parsed.algorithm) form.algorithm = parsed.algorithm;
  activeTab.value = 'tabManual';
}

function parseURIString(uriStr: string) {
  try {
    const u = new URL(uriStr);
    if (u.protocol !== 'otpauth:') throw new Error('无效的 otpauth:// 协议');

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
    if (q.get('issuer')) issuer = q.get('issuer')!;
    const secret = q.get('secret') || '';
    const algo = (q.get('algorithm') || 'SHA1').toUpperCase();
    const digits = parseInt(q.get('digits') || '6');
    const period = parseInt(q.get('period') || '30');

    applyParsedData({
      issuer,
      account_name: account,
      secret,
      type: type === 'hotp' ? 'hotp' : 'totp',
      digits,
      period,
      algorithm: ['SHA1', 'SHA256', 'SHA512'].includes(algo) ? algo : 'SHA1',
    });
  } catch (err: any) {
    showToast(`解析 URI 失败: ${err.message}`, true);
  }
}

function parseURI() {
  if (!uriInput.value.trim()) return;
  parseURIString(uriInput.value.trim());
}

async function onFileSelected(e: Event) {
  const target = e.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;
  await processImageFile(file);
}

async function processImageFile(file: File) {
  showToast('正在识别二维码图片...');
  try {
    const res = await API.decodeImageFile(file);
    if (res.parsed) {
      applyParsedData(res.parsed);
    } else if (res.text) {
      parseURIString(res.text);
    }
    showToast('二维码图片识别成功！');
  } catch (err: any) {
    showToast(`识别失败: ${err.message}`, true);
  }
}

async function decodeURL() {
  const url = imageUrlInput.value.trim();
  if (!url) return;
  isDecodingURL.value = true;
  showToast('正在获取并解析链接中的二维码...');
  imageUrlPreview.value = url;

  try {
    const res = await API.decodeImageURL(url);
    if (res.parsed) {
      applyParsedData(res.parsed);
    } else if (res.text) {
      parseURIString(res.text);
    }
    showToast('链接二维码解析成功！已自动填充信息');
  } catch (err: any) {
    showToast(`解析失败: ${err.message}`, true);
  } finally {
    isDecodingURL.value = false;
  }
}

function onDrop(e: DragEvent) {
  isDropOver.value = false;
  const dt = e.dataTransfer;
  if (!dt) return;

  if (dt.files && dt.files.length > 0) {
    const file = dt.files[0];
    if (file.type.startsWith('image/')) {
      processImageFile(file);
      return;
    }
  }

  const uri = dt.getData('text/uri-list');
  if (uri && (uri.startsWith('http://') || uri.startsWith('https://') || uri.startsWith('data:image/'))) {
    imageUrlInput.value = uri.trim();
    decodeURL();
    return;
  }

  const text = dt.getData('text/plain')?.trim();
  if (text) {
    if (text.startsWith('otpauth://')) {
      parseURIString(text);
    } else if (text.startsWith('http://') || text.startsWith('https://')) {
      imageUrlInput.value = text;
      decodeURL();
    }
  }
}

// Camera Control
async function toggleCamera() {
  if (isCameraActive.value) {
    stopCamera();
  } else {
    await startCamera();
  }
}

async function startCamera() {
  stopCamera();
  if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
    showToast('当前环境不支持或未授权摄像头访问', true);
    return;
  }

  try {
    cameraStream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment' },
    });
    isCameraActive.value = true;
    if (videoRef.value) {
      videoRef.value.srcObject = cameraStream;
      await videoRef.value.play();
    }

    if ('BarcodeDetector' in window) {
      const detector = new (window as any).BarcodeDetector({ formats: ['qr_code'] });
      scanInterval = window.setInterval(async () => {
        if (!videoRef.value || videoRef.value.readyState < 2) return;
        try {
          const barcodes = await detector.detect(videoRef.value);
          if (barcodes.length > 0) {
            const val = barcodes[0].rawValue;
            stopCamera();
            parseURIString(val);
            showToast('摄像头扫描成功！');
          }
        } catch {}
      }, 300);
    }
  } catch (err: any) {
    showToast('无法启动摄像头: ' + err.message, true);
    stopCamera();
  }
}

function stopCamera() {
  if (scanInterval) {
    clearInterval(scanInterval);
    scanInterval = null;
  }
  if (cameraStream) {
    cameraStream.getTracks().forEach((t) => t.stop());
    cameraStream = null;
  }
  isCameraActive.value = false;
}

async function handleSave() {
  if (!form.secret.trim()) {
    showToast('请输入或扫描 Base32 密钥', true);
    return;
  }

  const tags = tagsInput.value
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean);

  const recoveryCodes = recoveryCodesInput.value
    .split('\n')
    .map((c) => c.trim())
    .filter(Boolean)
    .map((code) => {
      const existing = props.accountToEdit?.recovery_codes?.find((r) => r.code === code);
      return { code, used: existing ? existing.used : false };
    });

  const payload: Partial<Account> = {
    issuer: form.issuer.trim(),
    account_name: form.account_name.trim(),
    secret: form.secret.trim().replace(/\s+/g, '').toUpperCase(),
    type: form.type,
    digits: Number(form.digits) || 6,
    period: Number(form.period) || 30,
    algorithm: form.algorithm,
    tags,
    notes: form.notes.trim(),
    recovery_codes: recoveryCodes,
  };

  isSaving.value = true;
  try {
    if (props.accountToEdit) {
      await API.updateAccount(props.accountToEdit.id, payload);
      showToast('令牌已成功更新');
    } else {
      await API.createAccount(payload);
      showToast('令牌已成功添加');
    }
    await refreshAccounts();
    await refreshTags();
    emit('saved');
    close();
  } catch (err: any) {
    showToast(`保存失败: ${err.message}`, true);
  } finally {
    isSaving.value = false;
  }
}

// Global Paste Event Listener inside Modal
function handleGlobalPaste(e: ClipboardEvent) {
  if (!props.modelValue) return;
  const items = e.clipboardData?.items;
  if (!items) return;

  for (const item of items) {
    if (item.type.indexOf('image') !== -1) {
      const file = item.getAsFile();
      if (file) {
        e.preventDefault();
        showToast('已检测到剪贴板截图，正在解析二维码...');
        processImageFile(file);
        return;
      }
    }
  }
}

onMounted(() => {
  window.addEventListener('paste', handleGlobalPaste);
});

onUnmounted(() => {
  stopCamera();
  window.removeEventListener('paste', handleGlobalPaste);
});
</script>
