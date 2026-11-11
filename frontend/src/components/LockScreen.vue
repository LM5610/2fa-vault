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
        {{ status.initialized ? '请输入主密码以解密并查看您的 2FA 令牌' : '首次启动，请设置主密码以初始化您的本地加密保险库' }}
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

      <!-- Server Configuration Pill -->
      <div class="mt-4">
        <button
          type="button"
          @click="openServerConfig"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium border transition-colors"
          :class="serverUrl
            ? 'border-brand-500/30 text-brand-400 bg-brand-500/10 hover:bg-brand-500/20'
            : 'border-slate-700 text-slate-400 hover:text-slate-200 bg-slate-800/50 hover:bg-slate-800'"
        >
          <Server :size="13" />
          <span class="max-w-[200px] truncate">{{ serverUrl ? '已连接: ' + serverUrl : '设置后端服务地址 (手机/局域网连接)' }}</span>
          <Settings2 :size="12" class="opacity-70" />
        </button>
      </div>

      <div class="mt-6 pt-6 border-t border-slate-800/80 flex items-center justify-center gap-4 text-xs text-slate-500">
        <div class="flex items-center gap-1.5">
          <ShieldAlert :size="14" class="text-brand-400" />
          <span>Argon2id + AES-256-GCM 硬件级加密</span>
        </div>
      </div>
    </div>

    <!-- Server Configuration Modal -->
    <div v-if="showServerModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-md">
      <div class="w-full max-w-sm rounded-3xl border border-slate-800 bg-slate-900 p-6 text-left shadow-2xl animate-in fade-in zoom-in-95 duration-150">
        <div class="flex items-center justify-between pb-3 border-b border-slate-800">
          <div class="flex items-center gap-2">
            <Server :size="18" class="text-brand-500" />
            <h3 class="font-bold text-sm text-white">后端服务地址配置</h3>
          </div>
          <button @click="showServerModal = false" class="p-1 rounded-lg text-slate-400 hover:text-white">
            <X :size="16" />
          </button>
        </div>

        <div class="py-4 space-y-3">
          <p class="text-xs text-slate-400 leading-relaxed">
            在手机端独立 App 运行时，需要连接到运行 2FA Vault 的电脑（处于同一 Wi-Fi）或私有云服务器。
          </p>

          <div>
            <label class="block text-xs font-semibold text-slate-300 mb-1.5">
              服务端地址 (IP + 端口)
            </label>
            <input
              v-model="inputServerUrl"
              type="url"
              placeholder="例如: http://192.168.1.100:2233"
              class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-slate-950 border border-slate-700 focus:border-brand-500 text-white outline-none font-mono"
            />
            <p class="text-[11px] text-slate-500 mt-1">
              可在电脑端 2FA Vault 运行界面或控制台查看局域网 IP
            </p>
          </div>

          <div class="flex gap-2 pt-2">
            <button
              type="button"
              @click="testAndSaveServer"
              :disabled="testingServer"
              class="flex-1 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-bold text-xs transition-colors flex items-center justify-center gap-1.5 disabled:opacity-50"
            >
              <RefreshCw v-if="testingServer" :size="13" class="animate-spin" />
              <span>{{ testingServer ? '正在测试连接...' : '测试并保存' }}</span>
            </button>
            <button
              type="button"
              @click="resetToDefault"
              class="px-3 py-2.5 rounded-xl border border-slate-700 hover:bg-slate-800 text-slate-400 hover:text-white text-xs transition-colors"
            >
              重置
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { Lock, ShieldCheck, ShieldAlert, Eye, EyeOff, Server, Settings2, X, RefreshCw } from 'lucide-vue-next';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';
import { getApiBaseUrl, setApiBaseUrl, API } from '../api/client';

const { status, unlock, init, refreshStatus } = useVault();
const { showToast } = useToast();

const password = ref('');
const showPassword = ref(false);
const loading = ref(false);

const serverUrl = ref(getApiBaseUrl());
const showServerModal = ref(false);
const inputServerUrl = ref('');
const testingServer = ref(false);

function openServerConfig() {
  inputServerUrl.value = serverUrl.value || '';
  showServerModal.value = true;
}

async function testAndSaveServer() {
  const target = inputServerUrl.value.trim().replace(/\/+$/, '');
  if (!target) {
    showToast('请输入有效的服务器地址', true);
    return;
  }
  testingServer.value = true;
  try {
    await API.testServerUrl(target);
    setApiBaseUrl(target);
    serverUrl.value = target;
    showToast('已成功连接到后端服务！');
    showServerModal.value = false;
    await refreshStatus();
  } catch (err: any) {
    showToast(`连接失败: ${err.message}`, true);
  } finally {
    testingServer.value = false;
  }
}

async function resetToDefault() {
  setApiBaseUrl('');
  serverUrl.value = '';
  showServerModal.value = false;
  showToast('已重置为默认本地服务');
  await refreshStatus();
}

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
