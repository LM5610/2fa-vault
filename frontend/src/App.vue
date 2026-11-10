<template>
  <div class="min-h-screen flex flex-col bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 transition-colors duration-150">
    <!-- Mini Window Mode (Desktop Lightweight View) -->
    <template v-if="isMiniMode && isUnlocked">
      <MiniView @exit-mini="toggleMiniMode" />
    </template>

    <!-- Normal Full-featured View -->
    <template v-else>
      <!-- Navbar Header -->
      <Navbar
        v-if="isUnlocked"
        @open-add="openAddModal(null)"
        @open-import-export="showImportExportModal = true"
        @open-sync="showSyncModal = true"
        @toggle-mini="toggleMiniMode"
        @open-settings="showSettingsModal = true"
      />

      <!-- Main Content Area -->
      <main class="flex-1 max-w-6xl w-full mx-auto px-4 sm:px-6 py-6 pb-16 safe-bottom">
        <div v-if="isUnlocked">
          <!-- Accounts Grid (Draggable & Reorderable) -->
          <div v-if="filteredAccounts.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div
              v-for="acc in filteredAccounts"
              :key="acc.id"
              class="transition-all duration-200 select-none"
              :class="{
                'opacity-40 scale-95 ring-2 ring-brand-500 rounded-2xl': draggingAccountId === acc.id,
                'cursor-grab active:cursor-grabbing': !searchQuery && !activeTag
              }"
              :draggable="!searchQuery && !activeTag"
              @dragstart="handleCardDragStart($event, acc)"
              @dragover.prevent="handleCardDragOver($event, acc)"
              @drop.stop="handleCardDrop($event, acc)"
              @dragend="handleCardDragEnd"
            >
              <AccountCard
                :account="acc"
                @view-qr="selectedAccountForQR = $event"
                @edit="openAddModal($event)"
                @delete="confirmDeleteAccount($event)"
              />
            </div>
          </div>

          <!-- Empty State -->
          <div v-else class="py-20 flex flex-col items-center justify-center text-center">
            <div class="w-20 h-20 rounded-3xl bg-brand-50 dark:bg-brand-500/10 text-brand-500 flex items-center justify-center mb-5 border border-brand-500/20">
              <ShieldPlus :size="40" />
            </div>
            <h3 class="text-xl font-bold text-slate-900 dark:text-white mb-2">
              {{ searchQuery || activeTag ? '未找到匹配的 2FA 令牌' : '还没有添加任何 2FA 令牌' }}
            </h3>
            <p class="text-sm text-slate-500 dark:text-slate-400 max-w-sm mb-6">
              {{ searchQuery || activeTag ? '请尝试更换搜索关键字或清除标签筛选' : '点击下方按钮，支持通过截图粘贴、网页图片拖拽、链接或手动添加令牌' }}
            </p>
            <button
              v-if="!searchQuery && !activeTag"
              @click="openAddModal(null)"
              class="px-5 py-2.5 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-semibold text-sm shadow-lg shadow-brand-500/25 flex items-center gap-2 transition-all"
            >
              <Plus :size="16" />
              <span>添加第一个 2FA 令牌</span>
            </button>
          </div>
        </div>
      </main>
    </template>

    <!-- Lock Screen (When locked or uninitialized) -->
    <LockScreen v-if="!isUnlocked" />

    <!-- Modals -->
    <AddModal
      v-model="showAddModal"
      :account-to-edit="accountToEdit"
      @saved="accountToEdit = null"
    />

    <QRModal
      :account="selectedAccountForQR"
      @close="selectedAccountForQR = null"
    />

    <SettingsModal v-model="showSettingsModal" />

    <ImportExportModal v-model="showImportExportModal" />

    <SyncModal v-model="showSyncModal" />

    <!-- Fullscreen Drag & Drop Overlay Indicator (Only for external images/URLs) -->
    <DragOverlay :is-dragging="isGlobalDragging" />

    <!-- Toast Notifications -->
    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { ShieldPlus, Plus } from 'lucide-vue-next';
import Navbar from './components/Navbar.vue';
import LockScreen from './components/LockScreen.vue';
import AccountCard from './components/AccountCard.vue';
import MiniView from './components/MiniView.vue';
import AddModal from './components/AddModal.vue';
import QRModal from './components/QRModal.vue';
import SettingsModal from './components/SettingsModal.vue';
import ImportExportModal from './components/ImportExportModal.vue';
import SyncModal from './components/SyncModal.vue';
import DragOverlay from './components/DragOverlay.vue';
import ToastContainer from './components/ToastContainer.vue';
import type { Account } from './types/vault';
import { useVault } from './composables/useVault';
import { useToast } from './composables/useToast';
import { API } from './api/client';

const { isUnlocked, accounts, filteredAccounts, searchQuery, activeTag, refreshStatus, refreshAccounts, refreshTags, startClock, lock } = useVault();
const { showToast } = useToast();

const isMiniMode = ref(new URLSearchParams(window.location.search).get('view') === 'mini');
const showAddModal = ref(false);
const showSettingsModal = ref(false);
const showImportExportModal = ref(false);
const showSyncModal = ref(false);
const accountToEdit = ref<Account | null>(null);
const selectedAccountForQR = ref<Account | null>(null);
const isGlobalDragging = ref(false);
const draggingAccountId = ref<string | null>(null);

let dragCounter = 0;

function toggleMiniMode() {
  isMiniMode.value = !isMiniMode.value;
  const url = new URL(window.location.href);
  if (isMiniMode.value) {
    url.searchParams.set('view', 'mini');
  } else {
    url.searchParams.delete('view');
  }
  window.history.replaceState({}, '', url.toString());
}

function openAddModal(acc: Account | null = null) {
  accountToEdit.value = acc;
  showAddModal.value = true;
}

async function confirmDeleteAccount(acc: Account) {
  const name = acc.issuer || acc.account_name || '此账号';
  if (!confirm(`确定要永久删除 ${name} 的 2FA 令牌吗？此操作无法撤销。`)) return;

  try {
    await API.deleteAccount(acc.id);
    showToast(`已删除 ${name}`);
    await refreshAccounts();
    await refreshTags();
  } catch (err: any) {
    showToast(`删除失败: ${err.message}`, true);
  }
}

// Card Drag-and-Drop Reordering Handlers
function handleCardDragStart(e: DragEvent, acc: Account) {
  draggingAccountId.value = acc.id;
  if (e.dataTransfer) {
    e.dataTransfer.setData('application/x-2fa-card-id', acc.id);
    e.dataTransfer.effectAllowed = 'move';
  }
}

function handleCardDragOver(e: DragEvent, acc: Account) {
  e.preventDefault();
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'move';
  }
}

async function handleCardDrop(e: DragEvent, targetAcc: Account) {
  e.preventDefault();
  e.stopPropagation();
  const sourceId = draggingAccountId.value || e.dataTransfer?.getData('application/x-2fa-card-id');
  draggingAccountId.value = null;

  if (!sourceId || sourceId === targetAcc.id) return;

  const currentList = [...accounts.value];
  const sourceIndex = currentList.findIndex(a => a.id === sourceId);
  const targetIndex = currentList.findIndex(a => a.id === targetAcc.id);
  if (sourceIndex === -1 || targetIndex === -1) return;

  const [moved] = currentList.splice(sourceIndex, 1);
  currentList.splice(targetIndex, 0, moved);
  accounts.value = currentList;

  try {
    const newIds = currentList.map(a => a.id);
    await API.reorderAccounts(newIds);
    showToast('已更新令牌排序');
  } catch (err: any) {
    showToast(`保存排序失败: ${err.message}`, true);
    await refreshAccounts();
  }
}

function handleCardDragEnd() {
  draggingAccountId.value = null;
}

// Global Drag & Drop Handler (for images and URLs)
function handleDragEnter(e: DragEvent) {
  e.preventDefault();
  if (draggingAccountId.value) return;

  const dt = e.dataTransfer;
  if (!dt) return;
  const hasFiles = dt.types && (dt.types.includes('Files') || dt.types.includes('text/uri-list'));
  if (!hasFiles) return;

  dragCounter++;
  if (isUnlocked.value) {
    isGlobalDragging.value = true;
  }
}

function handleDragLeave(e: DragEvent) {
  e.preventDefault();
  if (draggingAccountId.value) return;

  dragCounter--;
  if (dragCounter <= 0) {
    dragCounter = 0;
    isGlobalDragging.value = false;
  }
}

async function handleGlobalDrop(e: DragEvent) {
  e.preventDefault();
  dragCounter = 0;
  isGlobalDragging.value = false;
  if (!isUnlocked.value) return;

  const dt = e.dataTransfer;
  if (!dt) return;

  // Case A: Image file
  if (dt.files && dt.files.length > 0) {
    const file = dt.files[0];
    if (file.type.startsWith('image/')) {
      openAddModal(null);
      setTimeout(async () => {
        try {
          showToast('正在识别拖入的二维码图片...');
          const res = await API.decodeImageFile(file);
          if (res.parsed || res.text) {
            showToast('二维码图片识别成功！');
          }
        } catch (err: any) {
          showToast(err.message, true);
        }
      }, 100);
      return;
    }
  }

  // Case B: URL list or text
  const uri = dt.getData('text/uri-list') || dt.getData('text/plain');
  if (uri && (uri.startsWith('http://') || uri.startsWith('https://') || uri.startsWith('otpauth://'))) {
    openAddModal(null);
    showToast('已捕获拖入的链接');
  }
}

// Global Screenshot Paste (Ctrl + V)
function handleGlobalPaste(e: ClipboardEvent) {
  if (!isUnlocked.value || showAddModal.value) return;

  const items = e.clipboardData?.items;
  if (!items) return;

  for (const item of items) {
    if (item.type.indexOf('image') !== -1) {
      const file = item.getAsFile();
      if (file) {
        e.preventDefault();
        openAddModal(null);
        showToast('已检测到剪贴板截图，正在打开添加弹窗...');
        return;
      }
    }
  }
}

// Global Keyboard Shortcuts
function handleKeyDown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
    e.preventDefault();
    const searchInput = document.querySelector('input[type="text"]') as HTMLInputElement;
    if (searchInput) searchInput.focus();
  } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'l') {
    e.preventDefault();
    if (isUnlocked.value) lock();
  } else if (e.key === 'F5' || ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'r')) {
    e.preventDefault();
    if (isUnlocked.value) {
      refreshAccounts();
      refreshTags();
      showToast('已刷新 2FA 令牌数据');
    }
  } else if (e.key === 'Escape') {
    showAddModal.value = false;
    showSettingsModal.value = false;
    showImportExportModal.value = false;
    showSyncModal.value = false;
    selectedAccountForQR.value = null;
  }
}

function handleContextMenu(e: MouseEvent) {
  const target = e.target as HTMLElement;
  if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')) {
    return;
  }
  e.preventDefault();
}

onMounted(async () => {
  await refreshStatus();
  if (isUnlocked.value) {
    await refreshAccounts();
    await refreshTags();
  }
  startClock();

  window.addEventListener('contextmenu', handleContextMenu);
  window.addEventListener('dragenter', handleDragEnter);
  window.addEventListener('dragover', (e) => e.preventDefault());
  window.addEventListener('dragleave', handleDragLeave);
  window.addEventListener('drop', handleGlobalDrop);
  window.addEventListener('paste', handleGlobalPaste);
  window.addEventListener('keydown', handleKeyDown);
});

onUnmounted(() => {
  window.removeEventListener('contextmenu', handleContextMenu);
  window.removeEventListener('dragenter', handleDragEnter);
  window.removeEventListener('dragleave', handleDragLeave);
  window.removeEventListener('drop', handleGlobalDrop);
  window.removeEventListener('paste', handleGlobalPaste);
  window.removeEventListener('keydown', handleKeyDown);
});
</script>
