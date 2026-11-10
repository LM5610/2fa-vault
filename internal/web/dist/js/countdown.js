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

      // If addModal is open and has secret, refresh preview code
      const addModal = document.getElementById('addModal');
      if (addModal && addModal.classList.contains('active')) {
        const secretInput = document.getElementById('accSecret');
        if (secretInput && secretInput.value.trim().length >= 4) {
          window.dispatchEvent(new CustomEvent('refreshModalPreview'));
        }
      }
    }

    // Smoothly tick Add Modal preview countdown if active
    const modalPreviewCircle = document.getElementById('modalPreviewCircle');
    const modalPreviewSeconds = document.getElementById('modalPreviewSeconds');
    const secretInput = document.getElementById('accSecret');
    const periodInput = document.getElementById('accPeriod');
    if (modalPreviewCircle && modalPreviewSeconds && secretInput && secretInput.value.trim().length >= 4) {
      const period = parseInt(periodInput?.value) || 30;
      const rem = period - (now % period);
      const prog = (now % period) / period;
      modalPreviewSeconds.textContent = `${rem}s`;
      modalPreviewCircle.style.strokeDashoffset = 100.53 * prog;
      if (rem <= 5) modalPreviewCircle.style.stroke = 'var(--danger)';
      else if (rem <= 10) modalPreviewCircle.style.stroke = 'var(--warning)';
      else modalPreviewCircle.style.stroke = 'var(--success)';
    }

    // Smoothly tick QR Modal preview countdown if active
    const qrModal = document.getElementById('qrModal');
    if (qrModal && qrModal.classList.contains('active')) {
      const qrCircle = document.getElementById('qrPreviewCircle');
      const qrSeconds = document.getElementById('qrPreviewSeconds');
      if (qrCircle && qrSeconds) {
        const rem = 30 - (now % 30);
        const prog = (now % 30) / 30;
        qrSeconds.textContent = `${rem}s`;
        qrCircle.style.strokeDashoffset = 100.53 * prog;
        if (rem <= 5) qrCircle.style.stroke = 'var(--danger)';
        else if (rem <= 10) qrCircle.style.stroke = 'var(--warning)';
        else qrCircle.style.stroke = 'var(--success)';
      }
    }

    lastSecond = currentSecond;
  }, 1000);
}
