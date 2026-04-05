import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { fileURLToPath } from 'url';

export default defineConfig({
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
});
