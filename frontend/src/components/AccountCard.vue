<template>
  <div
    class="group relative flex flex-col justify-between rounded-2xl border bg-white dark:bg-slate-900/90 p-5 shadow-sm hover:shadow-md transition-all duration-200"
    :class="account.pinned
      ? 'border-brand-500/50 shadow-brand-500/5 ring-1 ring-brand-500/20'
      : 'border-slate-200/80 dark:border-slate-800/80 hover:border-brand-500/40'"
  >
    <!-- Card Header -->
    <div class="flex items-start justify-between gap-3 mb-4">
      <div class="flex items-center gap-3 min-w-0">
        <!-- Brand Logo / Avatar -->
        <div
          class="w-11 h-11 rounded-2xl flex items-center justify-center font-bold text-base flex-shrink-0 shadow-sm overflow-hidden p-2 border border-slate-200/60 dark:border-slate-800/80"
          :style="{ backgroundColor: brandInfo.bg || 'rgba(99, 102, 241, 0.1)', color: brandInfo.color || '#6366f1' }"
        >
          <div v-if="brandInfo.svg" v-html="brandInfo.svg" class="w-full h-full flex items-center justify-center"></div>
          <img
            v-else-if="brandInfo.apiLogoUrl && !imgFailed"
            :src="brandInfo.apiLogoUrl"
            :alt="account.issuer"
            class="w-full h-full object-contain"
            @error="imgFailed = true"
          />
          <span v-else class="text-brand-600 dark:text-brand-400 font-extrabold text-lg">
            {{ avatarChar }}
          </span>
        </div>

        <div class="min-w-0">
          <div class="flex items-center gap-1.5">
            <h3 class="font-bold text-slate-900 dark:text-white truncate text-base leading-tight">
              {{ displayTitle }}
            </h3>
            <span v-if="account.pinned" class="text-brand-500" title="已置顶">
              <Pin :size="13" class="fill-brand-500" />
            </span>
          </div>
          <p class="text-xs text-slate-500 dark:text-slate-400 truncate mt-0.5">
            {{ account.account_name || '无账号名' }}
          </p>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-1">
        <!-- Pin Toggle Button -->
        <button
          @click="togglePin"
          class="p-1.5 rounded-lg transition-colors"
          :class="account.pinned
            ? 'text-brand-500 bg-brand-50 dark:bg-brand-500/10'
            : 'text-slate-400 hover:text-brand-500 hover:bg-slate-100 dark:hover:bg-slate-800'"
          :title="account.pinned ? '取消置顶' : '置顶此账号'"
        >
          <Pin :size="15" :class="{ 'fill-brand-500': account.pinned }" />
        </button>

        <button
          @click="$emit('view-qr', account)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-brand-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          title="查看二维码"
        >
          <QrCode :size="15" />
        </button>

        <button
          @click="$emit('edit', account)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-brand-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          title="编辑"
        >
          <Edit3 :size="15" />
        </button>

        <button
          @click="$emit('delete', account)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
          title="删除"
        >
          <Trash2 :size="15" />
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

    <!-- Emergency Recovery Codes Accordion Drawer -->
    <div v-if="hasRecoveryCodes" class="mt-3 pt-3 border-t border-slate-100 dark:border-slate-800/80">
      <button
        @click="showRecovery = !showRecovery"
        type="button"
        class="w-full flex items-center justify-between text-xs font-semibold text-slate-600 dark:text-slate-300 hover:text-brand-500 transition-colors py-1"
      >
        <div class="flex items-center gap-1.5">
          <KeyRound :size="14" class="text-amber-500" />
          <span>备用应急恢复码</span>
          <span class="px-1.5 py-0.2 rounded text-[10px] font-medium bg-amber-50 dark:bg-amber-950/40 text-amber-600 dark:text-amber-400 border border-amber-500/20">
            剩余 {{ unusedCount }}/{{ totalCount }} 可用
          </span>
        </div>
        <ChevronDown :size="14" class="transition-transform duration-200" :class="{ 'rotate-180': showRecovery }" />
      </button>

      <!-- Expandable Recovery Codes List -->
      <div v-show="showRecovery" class="mt-2.5 space-y-1.5 bg-slate-50 dark:bg-slate-950/40 p-3 rounded-xl border border-slate-200/60 dark:border-slate-800/60">
        <div class="text-[11px] text-slate-400 mb-2 flex items-center justify-between">
          <span>用过请勾选标记为已废弃</span>
          <span>点击代码直接复制</span>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
          <div
            v-for="(rc, idx) in account.recovery_codes"
            :key="idx"
            class="flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg border text-xs font-mono transition-all"
            :class="rc.used
              ? 'bg-slate-100/60 dark:bg-slate-900/40 border-transparent text-slate-400 line-through'
              : 'bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 text-slate-800 dark:text-slate-200 hover:border-brand-500/40 cursor-pointer'"
            @click="!rc.used && copySingleRecovery(rc.code)"
          >
            <div class="flex items-center gap-2 truncate">
              <input
                type="checkbox"
                :checked="rc.used"
                @click.stop="toggleRecovery(idx)"
                class="w-3.5 h-3.5 rounded border-slate-300 dark:border-slate-700 text-brand-600 focus:ring-0 cursor-pointer"
              />
              <span class="truncate">{{ rc.code }}</span>
            </div>
            <Copy v-if="!rc.used" :size="12" class="text-slate-400 hover:text-brand-500 flex-shrink-0" />
          </div>
        </div>
      </div>
    </div>

    <!-- Tags Strip -->
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
import { QrCode, Edit3, Trash2, RefreshCw, Pin, KeyRound, ChevronDown, Copy } from 'lucide-vue-next';
import type { Account } from '../types/vault';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';
import { API } from '../api/client';
import { getBrandLogo } from '../utils/brandIcons';

const props = defineProps<{
  account: Account;
}>();

const emit = defineEmits<{
  (e: 'view-qr', account: Account): void;
  (e: 'edit', account: Account): void;
  (e: 'delete', account: Account): void;
}>();

const { nowSeconds, refreshAccounts } = useVault();
const { showToast } = useToast();

const copied = ref(false);
const isIncrementing = ref(false);
const showRecovery = ref(false);
const imgFailed = ref(false);

const brandInfo = computed(() => getBrandLogo(props.account.issuer, props.account.account_name));

const displayTitle = computed(() => {
  if (props.account.issuer && props.account.issuer.trim() && props.account.issuer !== props.account.account_name && !props.account.issuer.includes('@')) {
    return props.account.issuer;
  }
  if (brandInfo.value.name) {
    return brandInfo.value.name;
  }
  return props.account.issuer || props.account.account_name || '未命名服务';
});

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

// Recovery codes stats
const hasRecoveryCodes = computed(() => {
  return props.account.recovery_codes && props.account.recovery_codes.length > 0;
});

const totalCount = computed(() => props.account.recovery_codes?.length || 0);
const unusedCount = computed(() => {
  return (props.account.recovery_codes || []).filter((r) => !r.used).length;
});

async function togglePin() {
  try {
    const updated = await API.togglePin(props.account.id);
    props.account.pinned = updated.pinned;
    showToast(updated.pinned ? '已置顶' : '已取消置顶');
    await refreshAccounts();
  } catch (err: any) {
    showToast(`置顶失败: ${err.message}`, true);
  }
}

async function toggleRecovery(index: number) {
  try {
    const updated = await API.toggleRecoveryCode(props.account.id, index);
    props.account.recovery_codes = updated.recovery_codes;
  } catch (err: any) {
    showToast(`切换恢复码状态失败: ${err.message}`, true);
  }
}

function copySingleRecovery(code: string) {
  navigator.clipboard.writeText(code).then(() => {
    showToast(`已复制恢复码: ${code}`);
  });
}

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
