// @ts-check
import { defineConfig } from 'astro/config';

// Deployed to GitHub Pages under https://luchrv.github.io/lazyncu/
export default defineConfig({
  site: 'https://luchrv.github.io',
  base: '/lazyncu',
  output: 'static',
  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'es'],
  },
});
