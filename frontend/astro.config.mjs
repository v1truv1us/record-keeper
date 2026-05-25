// @ts-check
import { defineConfig } from 'astro/config';

import svelte from '@astrojs/svelte';
import tailwindcss from '@tailwindcss/vite';
import sentry from '@sentry/astro';

// https://astro.build/config
export default defineConfig({
  integrations: [
    svelte(),
    sentry({
      dsn: process.env.PUBLIC_SENTRY_DSN,
      environment: process.env.PUBLIC_ENVIRONMENT || 'development',
      release: 'audiofile@0.2.0',
      tracesSampleRate: 0.2,
      replaysSessionSampleRate: 0,
      replaysOnErrorSampleRate: 1.0,
    }),
  ],
  server: {
    host: '0.0.0.0',
    port: 4321,
  },
  vite: {
    plugins: [tailwindcss()],
    cacheDir: '/tmp/vite-cache-build'
  }
});