import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { fileURLToPath } from 'url';

const usePolling = process.platform === 'linux';

export default defineConfig(({ command }) => ({
  plugins: [svelte()],
  resolve: {
    alias: {
      '$app/environment': fileURLToPath(
        new URL('./src/stubs/app-environment.ts', import.meta.url)
      ),
    },
  },
  optimizeDeps: {
    exclude: ['svelte-splitpanes'],
  },
  server: command === 'serve'
    ? {
        watch: {
          usePolling,
          interval: usePolling ? 120 : undefined,
          ignored: ['**/dist/**', '**/build/**', '**/bin/**'],
        },
      }
    : undefined,
}));
