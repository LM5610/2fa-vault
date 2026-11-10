<template>
  <header class="sticky top-0 z-40 border-b border-slate-200/80 dark:border-slate-800/80 bg-white/80 dark:bg-slate-950/80 backdrop-blur-md safe-top">
    <div class="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between gap-4">
      <!-- Brand Logo -->
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-xl bg-gradient-to-tr from-brand-600 to-indigo-500 text-white flex items-center justify-center shadow-md shadow-brand-500/20">
          <ShieldCheck :size="20" />
        </div>
        <div>
          <span class="font-bold text-lg text-slate-900 dark:text-white tracking-tight">2FA Vault</span>
          <span class="hidden sm:inline-block ml-2 px-2 py-0.5 rounded text-[10px] font-semibold bg-brand-50 dark:bg-brand-500/10 text-brand-600 dark:text-brand-400 border border-brand-500/20">
            v2.0 · Vue 3
          </span>
        </div>
      </div>

      <!-- Center Search Bar (Desktop) -->
      <div class="flex-1 max-w-md hidden md:block">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" :size="16" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索服务提供商、账号或标签 (Ctrl+F)"
            class="w-full pl-9 pr-16 py-2 rounded-xl text-sm bg-slate-100 dark:bg-slate-900/80 border border-transparent focus:border-brand-500 dark:focus:border-brand-500 text-slate-900 dark:text-slate-100 placeholder-slate-400 outline-none transition-all"
          />
          <kbd class="absolute right-3 top-1/2 -translate-y-1/2 px-1.5 py-0.5 rounded text-[10px] font-mono text-slate-400 border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800">
            Ctrl+F
          </kbd>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-2">
        <button
          @click="$emit('open-add')"
          class="px-3.5 py-2 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-semibold text-sm shadow-md shadow-brand-500/20 flex items-center gap-1.5 transition-all"
        >
          <Plus :size="16" />
          <span class="hidden sm:inline">添加令牌</span>
        </button>

        <button
          @click="$emit('open-import-export')"
          class="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-colors"
          title="导入与导出"
        >
          <ArrowUpDown :size="18" />
        </button>

        <button
          @click="$emit('open-sync')"
          class="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:text-brand-500 hover:bg-brand-50 dark:hover:bg-brand-950/30 border border-slate-200 dark:border-slate-800 transition-colors"
          title="PC ↔ 手机 局域网面对面同步"
        >
          <Wifi :size="18" />
        </button>

        <button
          @click="$emit('toggle-mini')"
          class="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:text-brand-500 hover:bg-slate-100 dark:hover:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-colors"
          title="切换至桌面轻量小窗模式"
        >
          <Minimize2 :size="18" />
        </button>

        <button
          @click="toggleTheme"
          class="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-colors"
          :title="theme === 'dark' ? '切换为浅色主题' : '切换为暗色主题'"
        >
          <Sun v-if="theme === 'dark'" :size="18" />
          <Moon v-else :size="18" />
        </button>

        <button
          @click="$emit('open-settings')"
          class="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-colors"
          title="设置"
        >
          <Settings :size="18" />
        </button>

        <button
          @click="lock"
          class="p-2 rounded-xl text-slate-600 dark:text-slate-300 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-950/30 border border-slate-200 dark:border-slate-800 transition-colors"
          title="锁定保险库 (Ctrl+L)"
        >
          <Lock :size="18" />
        </button>
      </div>
    </div>

    <!-- Mobile Search Bar (Only visible on small screens) -->
    <div class="px-4 pb-3 md:hidden">
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" :size="16" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索 2FA 账号..."
          class="w-full pl-9 pr-4 py-2 rounded-xl text-sm bg-slate-100 dark:bg-slate-900/80 border border-transparent focus:border-brand-500 dark:focus:border-brand-500 text-slate-900 dark:text-slate-100 placeholder-slate-400 outline-none"
        />
      </div>
    </div>

    <!-- Tag Pills Filter Strip -->
    <div v-if="tags.length > 0" class="max-w-6xl mx-auto px-4 sm:px-6 pb-2.5 flex items-center gap-2 overflow-x-auto no-scrollbar">
      <button
        @click="activeTag = ''"
        class="px-3 py-1 rounded-full text-xs font-semibold whitespace-nowrap transition-all"
        :class="activeTag === ''
          ? 'bg-brand-600 text-white shadow-sm'
          : 'bg-slate-100 dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-800'"
      >
        全部 ({{ accounts.length }})
      </button>

      <button
        v-for="tag in tags"
        :key="tag"
        @click="activeTag = activeTag === tag ? '' : tag"
        class="px-3 py-1 rounded-full text-xs font-semibold whitespace-nowrap transition-all"
        :class="activeTag === tag
          ? 'bg-brand-600 text-white shadow-sm'
          : 'bg-slate-100 dark:bg-slate-900 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-800'"
      >
        #{{ tag }}
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ShieldCheck, Plus, Search, ArrowUpDown, Sun, Moon, Settings, Lock, Wifi, Minimize2 } from 'lucide-vue-next';
import { useVault } from '../composables/useVault';
import { useTheme } from '../composables/useTheme';

defineEmits<{
  (e: 'open-add'): void;
  (e: 'open-import-export'): void;
  (e: 'open-sync'): void;
  (e: 'toggle-mini'): void;
  (e: 'open-settings'): void;
}>();

const { accounts, tags, searchQuery, activeTag, lock } = useVault();
const { theme, toggleTheme } = useTheme();
</script>
