// Application State Store

export const state = {
  initialized: false,
  locked: true,
  autoLockMinutes: 15,
  accounts: [],
  tags: [],
  activeTag: '',
  searchQuery: '',
  editingId: null,
  scanStream: null,
  scanInterval: null
};

const listeners = [];

export function subscribe(fn) {
  listeners.push(fn);
  return () => {
    const idx = listeners.indexOf(fn);
    if (idx !== -1) listeners.splice(idx, 1);
  };
}

export function notifyStateChange() {
  for (const fn of listeners) {
    try {
      fn(state);
    } catch (e) {
      console.error('State listener error:', e);
    }
  }
}
