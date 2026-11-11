<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/90 backdrop-blur-xl">
    <div class="w-full max-w-md rounded-3xl border border-slate-800 bg-slate-900/90 p-8 shadow-2xl shadow-brand-500/10 text-center animate-in fade-in zoom-in-95 duration-200">
      <!-- Icon Avatar -->
      <div class="mx-auto w-16 h-16 rounded-2xl bg-gradient-to-tr from-brand-600 to-indigo-500 text-white flex items-center justify-center shadow-lg shadow-brand-500/30 mb-6">
        <Lock v-if="status.initialized" :size="32" />
        <ShieldCheck v-else :size="32" />
      </div>

      <h2 class="text-2xl font-extrabold text-white tracking-tight">
        {{ status.initialized ? '保险库已安全锁定' : '欢迎使用 2FA Vault' }}
      </h2>
      <p class="text-sm text-slate-400 mt-2 mb-6">
        {{ status.initialized ? '请输入主密码以解密并查看您的 2FA 令牌' : '首次启动，请设置主密码以初始化您的纯本地加密保险库' }}
      </p>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div class="relative">
          <input
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            :placeholder="status.initialized ? '输入主密码解锁...' : '设置主密码 (建议至少 6 位)...'"
            required
            autofocus
            class="w-full px-4 py-3 pr-11 rounded-xl bg-slate-950 border border-slate-700 focus:border-brand-500 text-white placeholder-slate-500 outline-none transition-all text-sm font-medium"
          />
          <button
            type="button"
            @click="showPassword = !showPassword"
            class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200 transition-colors p-1"
          >
            <EyeOff v-if="showPassword" :size="18" />
            <Eye v-else :size="18" />
          </button>
        </div>

        <button
          type="submit"
          :disabled="loading || !password"
          class="w-full py-3 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-bold text-sm shadow-lg shadow-brand-500/25 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <span v-if="loading" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
          <span>{{ status.initialized ? '解锁保险库' : '创建主密码并初始化' }}</span>
        </button>
      </form>

      <div class="mt-6 pt-6 border-t border-slate-800/80 flex items-center justify-center gap-4 text-xs text-slate-500">
        <div class="flex items-center gap-1.5">
          <ShieldAlert :size="14" class="text-brand-400" />
          <span>纯本地沙盒加密 · PBKDF2 + AES-256-GCM 零网络依赖</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { Lock, ShieldCheck, ShieldAlert, Eye, EyeOff } from 'lucide-vue-next';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';

const { status, unlock, init } = useVault();
const { showToast } = useToast();

const password = ref('');
const showPassword = ref(false);
const loading = ref(false);

async function handleSubmit() {
  if (!password.value) return;
  loading.value = true;
  try {
    if (!status.value.initialized) {
      await init(password.value);
    } else {
      await unlock(password.value);
    }
    password.value = '';
  } catch (err: any) {
    showToast(err.message, true);
  } finally {
    loading.value = false;
  }
}
</script>
