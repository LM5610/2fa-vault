<template>
  <Transition
    enter-active-class="transition duration-200 ease-out"
    enter-from-class="opacity-0 scale-95"
    enter-to-class="opacity-100 scale-100"
    leave-active-class="transition duration-150 ease-in"
    leave-from-class="opacity-100 scale-100"
    leave-to-class="opacity-0 scale-95"
  >
    <div
      v-if="isDragging"
      class="fixed inset-0 z-[9000] bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-6 pointer-events-none"
    >
      <div class="border-2 border-dashed border-brand-500 bg-brand-500/10 rounded-3xl p-12 max-w-lg w-full flex flex-col items-center text-center gap-4 shadow-2xl shadow-brand-500/20">
        <div class="w-16 h-16 rounded-2xl bg-brand-500/20 flex items-center justify-center text-brand-400">
          <UploadCloud :size="36" />
        </div>
        <h3 class="text-2xl font-bold text-white tracking-tight">释放以解析二维码</h3>
        <p class="text-slate-300 text-sm">
          支持从其他网页直接拖入图片、本地图片文件或 otpauth:// 链接
        </p>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { UploadCloud } from 'lucide-vue-next';

defineProps<{
  isDragging: boolean;
}>();
</script>
