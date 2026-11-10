<template>
  <div v-if="modelValue" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/70 backdrop-blur-sm">
    <div class="w-full max-w-md rounded-3xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-2xl overflow-hidden p-6 text-center animate-in fade-in zoom-in-95 duration-150">
      <!-- Header -->
      <div class="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800">
        <div class="flex items-center gap-2">
          <Wifi :size="18" class="text-brand-500" />
          <h3 class="font-bold text-base text-slate-900 dark:text-white">PC ↔ 手机 局域网面对面同步</h3>
        </div>
        <button
          @click="$emit('update:modelValue', false)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          <X :size="18" />
        </button>
      </div>

      <div class="py-5 space-y-4">
        <p class="text-xs text-slate-500 dark:text-slate-400 text-left">
          无需任何第三方云盘，当手机和电脑在<strong>同一个 Wi-Fi</strong> 下时，手机打开自带相机或浏览器扫描下方配对码，即可将加密保险库安全传输至手机。
        </p>

        <!-- QR Code Display Box -->
        <div class="flex flex-col items-center justify-center p-4 rounded-2xl bg-white border border-slate-200 shadow-inner">
          <div v-if="loading" class="w-48 h-48 flex items-center justify-center text-slate-400">
            <RefreshCw class="animate-spin" :size="32" />
          </div>
          <img
            v-else-if="qrCodeData"
            :src="qrCodeData"
            alt="局域网配对二维码"
            class="w-48 h-48 object-contain select-none"
          />
          <div v-else class="w-48 h-48 flex items-center justify-center text-xs text-rose-500">
            生成配对二维码失败
          </div>

          <div v-if="pairingInfo" class="mt-2 text-[11px] text-slate-500 font-mono text-center">
            有效时间剩余: <span class="font-bold text-brand-600">{{ remainingTime }}s</span>
          </div>
        </div>

        <!-- Pairing URL -->
        <div v-if="pairingInfo" class="text-left space-y-1.5">
          <label class="block text-[11px] font-semibold text-slate-600 dark:text-slate-400">
            手机浏览器也可直接打开此网址同步：
          </label>
          <div class="flex items-center gap-2">
            <input
              type="text"
              readonly
              :value="pairingInfo.pairing_url"
              class="w-full px-3 py-1.5 rounded-lg text-xs font-mono bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 text-slate-700 dark:text-slate-300 outline-none"
            />
            <button
              @click="copyPairingURL"
              class="px-2.5 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 text-slate-700 dark:text-slate-300 text-xs font-semibold flex-shrink-0 transition-colors"
            >
              复制
            </button>
          </div>
        </div>

        <button
          type="button"
          @click="generatePairing"
          class="w-full py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-200 text-xs font-semibold transition-colors flex items-center justify-center gap-1.5"
        >
          <RefreshCw :size="14" />
          <span>刷新并重新生成配对码</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue';
import { X, Wifi, RefreshCw } from 'lucide-vue-next';
import { API } from '../api/client';
import { useToast } from '../composables/useToast';
import type { SyncPairInitResponse } from '../types/vault';

const props = defineProps<{
  modelValue: boolean;
}>();

defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
}>();

const { showToast } = useToast();

const loading = ref(false);
const pairingInfo = ref<SyncPairInitResponse | null>(null);
const qrCodeData = ref('');
const remainingTime = ref(300);
let timer: number | null = null;

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      generatePairing();
    } else {
      clearTimer();
    }
  }
);

function clearTimer() {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
}

async function generatePairing() {
  clearTimer();
  loading.value = true;
  try {
    const res = await API.initSyncPair();
    pairingInfo.value = res;
    remainingTime.value = res.expires_in;

    // Generate QR code for the pairing URL
    qrCodeData.value = `/api/qr?data=${encodeURIComponent(res.pairing_url)}&size=300`;

    timer = window.setInterval(() => {
      remainingTime.value--;
      if (remainingTime.value <= 0) {
        clearTimer();
        pairingInfo.value = null;
        qrCodeData.value = '';
      }
    }, 1000);
  } catch (err: any) {
    showToast(`生成配对码失败: ${err.message}`, true);
  } finally {
    loading.value = false;
  }
}

function copyPairingURL() {
  if (!pairingInfo.value) return;
  navigator.clipboard.writeText(pairingInfo.value.pairing_url).then(() => {
    showToast('已复制局域网同步链接');
  });
}

onUnmounted(() => {
  clearTimer();
});
</script>
