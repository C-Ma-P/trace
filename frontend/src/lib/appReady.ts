import { tick } from 'svelte';

export async function notifyAppReady() {
  // Flush any pending Svelte rune state updates.
  await tick();
  // Two animation frames: the first lets legacy-store subscribers (writable/derived)
  // finish their DOM mutations; the second ensures that paint has completed so the
  // content behind the boot shell is actually visible before we start the fade.
  await new Promise<void>((resolve) =>
    window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve())),
  );
  window.dispatchEvent(new Event('trace:app-ready'));
}