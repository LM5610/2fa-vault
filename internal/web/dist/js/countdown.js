// Countdown Loop & Circular Animation Module

import { state } from './state.js';
import { API } from './api.js';
import { renderAccounts } from './ui.js';

let countdownTimer = null;
let lastSecond = -1;

export function startCountdownLoop() {
  if (countdownTimer) clearInterval(countdownTimer);

  countdownTimer = setInterval(() => {
    if (state.locked || !state.accounts || state.accounts.length === 0) return;

    const now = Math.floor(Date.now() / 1000);
    const currentSecond = now % 30;

    // Update circular meters smoothly
    document.querySelectorAll('.account-card').forEach(card => {
      const id = card.getAttribute('data-id');
      const acc = state.accounts.find(a => a.id === id);
      if (!acc || acc.type === 'hotp') return;

      const period = acc.period || 30;
      const remaining = period - (now % period);
      const progress = (now % period) / period;
      const dashoffset = 100.53 * progress;

      const circle = card.querySelector('.timer-circle');
      const text = card.querySelector('.timer-text');

      if (circle) {
        circle.style.strokeDashoffset = dashoffset;
        if (remaining <= 5) {
          circle.style.stroke = 'var(--danger)';
        } else if (remaining <= 10) {
          circle.style.stroke = 'var(--warning)';
        } else {
          circle.style.stroke = 'var(--success)';
        }
      }
      if (text) {
        text.textContent = `${remaining}s`;
      }
    });

    // Refresh codes when hitting period boundary
    if (currentSecond === 0 && lastSecond !== 0) {
      API.getAccounts(state.searchQuery, state.activeTag)
        .then(accs => {
          state.accounts = accs || [];
          renderAccounts();
        })
        .catch(() => {});
    }
    lastSecond = currentSecond;
  }, 1000);
}
