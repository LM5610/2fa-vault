<template>
  <div v-if="modelValue" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/70 backdrop-blur-sm">
    <div class="w-full max-w-lg rounded-3xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-2xl overflow-hidden p-6 animate-in fade-in zoom-in-95 duration-150 flex flex-col max-h-[85vh]">
      <!-- Header -->
      <div class="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800 flex-shrink-0">
        <h3 class="font-bold text-lg text-slate-900 dark:text-white">系统与安全设置</h3>
        <button
          @click="$emit('update:modelValue', false)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          <X :size="18" />
        </button>
      </div>

      <div class="py-5 space-y-6 overflow-y-auto pr-1 custom-scrollbar flex-1">
        <!-- 1. Auto-lock & Clipboard Security Settings -->
        <div class="space-y-4">
          <h4 class="text-xs font-bold uppercase tracking-wider text-slate-400">常规与隐私安全</h4>
          
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <!-- Auto-lock Setting -->
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                自动锁定时间
              </label>
              <select
                v-model="autoLock"
                @change="updateAutoLock"
                class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              >
                <option :value="1">1 分钟无操作</option>
                <option :value="5">5 分钟无操作</option>
                <option :value="15">15 分钟无操作 (推荐)</option>
                <option :value="30">30 分钟无操作</option>
                <option :value="60">1 小时无操作</option>
              </select>
            </div>

            <!-- Auto-clear Clipboard Setting -->
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5 flex items-center justify-between">
                <span>剪贴板自动焚毁</span>
                <span class="text-[10px] text-brand-500 font-normal">防嗅探</span>
              </label>
              <select
                v-model="clipboardClearSec"
                @change="updateClipboardClear"
                class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              >
                <option :value="15">15 秒后自动清空</option>
                <option :value="30">30 秒后自动清空 (推荐)</option>
                <option :value="60">60 秒后自动清空</option>
                <option :value="0">从不清空 (不推荐)</option>
              </select>
            </div>
          </div>
        </div>

        <!-- 2. Theme Segmented Control -->
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

        <!-- 3. Encrypted Rolling Snapshots & Rollback -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <div>
              <h4 class="text-xs font-bold uppercase tracking-wider text-slate-400">历史加密快照与防误删回滚</h4>
              <p class="text-[11px] text-slate-500 mt-0.5">每次修改自动在本地创建加密历史点，最多保留最近 10 次修改</p>
            </div>
            <button
              @click="loadSnapshots"
              type="button"
              class="p-1 rounded text-slate-400 hover:text-brand-500 transition-colors"
              title="刷新快照列表"
            >
              <RotateCcw :size="14" :class="{ 'animate-spin': loadingSnapshots }" />
            </button>
          </div>

          <div class="rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden bg-slate-50 dark:bg-slate-950/60 divide-y divide-slate-200 dark:divide-slate-800/80">
            <div v-if="loadingSnapshots" class="p-4 text-center text-xs text-slate-400">
              正在加载历史快照...
            </div>
            <div v-else-if="snapshots.length === 0" class="p-4 text-center text-xs text-slate-400">
              暂无历史快照
            </div>
            <div
              v-else
              v-for="(s, idx) in snapshots"
              :key="s.filename"
              class="p-3 flex items-center justify-between gap-3 text-xs"
            >
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-mono font-bold text-slate-900 dark:text-white">
                    {{ formatSnapshotTime(s.created_at) }}
                  </span>
                  <span v-if="idx === 0" class="px-1.5 py-0.2 rounded text-[10px] font-semibold bg-brand-50 dark:bg-brand-950/60 text-brand-600 dark:text-brand-400 border border-brand-500/20">
                    最新
                  </span>
                </div>
                <div class="text-[10px] text-slate-400 mt-0.5 font-mono">
                  {{ s.filename }} · {{ (s.size_bytes / 1024).toFixed(1) }} KB
                </div>
              </div>

              <button
                @click="confirmRollback(s)"
                :disabled="isRollingBack"
                type="button"
                class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-colors bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-brand-500 hover:text-brand-600 dark:hover:text-brand-400 disabled:opacity-50"
              >
                回滚到此版本
              </button>
            </div>
          </div>
        </div>

        <!-- 4. WebDAV Cloud Backup & Sync Collapsible Section -->
        <div class="rounded-2xl border border-slate-200 dark:border-slate-800 overflow-hidden bg-slate-50/50 dark:bg-slate-950/40">
          <button
            type="button"
            @click="isWebDAVOpen = !isWebDAVOpen"
            class="w-full px-4 py-3.5 flex items-center justify-between text-left hover:bg-slate-100/50 dark:hover:bg-slate-800/50 transition-colors"
          >
            <div class="flex items-center gap-2.5">
              <div class="p-2 rounded-xl bg-brand-500/10 text-brand-500 dark:text-brand-400">
                <Cloud :size="16" />
              </div>
              <div>
                <div class="flex items-center gap-2">
                  <span class="text-xs font-bold text-slate-900 dark:text-white">WebDAV 云端备份与同步</span>
                  <span v-if="webdavCfg.auto_sync" class="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
                    自动同步开启
                  </span>
                </div>
                <p class="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">
                  支持坚果云、Nextcloud、群晖 Synology 等标准 WebDAV 网盘
                </p>
              </div>
            </div>
            <ChevronDown
              :size="16"
              class="text-slate-400 transition-transform duration-200"
              :class="{ 'rotate-180': isWebDAVOpen }"
            />
          </button>

          <!-- Collapsible Body -->
          <div v-if="isWebDAVOpen" class="p-4 pt-2 border-t border-slate-200/80 dark:border-slate-800/80 space-y-4">
            <!-- Sync Status Tag / Message -->
            <div v-if="webdavCfg.last_sync || webdavCfg.last_error" class="p-2.5 rounded-xl text-xs flex items-center justify-between"
              :class="webdavCfg.last_status === 'error'
                ? 'bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 border border-rose-200 dark:border-rose-900/50'
                : 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-900/50'"
            >
              <div class="flex items-center gap-2 min-w-0">
                <AlertCircle v-if="webdavCfg.last_status === 'error'" :size="14" class="flex-shrink-0" />
                <Check v-else :size="14" class="flex-shrink-0" />
                <span class="truncate">
                  {{ webdavCfg.last_status === 'error' ? `同步异常: ${webdavCfg.last_error}` : `上次同步: ${formatSnapshotTime(webdavCfg.last_sync || '')}` }}
                </span>
              </div>
            </div>

            <!-- Server URL -->
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                WebDAV 服务器地址
              </label>
              <input
                v-model="webdavCfg.url"
                type="text"
                placeholder="例如 https://dav.jianguoyun.com/dav/ 或 Nextcloud 地址"
                class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              />
            </div>

            <!-- Username & Password -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                  用户名 / 账号
                </label>
                <input
                  v-model="webdavCfg.username"
                  type="text"
                  placeholder="坚果云/Nextcloud 用户名"
                  class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
                />
              </div>

              <div>
                <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5 flex items-center justify-between">
                  <span>密码 / 应用令牌</span>
                  <span v-if="webdavCfg.has_password" class="text-[10px] text-emerald-500 font-normal">已保存</span>
                </label>
                <input
                  v-model="webdavPasswordInput"
                  type="password"
                  :placeholder="webdavCfg.has_password ? '•••••••• (留空保持不变)' : '密码或应用专用令牌'"
                  class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
                />
              </div>
            </div>

            <!-- Remote Directory -->
            <div>
              <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                云端备份路径
              </label>
              <input
                v-model="webdavCfg.remote_dir"
                type="text"
                placeholder="默认 /2fa-vault (备份文件为 vault.enc)"
                class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
              />
            </div>

            <!-- AutoSync Toggle -->
            <div class="flex items-center justify-between p-3 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
              <div>
                <div class="text-xs font-semibold text-slate-900 dark:text-white">修改自动备份至云端</div>
                <div class="text-[11px] text-slate-500 dark:text-slate-400">本地数据变更时，自动加密上传至 WebDAV</div>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  v-model="webdavCfg.auto_sync"
                  @change="handleAutoSyncToggle"
                  class="sr-only peer"
                />
                <div class="w-9 h-5 bg-slate-300 peer-focus:outline-none rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-brand-600"></div>
              </label>
            </div>

            <!-- Action Buttons Grid -->
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1">
              <button
                type="button"
                @click="testWebDAVConnection"
                :disabled="isTestingWebDAV || !webdavCfg.url"
                class="px-2.5 py-2 rounded-xl text-xs font-semibold transition-all bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 hover:border-brand-500 hover:text-brand-600 dark:hover:text-brand-400 disabled:opacity-50 flex items-center justify-center gap-1.5"
              >
                <RefreshCw :size="13" :class="{ 'animate-spin': isTestingWebDAV }" />
                <span>{{ isTestingWebDAV ? '测试中...' : '测试连接' }}</span>
              </button>

              <button
                type="button"
                @click="saveWebDAVSettings"
                :disabled="isSavingWebDAV"
                class="px-2.5 py-2 rounded-xl text-xs font-semibold transition-all bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 hover:border-brand-500 hover:text-brand-600 dark:hover:text-brand-400 disabled:opacity-50 flex items-center justify-center gap-1.5"
              >
                <span>{{ isSavingWebDAV ? '保存中...' : '保存配置' }}</span>
              </button>

              <button
                type="button"
                @click="backupToWebDAV"
                :disabled="isBackingUpWebDAV || !webdavCfg.url"
                class="px-2.5 py-2 rounded-xl text-xs font-semibold transition-all bg-brand-600 hover:bg-brand-500 text-white shadow-sm disabled:opacity-50 flex items-center justify-center gap-1.5"
              >
                <CloudUpload :size="13" :class="{ 'animate-bounce': isBackingUpWebDAV }" />
                <span>{{ isBackingUpWebDAV ? '备份中...' : '立即备份' }}</span>
              </button>

              <button
                type="button"
                @click="restoreFromWebDAV"
                :disabled="isRestoringWebDAV || !webdavCfg.url"
                class="px-2.5 py-2 rounded-xl text-xs font-semibold transition-all bg-amber-600 hover:bg-amber-500 text-white shadow-sm disabled:opacity-50 flex items-center justify-center gap-1.5"
              >
                <CloudDownload :size="13" :class="{ 'animate-bounce': isRestoringWebDAV }" />
                <span>{{ isRestoringWebDAV ? '还原中...' : '从云端还原' }}</span>
              </button>
            </div>
          </div>
        </div>

        <hr class="border-slate-200 dark:border-slate-800" />

        <!-- 5. Change Master Password -->
        <form @submit.prevent="handleChangePassword" class="space-y-3">
          <h4 class="text-xs font-bold uppercase tracking-wider text-slate-400">修改主密码</h4>
          <input
            v-model="oldPassword"
            type="password"
            placeholder="当前主密码"
            required
            class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
          />
          <input
            v-model="newPassword"
            type="password"
            placeholder="新主密码 (至少 6 位)"
            required
            class="w-full px-3.5 py-2.5 rounded-xl text-xs bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
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
import { ref, onMounted, watch } from 'vue';
import { X, Moon, Sun, RotateCcw, Cloud, CloudUpload, CloudDownload, ChevronDown, Check, AlertCircle, RefreshCw } from 'lucide-vue-next';
import type { SnapshotInfo, WebDAVConfig } from '../types/vault';
import { useVault } from '../composables/useVault';
import { useTheme } from '../composables/useTheme';
import { useToast } from '../composables/useToast';
import { API } from '../api/client';
import { getAutoClearSeconds, setAutoClearSeconds } from '../utils/clipboard';

const props = defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
}>();

const { status, refreshAccounts, refreshTags } = useVault();
const { theme, applyTheme } = useTheme();
const { showToast } = useToast();

const autoLock = ref(status.value.auto_lock_minutes || 15);
const clipboardClearSec = ref(getAutoClearSeconds());
const oldPassword = ref('');
const newPassword = ref('');
const isUpdatingPassword = ref(false);

const snapshots = ref<SnapshotInfo[]>([]);
const loadingSnapshots = ref(false);
const isRollingBack = ref(false);

// WebDAV state
const isWebDAVOpen = ref(false);
const webdavCfg = ref<WebDAVConfig>({
  url: '',
  username: '',
  auto_sync: false,
  remote_dir: '/2fa-vault',
});
const webdavPasswordInput = ref('');
const isTestingWebDAV = ref(false);
const isSavingWebDAV = ref(false);
const isBackingUpWebDAV = ref(false);
const isRestoringWebDAV = ref(false);

async function loadWebDAVSettings() {
  try {
    const cfg = await API.getWebDAVConfig();
    if (cfg) {
      webdavCfg.value = cfg;
    }
  } catch (err: any) {
    console.error('Failed to load WebDAV settings:', err);
  }
}

async function saveWebDAVSettings() {
  isSavingWebDAV.value = true;
  try {
    const payload: Partial<WebDAVConfig> = {
      url: webdavCfg.value.url,
      username: webdavCfg.value.username,
      auto_sync: webdavCfg.value.auto_sync,
      remote_dir: webdavCfg.value.remote_dir,
    };
    if (webdavPasswordInput.value) {
      payload.password = webdavPasswordInput.value;
    }
    await API.saveWebDAVConfig(payload);
    showToast('WebDAV 配置已保存');
    webdavPasswordInput.value = '';
    await loadWebDAVSettings();
  } catch (err: any) {
    showToast(`保存失败: ${err.message}`, true);
  } finally {
    isSavingWebDAV.value = false;
  }
}

async function handleAutoSyncToggle() {
  await saveWebDAVSettings();
}

async function testWebDAVConnection() {
  if (!webdavCfg.value.url) {
    showToast('请输入 WebDAV 服务器地址', true);
    return;
  }
  isTestingWebDAV.value = true;
  try {
    const payload: { url?: string; username?: string; password?: string } = {
      url: webdavCfg.value.url,
      username: webdavCfg.value.username,
    };
    if (webdavPasswordInput.value) {
      payload.password = webdavPasswordInput.value;
    }
    const res = await API.testWebDAV(payload);
    showToast(res.message || 'WebDAV 连接成功！');
  } catch (err: any) {
    showToast(`连接失败: ${err.message}`, true);
  } finally {
    isTestingWebDAV.value = false;
  }
}

async function backupToWebDAV() {
  if (!webdavCfg.value.url) {
    showToast('请先配置 WebDAV 服务器地址', true);
    return;
  }
  isBackingUpWebDAV.value = true;
  try {
    if (webdavPasswordInput.value) {
      await saveWebDAVSettings();
    }
    const res = await API.backupWebDAV();
    showToast(res.message || '已成功备份至 WebDAV 云端！');
    await loadWebDAVSettings();
  } catch (err: any) {
    showToast(`备份失败: ${err.message}`, true);
  } finally {
    isBackingUpWebDAV.value = false;
  }
}

async function restoreFromWebDAV() {
  if (!webdavCfg.value.url) {
    showToast('请先配置 WebDAV 服务器地址', true);
    return;
  }
  if (!confirm('从 WebDAV 云端还原将会覆盖本地当前保险库，是否继续？')) {
    return;
  }
  isRestoringWebDAV.value = true;
  try {
    const res = await API.restoreWebDAV();
    showToast(res.message || '已成功从 WebDAV 云端还原保险库！');
    await refreshAccounts();
    await refreshTags();
    await loadSnapshots();
    await loadWebDAVSettings();
  } catch (err: any) {
    showToast(`还原失败: ${err.message}`, true);
  } finally {
    isRestoringWebDAV.value = false;
  }
}

async function loadSnapshots() {
  loadingSnapshots.value = true;
  try {
    const list = await API.getSnapshots();
    snapshots.value = list || [];
  } catch (err: any) {
    showToast(`加载快照失败: ${err.message}`, true);
  } finally {
    loadingSnapshots.value = false;
  }
}

function formatSnapshotTime(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    return d.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  } catch {
    return dateStr;
  }
}

async function confirmRollback(snapshot: SnapshotInfo) {
  const timeFormatted = formatSnapshotTime(snapshot.created_at);
  if (!confirm(`确定要将保险库回滚到【${timeFormatted}】的版本吗？\n当前的最新状态也会自动备份为一个新快照，安全无忧。`)) {
    return;
  }

  isRollingBack.value = true;
  try {
    await API.rollbackSnapshot(snapshot.filename);
    showToast(`已成功回滚到 ${timeFormatted} 的快照！`);
    await refreshAccounts();
    await refreshTags();
    await loadSnapshots();
    emit('update:modelValue', false);
  } catch (err: any) {
    showToast(`回滚失败: ${err.message}`, true);
  } finally {
    isRollingBack.value = false;
  }
}

async function updateAutoLock() {
  try {
    await API.setAutoLock(autoLock.value);
    showToast('自动锁定时间已更新');
  } catch (err: any) {
    showToast(`更新失败: ${err.message}`, true);
  }
}

function updateClipboardClear() {
  setAutoClearSeconds(clipboardClearSec.value);
  if (clipboardClearSec.value > 0) {
    showToast(`已启用剪贴板安全保护: 复制 ${clipboardClearSec.value} 秒后自动焚毁`);
  } else {
    showToast('已关闭剪贴板自动清空');
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

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      loadSnapshots();
      loadWebDAVSettings();
      clipboardClearSec.value = getAutoClearSeconds();
    }
  }
);

onMounted(() => {
  clipboardClearSec.value = getAutoClearSeconds();
  loadWebDAVSettings();
});
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 5px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.3);
  border-radius: 4px;
}
</style>
