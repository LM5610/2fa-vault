<template>
  <div v-if="account" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/70 backdrop-blur-sm">
    <div class="w-full max-w-sm rounded-3xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-2xl overflow-hidden p-6 text-center animate-in fade-in zoom-in-95 duration-150">
      <!-- Header -->
      <div class="flex items-center justify-between mb-4">
        <h3 class="font-bold text-base text-slate-900 dark:text-white truncate">
          {{ account.issuer || '2FA 二维码' }}
        </h3>
        <button
          @click="$emit('close')"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          <X :size="18" />
        </button>
      </div>

      <!-- QR Code Image Box -->
      <div class="p-4 bg-white rounded-2xl shadow-inner border border-slate-200 inline-block mb-4">
        <div v-if="loading" class="w-48 h-48 flex items-center justify-center text-slate-400">
          <RefreshCw class="animate-spin" :size="32" />
        </div>
        <img
          v-else-if="qrData"
          :src="qrData"
          alt="2FA QR Code"
          class="w-48 h-48 object-contain select-none"
        />
        <div v-else class="w-48 h-48 flex items-center justify-center text-xs text-rose-500">
          二维码生成失败
        </div>
      </div>

      <!-- Live Code Widget inside QR Modal -->
      <div class="rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950 p-3 mb-4 flex items-center justify-between">
        <div class="text-left">
          <span class="text-[10px] uppercase font-bold text-slate-400 tracking-wider">实时验证码</span>
          <div class="font-mono text-xl font-bold text-slate-900 dark:text-white tracking-wider">
            {{ formattedCode }}
          </div>
        </div>

        <div class="relative w-8 h-8 flex items-center justify-center flex-shrink-0">
          <svg class="w-8 h-8 -rotate-90 transform" viewBox="0 0 38 38">
            <circle class="text-slate-200 dark:text-slate-800" stroke-width="3" stroke="currentColor" fill="transparent" cx="19" cy="19" r="16" />
            <circle
              class="transition-all duration-300"
              stroke-width="3"
              :stroke-dasharray="100.53"
              :stroke-dashoffset="dashOffset"
              stroke-linecap="round"
              :stroke="timerColor"
              fill="transparent"
              cx="19"
              cy="19"
              r="16"
            />
          </svg>
          <span class="absolute font-mono text-[9px] font-bold text-slate-600 dark:text-slate-300">
            {{ remainingSeconds }}s
          </span>
        </div>
      </div>

      <!-- Copy URI Button -->
      <button
        @click="copyURI"
        class="w-full py-2.5 rounded-xl border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-200 text-xs font-semibold transition-colors flex items-center justify-center gap-1.5"
      >
        <Copy :size="14" />
        <span>复制 otpauth:// 链接</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { X, RefreshCw, Copy } from 'lucide-vue-next';
import type { Account } from '../types/vault';
import { API } from '../api/client';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';

const props = defineProps<{
  account: Account | null;
}>();

defineEmits<{
  (e: 'close'): void;
}>();

const { nowSeconds } = useVault();
const { showToast } = useToast();

const loading = ref(false);
const qrData = ref('');
const otpauthUri = ref('');

watch(
  () => props.account,
  async (acc) => {
    if (!acc) return;
    loading.value = true;
    try {
      const res = await API.getAccountQR(acc.id);
      qrData.value = res.qr_data;
      otpauthUri.value = res.otpauth_uri;
    } catch (err: any) {
      showToast(`获取二维码失败: ${err.message}`, true);
    } finally {
      loading.value = false;
    }
  },
  { immediate: true }
);

const formattedCode = computed(() => {
  const c = props.account?.code || '------';
  if (c.length === 6) return `${c.slice(0, 3)} ${c.slice(3)}`;
  if (c.length === 8) return `${c.slice(0, 4)} ${c.slice(4)}`;
  return c;
});

const remainingSeconds = computed(() => {
  if (!props.account || props.account.type === 'hotp') return 0;
  const p = props.account.period || 30;
  return p - (nowSeconds.value % p);
});

const dashOffset = computed(() => {
  if (!props.account || props.account.type === 'hotp') return 0;
  const p = props.account.period || 30;
  const prog = (nowSeconds.value % p) / p;
  return 100.53 * prog;
});

const timerColor = computed(() => {
  const rem = remainingSeconds.value;
  if (rem <= 5) return '#f43f5e';
  if (rem <= 10) return '#f59e0b';
  return '#10b981';
});

function copyURI() {
  if (!otpauthUri.value) return;
  navigator.clipboard.writeText(otpauthUri.value).then(() => {
    showToast('已复制 otpauth:// 协议链接');
  });
}
</script>
