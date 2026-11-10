<template>
  <div class="fixed bottom-6 right-6 z-[9999] flex flex-col gap-2 pointer-events-none max-w-sm w-full px-4">
    <TransitionGroup
      enter-active-class="transform ease-out duration-300 transition"
      enter-from-class="translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2"
      enter-to-class="translate-y-0 opacity-100 sm:translate-x-0"
      leave-active-class="transition ease-in duration-200"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="pointer-events-auto flex items-center justify-between gap-3 px-4 py-3 rounded-xl shadow-xl border backdrop-blur-md text-sm font-medium transition-all"
        :class="toast.isError
          ? 'bg-rose-950/90 border-rose-500/30 text-rose-200'
          : 'bg-slate-900/90 dark:bg-slate-900/90 border-slate-700/50 text-slate-100 shadow-brand-500/10'"
      >
        <div class="flex items-center gap-2.5">
          <span v-if="toast.isError" class="w-2 h-2 rounded-full bg-rose-500 animate-pulse"></span>
          <span v-else class="w-2 h-2 rounded-full bg-emerald-400"></span>
          <span>{{ toast.message }}</span>
        </div>
        <button
          @click="removeToast(toast.id)"
          class="text-slate-400 hover:text-slate-200 transition-colors p-1"
        >
          &times;
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { useToast } from '../composables/useToast';

const { toasts, removeToast } = useToast();
</script>
