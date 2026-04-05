import { mount } from 'svelte';
import App from './App.svelte';
import './app.css';

// Ensure dark theme is applied (mirrors the data-theme attribute on <html>)
document.documentElement.setAttribute('data-theme', 'dark');

function dismissBootShell() {
  const bootShell = document.getElementById('boot-shell');
  if (!bootShell || bootShell.classList.contains('boot-shell-hidden')) {
    return;
  }

  bootShell.classList.add('boot-shell-hidden');
  window.setTimeout(() => bootShell.remove(), 160);
}

const app = mount(App, {
  target: document.getElementById('app')!,
});

window.addEventListener('trace:app-ready', dismissBootShell, { once: true });
window.setTimeout(dismissBootShell, 5000);

export default app;
