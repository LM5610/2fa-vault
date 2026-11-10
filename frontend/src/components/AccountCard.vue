<template>
  <div
    class="group relative flex flex-col justify-between rounded-2xl border border-slate-200/80 dark:border-slate-800/80 bg-white dark:bg-slate-900/90 p-5 shadow-sm hover:shadow-md hover:border-brand-500/40 dark:hover:border-brand-500/40 transition-all duration-200"
  >
    <!-- Card Header -->
    <div class="flex items-start justify-between gap-3 mb-4">
      <div class="flex items-center gap-3 min-w-0">
        <!-- Account Icon Avatar -->
        <div class="w-10 h-10 rounded-xl bg-brand-50 dark:bg-brand-500/10 text-brand-600 dark:text-brand-400 flex items-center justify-center font-bold text-lg flex-shrink-0 border border-brand-500/20">
          {{ avatarChar }}
        </div>

        <div class="min-w-0">
          <h3 class="font-bold text-slate-900 dark:text-white truncate text-base leading-tight">
            {{ account.issuer || account.account_name || '未命名服务' }}
          </h3>
          <p class="text-xs text-slate-500 dark:text-slate-400 truncate mt-0.5">
            {{ account.account_name || '无账号名' }}
          </p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-1 opacity-80 group-hover:opacity-100 transition-opacity">
        <button
          @click="$emit('view-qr', account)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-brand-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          title="查看二维码"
        >
          <QrCode :size="16" />
        </button>
        <button
          @click="$emit('edit', account)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-brand-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          title="编辑"
        >
          <Edit3 :size="16" />
        </button>
        <button
          @click="$emit('delete', account)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
          title="删除"
        >
          <Trash2 :size="16" />
        </button>
      </div>
    </div>

    <!-- Code Display Box -->
    <div
      @click="copyCode"
      class="group/code relative flex items-center justify-between rounded-xl border border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-950/60 p-3 px-4 cursor-pointer hover:border-brand-500/30 hover:bg-brand-500/5 transition-all select-none"
    >
      <div class="flex flex-col">
        <span class="font-mono text-2xl font-bold tracking-wider text-slate-900 dark:text-white">
          {{ formattedCode }}
        </span>
        <span v-if="copied" class="text-[11px] font-semibold text-emerald-500 animate-pulse">
          ✓ 已复制到剪贴板
        </span>
        <span v-else class="text-[11px] text-slate-400 group-hover/code:text-brand-500 transition-colors">
          点击快速复制
        </span>
      </div>

      <!-- Timer or HOTP Trigger -->
      <div class="flex items-center gap-2">
        <template v-if="account.type === 'hotp'">
          <button
            @click.stop="incrementHOTP"
            :disabled="isIncrementing"
            class="px-3 py-1.5 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-medium text-xs shadow-sm transition-colors flex items-center gap-1.5 disabled:opacity-50"
          >
            <RefreshCw :size="12" :class="{ 'animate-spin': isIncrementing }" />
            <span>生成</span>
          </button>
        </template>
        <template v-else>
          <div class="relative w-10 h-10 flex items-center justify-center flex-shrink-0">
            <svg class="w-10 h-10 -rotate-90 transform" viewBox="0 0 38 38">
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
            <span class="absolute font-mono text-[11px] font-bold text-slate-600 dark:text-slate-300">
              {{ remainingSeconds }}s
            </span>
          </div>
        </template>
      </div>
    </div>

    <!-- Tags and Details Footer -->
    <div v-if="account.tags && account.tags.length > 0" class="flex flex-wrap gap-1.5 mt-3">
      <span
        v-for="tag in account.tags"
        :key="tag"
        class="px-2 py-0.5 rounded-md bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 text-[11px] font-medium"
      >
        #{{ tag }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { QrCode, Edit3, Trash2, RefreshCw } from 'lucide-vue-next';
import type { Account } from '../types/vault';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';
import { API } from '../api/client';

const props = defineProps<{
  account: Account;
}>();

const emit = defineEmits<{
  (e: 'view-qr', account: Account): void;
  (e: 'edit', account: Account): void;
  (e: 'delete', account: Account): void;
}>();

const { nowSeconds } = useVault();
const { showToast } = useToast();

const copied = ref(false);
const isIncrementing = ref(false);

const avatarChar = computed(() => {
  const name = props.account.issuer || props.account.account_name || '2';
  return name.charAt(0).toUpperCase();
});

const formattedCode = computed(() => {
  const c = props.account.code || '------';
  if (c.length === 6) return `${c.slice(0, 3)} ${c.slice(3)}`;
  if (c.length === 8) return `${c.slice(0, 4)} ${c.slice(4)}`;
  return c;
});

const remainingSeconds = computed(() => {
  if (props.account.type === 'hotp') return 0;
  const p = props.account.period || 30;
  return p - (nowSeconds.value % p);
});

const dashOffset = computed(() => {
  if (props.account.type === 'hotp') return 0;
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

function copyCode() {
  const raw = props.account.code || '';
  if (!raw || raw === '------') return;

  navigator.clipboard.writeText(raw).then(() => {
    copied.value = true;
    showToast(`已复制验证码: ${raw}`);
    setTimeout(() => {
      copied.value = false;
    }, 2000);
  });
}

async function incrementHOTP() {
  if (isIncrementing.value) return;
  isIncrementing.value = true;
  try {
    const res = await API.incrementHOTP(props.account.id);
    props.account.code = res.code;
    props.account.counter = res.counter;
    showToast(`已生成新验证码: ${res.code}`);
  } catch (err: any) {
    showToast(`HOTP 生成失败: ${err.message}`, true);
  } finally {
    isIncrementing.value = false;
  }
}
</script>
