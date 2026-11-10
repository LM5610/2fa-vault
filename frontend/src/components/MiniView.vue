<template>
  <div class="h-screen w-screen flex flex-col bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 select-none overflow-hidden">
    <!-- Mini Header -->
    <header class="flex-shrink-0 px-3 py-2.5 border-b border-slate-200 dark:border-slate-800 bg-white/95 dark:bg-slate-900/95 backdrop-blur flex items-center justify-between gap-2">
      <div class="flex items-center gap-2 min-w-0">
        <div class="w-6 h-6 rounded-lg bg-gradient-to-tr from-brand-600 to-indigo-500 text-white flex items-center justify-center flex-shrink-0 shadow-sm">
          <ShieldCheck :size="14" />
        </div>
        <span class="font-bold text-xs text-slate-900 dark:text-white tracking-tight truncate">2FA Vault</span>
      </div>

      <!-- Compact Search Box -->
      <div class="flex-1 max-w-[180px] relative">
        <Search class="absolute left-2 top-1/2 -translate-y-1/2 text-slate-400" :size="12" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索令牌..."
          class="w-full pl-6 pr-2 py-1 rounded-lg text-xs bg-slate-100 dark:bg-slate-800 border border-transparent focus:border-brand-500 text-slate-900 dark:text-white placeholder-slate-400 outline-none transition-all"
        />
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-1 flex-shrink-0">
        <!-- Switch to Fullscreen / Normal View -->
        <button
          @click="$emit('exit-mini')"
          class="p-1 rounded-md text-slate-500 hover:text-brand-500 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          title="切换至完整视图"
        >
          <Maximize2 :size="14" />
        </button>

        <!-- Theme Toggle -->
        <button
          @click="toggleTheme"
          class="p-1 rounded-md text-slate-500 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          :title="theme === 'dark' ? '浅色模式' : '暗色模式'"
        >
          <Sun v-if="theme === 'dark'" :size="14" />
          <Moon v-else :size="14" />
        </button>

        <!-- Lock Button -->
        <button
          @click="lock"
          class="p-1 rounded-md text-slate-500 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-colors"
          title="锁定"
        >
          <Lock :size="14" />
        </button>
      </div>
    </header>

    <!-- Mini Account List -->
    <div class="flex-1 overflow-y-auto px-2 py-2 space-y-1.5 custom-scrollbar">
      <div
        v-for="acc in filteredAccounts"
        :key="acc.id"
        @click="copyCode(acc)"
        class="group relative flex items-center justify-between p-2 rounded-xl border border-slate-200/80 dark:border-slate-800/80 bg-white dark:bg-slate-900/90 hover:border-brand-500/50 hover:bg-brand-50/30 dark:hover:bg-brand-950/20 cursor-pointer shadow-sm transition-all"
      >
        <!-- Left: Logo & Names -->
        <div class="flex items-center gap-2.5 min-w-0 pr-2">
          <!-- Logo -->
          <div
            class="w-8 h-8 rounded-xl flex items-center justify-center font-bold text-xs flex-shrink-0 shadow-xs overflow-hidden border border-slate-200/60 dark:border-slate-800/80"
            :class="getBrand(acc).apiLogoUrl ? 'p-0 bg-slate-100 dark:bg-slate-800' : 'p-1.5'"
            :style="getBrand(acc).apiLogoUrl ? {} : { backgroundColor: getBrand(acc).bg || 'rgba(99, 102, 241, 0.1)', color: getBrand(acc).color || '#6366f1' }"
          >
            <div v-if="getBrand(acc).svg" v-html="getBrand(acc).svg" class="w-full h-full flex items-center justify-center"></div>
            <img
              v-else-if="getBrand(acc).apiLogoUrl"
              :src="getBrand(acc).apiLogoUrl"
              :alt="acc.issuer"
              class="w-full h-full object-cover"
            />
            <span v-else class="text-brand-600 dark:text-brand-400 font-bold">
              {{ (acc.issuer || acc.account_name || '?').charAt(0).toUpperCase() }}
            </span>
          </div>

          <!-- Name & Issuer -->
          <div class="min-w-0">
            <div class="flex items-center gap-1">
              <span class="font-bold text-xs text-slate-900 dark:text-white truncate leading-tight">
                {{ getDisplayTitle(acc) }}
              </span>
              <Pin v-if="acc.pinned" :size="10" class="text-brand-500 fill-brand-500 flex-shrink-0" />
            </div>
            <div class="text-[10px] text-slate-400 truncate">
              {{ acc.account_name || '无账号' }}
            </div>
          </div>
        </div>

        <!-- Right: TOTP Code & Progress -->
        <div class="flex items-center gap-2 flex-shrink-0">
          <div class="text-right">
            <span
              v-if="copiedId === acc.id"
              class="font-mono text-xs font-bold text-emerald-500 animate-pulse"
            >
              已复制!
            </span>
            <span
              v-else
              class="font-mono text-base font-extrabold tracking-wider transition-all duration-300"
              :class="remaining(acc) <= 5
                ? 'text-rose-600 dark:text-rose-400 drop-shadow-[0_0_6px_rgba(244,63,94,0.4)] animate-pulse'
                : 'text-slate-900 dark:text-white'"
            >
              {{ formatCode(acc.code) }}
            </span>
          </div>

          <!-- Countdown Circle Timer -->
          <div class="relative w-5 h-5 flex items-center justify-center">
            <svg class="w-full h-full -rotate-90" viewBox="0 0 36 36">
              <path
                class="text-slate-100 dark:text-slate-800"
                stroke-width="4"
                stroke="currentColor"
                fill="none"
                d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
              />
              <path
                :class="progressColor(acc)"
                stroke-dasharray="100, 100"
                :stroke-dashoffset="100 - (progress(acc) * 100)"
                stroke-linecap="round"
                stroke-width="4"
                stroke="currentColor"
                fill="none"
                d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
              />
            </svg>
            <span class="absolute text-[8px] font-bold font-mono text-slate-400">
              {{ remaining(acc) }}
            </span>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-if="filteredAccounts.length === 0" class="py-10 text-center text-xs text-slate-400">
        {{ searchQuery ? '未找到匹配令牌' : '暂无 2FA 令牌' }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { ShieldCheck, Search, Maximize2, Sun, Moon, Lock, Pin } from 'lucide-vue-next';
import type { Account } from '../types/vault';
import { useVault } from '../composables/useVault';
import { useTheme } from '../composables/useTheme';
import { useToast } from '../composables/useToast';
import { getBrandLogo } from '../utils/brandIcons';
import { copyWithAutoClear } from '../utils/clipboard';

defineEmits<{
  (e: 'exit-mini'): void;
}>();

const { filteredAccounts, nowSeconds, lock } = useVault();
const { theme, toggleTheme } = useTheme();
const { showToast } = useToast();

const searchQuery = ref('');
const copiedId = ref<string | null>(null);

function getBrand(acc: Account) {
  return getBrandLogo(acc.issuer || '', acc.account_name || '');
}

function getDisplayTitle(acc: Account): string {
  if (acc.issuer && acc.issuer.trim() && acc.issuer !== acc.account_name && !acc.issuer.includes('@')) {
    return acc.issuer;
  }
  const brand = getBrand(acc);
  if (brand.name) {
    return brand.name;
  }
  return acc.issuer || acc.account_name || '未命名';
}

function formatCode(code?: string): string {
  if (!code) return '------';
  if (code.length === 6) {
    return `${code.slice(0, 3)} ${code.slice(3)}`;
  }
  if (code.length === 8) {
    return `${code.slice(0, 4)} ${code.slice(4)}`;
  }
  return code;
}

function remaining(acc: Account): number {
  if (acc.type === 'hotp') return 0;
  const period = acc.period || 30;
  return period - (nowSeconds.value % period);
}

function progress(acc: Account): number {
  if (acc.type === 'hotp') return 1;
  const period = acc.period || 30;
  return remaining(acc) / period;
}

function progressColor(acc: Account): string {
  const rem = remaining(acc);
  if (rem <= 5) return 'text-rose-500';
  if (rem <= 10) return 'text-amber-500';
  return 'text-brand-500';
}

async function copyCode(acc: Account) {
  const code = acc.code;
  if (!code || code === '------') return;

  const success = await copyWithAutoClear(
    code,
    `${getDisplayTitle(acc)} 动态码: ${code}`,
    showToast
  );
  if (success) {
    copiedId.value = acc.id;
    setTimeout(() => {
      if (copiedId.value === acc.id) {
        copiedId.value = null;
      }
    }, 1500);
  }
}
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.3);
  border-radius: 4px;
}
</style>
