import { ref } from 'vue';

export interface ToastItem {
  id: number;
  message: string;
  isError: boolean;
}

const toasts = ref<ToastItem[]>([]);
let toastId = 0;

export function useToast() {
  function showToast(message: string, isError = false, duration = 3000) {
    const id = ++toastId;
    toasts.value.push({ id, message, isError });
    setTimeout(() => {
      toasts.value = toasts.value.filter(t => t.id !== id);
    }, duration);
  }

  function removeToast(id: number) {
    toasts.value = toasts.value.filter(t => t.id !== id);
  }

  return {
    toasts,
    showToast,
    removeToast,
  };
}
