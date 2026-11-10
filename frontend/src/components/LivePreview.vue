<template>
  <div
    class="group relative overflow-hidden rounded-2xl border p-4 transition-all duration-200 cursor-pointer select-none"
    :class="isValidSecret
      ? 'border-brand-500/30 bg-brand-500/5 hover:border-brand-500/50 hover:bg-brand-500/10'
      : 'border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-900/50 opacity-80'"
    @click="copyCode"
    title="点击直接复制预览验证码"
  >
    <div class="flex items-center justify-between gap-4">
      <div class="flex flex-col gap-1 min-w-0">
        <div class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-brand-600 dark:text-brand-400">
          <Sparkles :size="14" />
          <span>实时动态验证码预览 (点击复制)</span>
        </div>

        <div class="font-mono text-3xl font-bold tracking-wider text-slate-900 dark:text-white my-0.5">
          {{ displayCode }}
        </div>

        <div class="text-xs font-medium text-slate-500 dark:text-slate-400 truncate">
          {{ statusHint }}
        </div>
      </div>

      <div class="flex-shrink-0 flex items-center justify-center">
        <div class="relative w-12 h-12 flex items-center justify-center">
          <svg class="w-12 h-12 -rotate-90 transform" viewBox="0 0 38 38">
            <circle
              class="text-slate-200 dark:text-slate-800"
              stroke-width="3"
              stroke="currentColor"
              fill="transparent"
              cx="19"
              cy="19"
              r="16"
            />
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
          <span class="absolute font-mono text-xs font-bold text-slate-700 dark:text-slate-200">
            {{ remainingSeconds !== null ? `${remainingSeconds}s` : '--' }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { Sparkles } from 'lucide-vue-next';
import { API } from '../api/client';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';

const props = withDefaults(
  defineProps<{
    secret: string;
    digits?: number;
    period?: number;
    algorithm?: string;
    type?: string;
  }>(),
  {
    digits: 6,
    period: 30,
    algorithm: 'SHA1',
    type: 'totp',
  }
);

const { nowSeconds } = useVault();
const { showToast } = useToast();

const previewCode = ref('------');
const isValidSecret = ref(false);
const statusHint = ref('输入或解析密钥后即时展示');
let lastPreviewPeriod = -1;

// Clean secret string
const cleanSecret = computed(() => props.secret.replace(/[\s\-_]/g, '').toUpperCase());

// Calculate remaining seconds and progress reactively
const remainingSeconds = computed(() => {
  if (!isValidSecret.value || props.type === 'hotp') return null;
  const p = props.period || 30;
  return p - (nowSeconds.value % p);
});

const progress = computed(() => {
  if (!isValidSecret.value || props.type === 'hotp') return 0;
  const p = props.period || 30;
  return (nowSeconds.value % p) / p;
});

const dashOffset = computed(() => {
  return 100.53 * progress.value;
});

const timerColor = computed(() => {
  const rem = remainingSeconds.value;
  if (rem === null) return 'currentColor';
  if (rem <= 5) return '#f43f5e'; // rose-500
  if (rem <= 10) return '#f59e0b'; // amber-500
  return '#10b981'; // emerald-500
});

const displayCode = computed(() => {
  return previewCode.value;
});

async function updatePreview() {
  const sec = cleanSecret.value;
  if (sec.length < 4) {
    previewCode.value = '------';
    isValidSecret.value = false;
    statusHint.value = '输入或解析密钥后即时展示';
    return;
  }

  try {
    const res = await API.previewOTP({
      secret: sec,
      digits: props.digits,
      period: props.period,
      algorithm: props.algorithm,
      type: props.type,
    });

    if (res && res.valid) {
      previewCode.value = res.formatted_code || res.code || '------';
      isValidSecret.value = true;
      statusHint.value = '已验证密钥有效 · 实时计算中 (点击复制)';
    } else {
      previewCode.value = '------';
      isValidSecret.value = false;
      statusHint.value = res.error || '密钥格式有误 (需为标准 Base32 编码)';
    }
  } catch (err: any) {
    previewCode.value = '------';
    isValidSecret.value = false;
    statusHint.value = '计算错误: ' + err.message;
  }
}

// Watch secret, digits, period, algorithm, type changes
watch(
  [() => props.secret, () => props.digits, () => props.period, () => props.algorithm, () => props.type],
  () => {
    updatePreview();
  },
  { immediate: true }
);

// Watch nowSeconds to trigger code refresh when period boundary rolls over
watch(nowSeconds, (now) => {
  if (!isValidSecret.value || props.type === 'hotp') return;
  const p = props.period || 30;
  const currentPeriod = Math.floor(now / p);
  if (lastPreviewPeriod !== -1 && currentPeriod !== lastPreviewPeriod) {
    updatePreview();
  }
  lastPreviewPeriod = currentPeriod;
});

function copyCode() {
  const code = previewCode.value.replace(/\s+/g, '');
  if (!isValidSecret.value || code === '------') return;

  navigator.clipboard.writeText(code).then(() => {
    showToast(`已复制预览验证码: ${code}`);
  });
}
</script>
