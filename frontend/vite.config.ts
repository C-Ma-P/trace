import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { fileURLToPath } from 'url';
import { viteStaticCopy } from 'vite-plugin-static-copy';

const usePolling = process.platform === 'linux';

export default defineConfig(({ command }) => ({
  plugins: [
    svelte(),
    viteStaticCopy({
      targets: [
        {
          src: 'node_modules/occt-import-js/dist/occt-import-js.wasm',
          dest: '.',
        },
      ],
    }),
  ],
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
