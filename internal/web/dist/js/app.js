// 2FA Vault - Main Application Entrypoint (ES Module)

import { setupEventListeners, refreshApp } from './events.js';
import { setupTheme } from './ui.js';
import { startCountdownLoop } from './countdown.js';

// Setup PWA Service Worker for offline capability
function setupPWA() {
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/service-worker.js').catch((err) => {
      console.warn('Service Worker registration skipped:', err);
    });
  }
}

// Boot application
async function boot() {
  setupTheme();
  setupEventListeners();
  setupPWA();
  await refreshApp();
  startCountdownLoop();
}

document.addEventListener('DOMContentLoaded', boot);
