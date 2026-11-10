<template>
  <div v-if="modelValue" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/70 backdrop-blur-sm">
    <div class="w-full max-w-md rounded-3xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-2xl overflow-hidden p-6 animate-in fade-in zoom-in-95 duration-150">
      <!-- Header -->
      <div class="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800">
        <h3 class="font-bold text-lg text-slate-900 dark:text-white">系统与安全设置</h3>
        <button
          @click="$emit('update:modelValue', false)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          <X :size="18" />
        </button>
      </div>

      <div class="py-5 space-y-5">
        <!-- Auto-lock Setting -->
        <div>
          <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
            自动锁定时间
          </label>
          <select
            v-model="autoLock"
            @change="updateAutoLock"
            class="w-full px-3.5 py-2.5 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
          >
            <option :value="1">1 分钟无操作</option>
            <option :value="5">5 分钟无操作</option>
            <option :value="15">15 分钟无操作 (推荐)</option>
            <option :value="30">30 分钟无操作</option>
            <option :value="60">1 小时无操作</option>
          </select>
        </div>

        <!-- Theme Segmented Control -->
        <div>
          <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
            界面色彩主题
          </label>
          <div class="grid grid-cols-2 gap-2 p-1 rounded-xl bg-slate-100 dark:bg-slate-950 border border-slate-200 dark:border-slate-800">
            <button
              type="button"
              @click="applyTheme('dark')"
              class="py-2 rounded-lg text-xs font-semibold transition-all flex items-center justify-center gap-1.5"
              :class="theme === 'dark'
                ? 'bg-brand-600 text-white shadow-sm'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'"
            >
              <Moon :size="14" />
              <span>🌙 暗色模式 (推荐)</span>
            </button>
            <button
              type="button"
              @click="applyTheme('light')"
              class="py-2 rounded-lg text-xs font-semibold transition-all flex items-center justify-center gap-1.5"
              :class="theme === 'light'
                ? 'bg-brand-600 text-white shadow-sm'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'"
            >
              <Sun :size="14" />
              <span>☀️ 亮色模式</span>
            </button>
          </div>
        </div>

        <hr class="border-slate-200 dark:border-slate-800" />

        <!-- Change Master Password -->
        <form @submit.prevent="handleChangePassword" class="space-y-3">
          <h4 class="text-sm font-bold text-slate-900 dark:text-white">修改主密码</h4>
          <input
            v-model="oldPassword"
            type="password"
            placeholder="当前主密码"
            required
            class="w-full px-3.5 py-2.5 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
          />
          <input
            v-model="newPassword"
            type="password"
            placeholder="新主密码 (至少 6 位)"
            required
            class="w-full px-3.5 py-2.5 rounded-xl text-sm bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
          />
          <button
            type="submit"
            :disabled="isUpdatingPassword || !oldPassword || !newPassword"
            class="w-full py-2.5 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-semibold text-xs shadow-md shadow-brand-500/20 transition-all disabled:opacity-50"
          >
            {{ isUpdatingPassword ? '正在更新密码...' : '更新主密码' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { X, Moon, Sun } from 'lucide-vue-next';
import { useVault } from '../composables/useVault';
import { useTheme } from '../composables/useTheme';
import { useToast } from '../composables/useToast';
import { API } from '../api/client';

defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
}>();

const { status } = useVault();
const { theme, applyTheme } = useTheme();
const { showToast } = useToast();

const autoLock = ref(status.value.auto_lock_minutes || 15);
const oldPassword = ref('');
const newPassword = ref('');
const isUpdatingPassword = ref(false);

async function updateAutoLock() {
  try {
    await API.setAutoLock(autoLock.value);
    showToast('自动锁定时间已更新');
  } catch (err: any) {
    showToast(`更新失败: ${err.message}`, true);
  }
}

async function handleChangePassword() {
  if (!oldPassword.value || !newPassword.value) return;
  isUpdatingPassword.value = true;
  try {
    await API.changePassword(oldPassword.value, newPassword.value);
    showToast('主密码修改成功！');
    oldPassword.value = '';
    newPassword.value = '';
    emit('update:modelValue', false);
  } catch (err: any) {
    showToast(`修改失败: ${err.message}`, true);
  } finally {
    isUpdatingPassword.value = false;
  }
}
</script>
