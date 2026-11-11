import { ref, computed } from 'vue';
import { API } from '../api/client';
import type { Account, VaultStatus } from '../types/vault';
import { useToast } from './useToast';

const status = ref<VaultStatus>({
  initialized: false,
  locked: true,
  auto_lock_minutes: 15,
});

const accounts = ref<Account[]>([]);
const tags = ref<string[]>([]);
const searchQuery = ref('');
const activeTag = ref('');
const nowSeconds = ref(Math.floor(Date.now() / 1000));
const loading = ref(false);
const timeDriftSeconds = ref<number>(0);
const isTimeDrifted = ref(false);

let timer: number | null = null;
let lastStep = -1;

export function useVault() {
  const { showToast } = useToast();

  const isUnlocked = computed(() => status.value.initialized && !status.value.locked);

  const filteredAccounts = computed(() => {
    let list = accounts.value;
    if (activeTag.value) {
      list = list.filter(a => a.tags && a.tags.includes(activeTag.value));
    }
    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase().trim();
      list = list.filter(a =>
        a.issuer.toLowerCase().includes(q) ||
        a.account_name.toLowerCase().includes(q) ||
        (a.tags && a.tags.some(t => t.toLowerCase().includes(q)))
      );
    }
    return list;
  });

  async function refreshStatus() {
    try {
      const res = await API.getStatus();
      status.value = res;
    } catch (err: any) {
      showToast('连接后台服务失败，请确认服务已启动', true);
    }
  }

  async function refreshAccounts() {
    if (!isUnlocked.value) return;
    try {
      loading.value = true;
      const res = await API.getAccounts(searchQuery.value, activeTag.value);
      accounts.value = res || [];
    } catch (err: any) {
      showToast(`加载账号失败: ${err.message}`, true);
    } finally {
      loading.value = false;
    }
  }

  async function refreshTags() {
    if (!isUnlocked.value) return;
    try {
      const res = await API.getTags();
      tags.value = res || [];
    } catch {}
  }

  async function checkTimeDrift() {
    try {
      const res = await API.getTimeDrift();
      if (res) {
        timeDriftSeconds.value = Math.abs(res.drift_seconds);
        if (res.is_drifted || Math.abs(res.drift_seconds) > 5) {
          isTimeDrifted.value = true;
          showToast(`⚠️ 检测到系统时钟偏差 ${Math.abs(res.drift_seconds).toFixed(1)} 秒，可能导致验证码计算错误，请校准系统时间`, true);
        } else {
          isTimeDrifted.value = false;
        }
      }
    } catch {
      // Quietly ignore network/offline issues
    }
  }

  async function unlock(password: string) {
    await API.unlockVault(password);
    await refreshStatus();
    await refreshAccounts();
    await refreshTags();
    showToast('保险库已解锁');
    checkTimeDrift();
  }

  async function lock() {
    await API.lockVault();
    await refreshStatus();
    accounts.value = [];
    showToast('保险库已安全锁定');
  }

  async function init(password: string) {
    await API.initVault(password);
    await refreshStatus();
    await refreshAccounts();
    await refreshTags();
    showToast('保险库已成功初始化并解锁');
    checkTimeDrift();
  }

  // Start global synchronized clock tick
  function startClock() {
    if (timer) return;
    timer = window.setInterval(() => {
      const now = Math.floor(Date.now() / 1000);
      nowSeconds.value = now;

      // Check 30s period rollover to update accounts
      if (isUnlocked.value && accounts.value.length > 0) {
        const currentStep = Math.floor(now / 30);
        if (lastStep !== -1 && currentStep !== lastStep) {
          // Quietly re-fetch accounts to update dynamic codes
          API.getAccounts(searchQuery.value, activeTag.value)
            .then(res => {
              accounts.value = res || [];
            })
            .catch(() => {});
        }
        lastStep = currentStep;
      }
    }, 1000);
  }

  return {
    status,
    accounts,
    tags,
    searchQuery,
    activeTag,
    nowSeconds,
    loading,
    isUnlocked,
    filteredAccounts,
    timeDriftSeconds,
    isTimeDrifted,
    checkTimeDrift,
    refreshStatus,
    refreshAccounts,
    refreshTags,
    unlock,
    lock,
    init,
    startClock,
  };
}
