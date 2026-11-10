// Countdown Loop & Circular Animation Module

import { state } from './state.js';
import { API } from './api.js';
import { renderAccounts } from './ui.js';
import { updateModalLivePreview } from './events.js';

let countdownTimer = null;
let lastStep = -1;
let lastPreviewStep = -1;
let lastQRStep = -1;

export function startCountdownLoop() {
  if (countdownTimer) clearInterval(countdownTimer);

  countdownTimer = setInterval(() => {
    const now = Math.floor(Date.now() / 1000);

    // 1. Dashboard accounts countdown (only when unlocked and has accounts)
    if (!state.locked && state.accounts && state.accounts.length > 0) {
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

      // Refresh dashboard codes when hitting period boundary
      const currentStep = Math.floor(now / 30);
      if (lastStep !== -1 && currentStep !== lastStep) {
        API.getAccounts(state.searchQuery, state.activeTag)
          .then(accs => {
            state.accounts = accs || [];
            renderAccounts();
          })
          .catch(() => {});
      }
      lastStep = currentStep;
    }

    // 2. Add / Edit Modal Live Preview Countdown & Auto Refresh
    const addModal = document.getElementById('addModal');
    if (addModal && addModal.classList.contains('active')) {
      const modalPreviewCircle = document.getElementById('modalPreviewCircle');
      const modalPreviewSeconds = document.getElementById('modalPreviewSeconds');
      const secretInput = document.getElementById('accSecret');
      const periodInput = document.getElementById('accPeriod');

      if (secretInput && secretInput.value.trim().length >= 4) {
        const period = parseInt(periodInput?.value) || 30;
        const remaining = period - (now % period);
        const progress = (now % period) / period;

        if (modalPreviewSeconds) {
          modalPreviewSeconds.textContent = `${remaining}s`;
        }
        if (modalPreviewCircle) {
          modalPreviewCircle.style.strokeDashoffset = 100.53 * progress;
          if (remaining <= 5) {
            modalPreviewCircle.style.stroke = 'var(--danger)';
          } else if (remaining <= 10) {
            modalPreviewCircle.style.stroke = 'var(--warning)';
          } else {
            modalPreviewCircle.style.stroke = 'var(--success)';
          }
        }

        // When period window rolls over, immediately fetch the new code!
        const currentPreviewStep = Math.floor(now / period);
        if (lastPreviewStep !== -1 && currentPreviewStep !== lastPreviewStep) {
          updateModalLivePreview();
        }
        lastPreviewStep = currentPreviewStep;
      }
    }

    // 3. QR Modal Live Code Countdown & Auto Refresh
    const qrModal = document.getElementById('qrModal');
    if (qrModal && qrModal.classList.contains('active')) {
      const qrCircle = document.getElementById('qrPreviewCircle');
      const qrSeconds = document.getElementById('qrPreviewSeconds');
      if (qrCircle && qrSeconds) {
        const remaining = 30 - (now % 30);
        const progress = (now % 30) / 30;
        qrSeconds.textContent = `${remaining}s`;
        qrCircle.style.strokeDashoffset = 100.53 * progress;
        if (remaining <= 5) {
          qrCircle.style.stroke = 'var(--danger)';
        } else if (remaining <= 10) {
          qrCircle.style.stroke = 'var(--warning)';
        } else {
          qrCircle.style.stroke = 'var(--success)';
        }

        const currentQRStep = Math.floor(now / 30);
        if (lastQRStep !== -1 && currentQRStep !== lastQRStep) {
          window.dispatchEvent(new CustomEvent('refreshQRModalCode'));
        }
        lastQRStep = currentQRStep;
      }
    }
  }, 1000);
}
