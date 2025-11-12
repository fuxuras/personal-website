// @ts-check
import { defineConfig, passthroughImageService } from 'astro/config';
import icon from 'astro-icon';
import sitemap from '@astrojs/sitemap';

import tailwindcss from '@tailwindcss/vite';

import react from '@astrojs/react';

// https://astro.build/config
export default defineConfig({
  site: 'https://fuxuras.dev',
  integrations: [icon(), react(), sitemap()],

  vite: {
    plugins: [tailwindcss()]
  },

  image: {
    service: passthroughImageService()
  }
});