import { createApp } from 'vue';
import App from './App.vue';
import './style.css';

const app = createApp(App);
app.mount('#app');

// Register Service Worker for PWA (iOS Home Screen & Offline caching)
if ('serviceWorker' in navigator && import.meta.env.PROD) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/service-worker.js').catch((err) => {
      console.warn('ServiceWorker registration failed: ', err);
    });
  });
}

