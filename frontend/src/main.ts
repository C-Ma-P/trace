import { mount } from 'svelte';
import App from './App.svelte';
import './app.css';

// Timing relative to navigation start (performance.now() === 0 at page-load).
const fmtMs = () => `+${Math.round(performance.now())}ms`;

console.log(`[startup/frontend ${fmtMs()}] bootstrap`);

// Ensure dark theme is applied (mirrors the data-theme attribute on <html>)
document.documentElement.setAttribute('data-theme', 'dark');

function dismissBootShell() {
  const bootShell = document.getElementById('boot-shell');
  if (!bootShell || bootShell.classList.contains('boot-shell-hidden')) {
    return;
  }

  console.log(`[startup/frontend ${fmtMs()}] boot shell dismiss`);
  bootShell.classList.add('boot-shell-hidden');
  window.setTimeout(() => {
    bootShell.remove();
    console.log(`[startup/frontend ${fmtMs()}] boot shell removed`);
  }, 100);
}

console.log(`[startup/frontend ${fmtMs()}] before mount`);

const app = mount(App, {
  target: document.getElementById('app')!,
});

console.log(`[startup/frontend ${fmtMs()}] after mount`);

window.addEventListener(
  'trace:app-ready',
  () => {
    console.log(`[startup/frontend ${fmtMs()}] app-ready signal`);
    dismissBootShell();
  },
  { once: true },
);
window.setTimeout(dismissBootShell, 5000);

export default app;
