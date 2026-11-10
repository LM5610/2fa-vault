<template>
  <div v-if="modelValue" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/70 backdrop-blur-sm">
    <div class="w-full max-w-lg rounded-3xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 shadow-2xl overflow-hidden p-6 animate-in fade-in zoom-in-95 duration-150">
      <!-- Header -->
      <div class="flex items-center justify-between pb-4 border-b border-slate-200 dark:border-slate-800">
        <h3 class="font-bold text-lg text-slate-900 dark:text-white">数据备份与迁移</h3>
        <button
          @click="$emit('update:modelValue', false)"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
        >
          <X :size="18" />
        </button>
      </div>

      <!-- Segment Tabs -->
      <div class="flex gap-2 p-1 rounded-xl bg-slate-100 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 my-4 text-xs font-semibold">
        <button
          @click="activeTab = 'export'"
          class="flex-1 py-2 rounded-lg transition-all"
          :class="activeTab === 'export' ? 'bg-brand-600 text-white shadow-sm' : 'text-slate-500 hover:text-slate-900 dark:hover:text-white'"
        >
          导出 / 备份 (含 iCloud)
        </button>
        <button
          @click="activeTab = 'import'"
          class="flex-1 py-2 rounded-lg transition-all"
          :class="activeTab === 'import' ? 'bg-brand-600 text-white shadow-sm' : 'text-slate-500 hover:text-slate-900 dark:hover:text-white'"
        >
          导入数据
        </button>
      </div>

      <!-- Export Tab -->
      <div v-if="activeTab === 'export'" class="space-y-4">
        <p class="text-xs text-slate-500 dark:text-slate-400">
          您可以将所有 2FA 令牌导出为 Aegis 标准格式的加密 JSON 文件，或明文 otpauth:// URI 列表。
        </p>

        <div class="space-y-2">
          <!-- iCloud / iOS Share Sheet -->
          <button
            @click="exportBackup('icloud')"
            class="w-full py-3 px-4 rounded-xl border border-brand-500/30 bg-brand-50/50 dark:bg-brand-500/10 hover:bg-brand-500/20 text-brand-600 dark:text-brand-300 font-semibold text-xs transition-colors flex items-center justify-between"
          >
            <div class="flex items-center gap-2.5">
              <Cloud :size="18" />
              <div class="text-left">
                <div class="font-bold">备份至 iCloud 云盘 (iOS) / 系统分享</div>
                <div class="text-[11px] opacity-80">通过原生分享面板保存到文件或 iCloud Drive</div>
              </div>
            </div>
            <ArrowRight :size="16" />
          </button>

          <!-- Download JSON -->
          <button
            @click="exportBackup('json')"
            class="w-full py-3 px-4 rounded-xl border border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-950 text-slate-800 dark:text-slate-200 font-semibold text-xs transition-colors flex items-center justify-between"
          >
            <div class="flex items-center gap-2.5">
              <FileJson :size="18" class="text-brand-500" />
              <div class="text-left">
                <div class="font-bold">导出 Aegis JSON 文件</div>
                <div class="text-[11px] text-slate-400">兼容 Aegis, 2FA Vault 及标准验证器</div>
              </div>
            </div>
            <Download :size="16" />
          </button>

          <!-- Download URIs -->
          <button
            @click="exportBackup('uri')"
            class="w-full py-3 px-4 rounded-xl border border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-950 text-slate-800 dark:text-slate-200 font-semibold text-xs transition-colors flex items-center justify-between"
          >
            <div class="flex items-center gap-2.5">
              <Link :size="18" class="text-emerald-500" />
              <div class="text-left">
                <div class="font-bold">导出 otpauth:// 文本列表</div>
                <div class="text-[11px] text-slate-400">纯文本逐行导出，方便跨系统快速导入</div>
              </div>
            </div>
            <Download :size="16" />
          </button>
        </div>
      </div>

      <!-- Import Tab -->
      <div v-else class="space-y-4">
        <div>
          <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
            输入或粘贴 JSON 备份 / otpauth:// 列表
          </label>
          <textarea
            v-model="importText"
            rows="5"
            placeholder="粘贴 JSON 数据或多行 otpauth:// 链接..."
            class="w-full px-3.5 py-2.5 rounded-xl text-xs font-mono bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-brand-500 text-slate-900 dark:text-white outline-none"
          ></textarea>
        </div>

        <div class="flex items-center justify-between gap-3">
          <label class="px-3.5 py-2 rounded-xl border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300 font-semibold text-xs cursor-pointer transition-colors flex items-center gap-1.5">
            <Upload :size="14" />
            <span>选择本地备份文件</span>
            <input type="file" accept=".json,.txt" class="hidden" @change="onFileImport" />
          </label>

          <button
            @click="submitImport"
            :disabled="isImporting || !importText.trim()"
            class="px-5 py-2 rounded-xl bg-brand-600 hover:bg-brand-500 text-white font-semibold text-xs shadow-md shadow-brand-500/20 transition-all disabled:opacity-50"
          >
            {{ isImporting ? '正在导入...' : '开始导入' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { X, Cloud, FileJson, Link, Download, Upload, ArrowRight } from 'lucide-vue-next';
import { API } from '../api/client';
import { useVault } from '../composables/useVault';
import { useToast } from '../composables/useToast';

defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
}>();

const { refreshAccounts, refreshTags } = useVault();
const { showToast } = useToast();

const activeTab = ref<'export' | 'import'>('export');
const importText = ref('');
const isImporting = ref(false);

async function exportBackup(type: 'icloud' | 'json' | 'uri') {
  try {
    const format = type === 'uri' ? 'uri' : 'json';
    const res = await API.getExportData(format);
    const content = res.data;
    const dateStr = new Date().toISOString().slice(0, 10);
    const filename = `2fa-vault-backup-${dateStr}.${format === 'json' ? 'json' : 'txt'}`;

    if (type === 'icloud' && navigator.share) {
      const file = new File([content], filename, {
        type: format === 'json' ? 'application/json' : 'text/plain',
      });
      if (navigator.canShare && navigator.canShare({ files: [file] })) {
        await navigator.share({
          files: [file],
          title: '2FA Vault 备份',
          text: '导出 2FA 加密备份至 iCloud 云盘或本地文件',
        });
        showToast('已调起系统分享 / iCloud 云盘存储');
        return;
      }
    }

    // Direct browser download
    const blob = new Blob([content], {
      type: format === 'json' ? 'application/json;charset=utf-8' : 'text/plain;charset=utf-8',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    showToast(`备份文件已导出: ${filename}`);
  } catch (err: any) {
    showToast(`导出失败: ${err.message}`, true);
  }
}

function onFileImport(e: Event) {
  const target = e.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = (evt) => {
    importText.value = (evt.target?.result as string) || '';
  };
  reader.readAsText(file);
}

async function submitImport() {
  if (!importText.value.trim()) return;
  isImporting.value = true;
  try {
    const res = await API.importData(importText.value.trim());
    showToast(`成功导入 ${res.imported} 个 2FA 令牌！`);
    importText.value = '';
    await refreshAccounts();
    await refreshTags();
    emit('update:modelValue', false);
  } catch (err: any) {
    showToast(`导入失败: ${err.message}`, true);
  } finally {
    isImporting.value = false;
  }
}
</script>
