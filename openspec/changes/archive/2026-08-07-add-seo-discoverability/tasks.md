# Tasks: add-seo-discoverability

## 1. Sitemap

- [x] 1.1 Add `@astrojs/sitemap` via `npx astro add sitemap` in `website/` and configure the `i18n` option (`defaultLocale: 'en'`, locales `en`/`es`) in `astro.config.mjs`
- [x] 1.2 Build the site and verify `dist/sitemap-index.xml` exists and `dist/sitemap-0.xml` lists both `https://luchrv.github.io/lazyncu/` and `https://luchrv.github.io/lazyncu/es/` with hreflang alternates

## 2. robots.txt

- [x] 2.1 Create `website/public/robots.txt` allowing all crawlers with `Sitemap: https://luchrv.github.io/lazyncu/sitemap-index.xml`
- [x] 2.2 Build and verify `dist/robots.txt` is emitted verbatim

## 3. Search Console verification tag

- [x] 3.1 Add `<meta name="google-site-verification" content="1S6KXVsP3OOR7Ira87pAqFJdq5udANZewR753rL3Uow" />` to the `<head>` in `website/src/layouts/Base.astro`
- [x] 3.2 Build and verify the tag is present in both `dist/index.html` and `dist/es/index.html`

## 4. Ship and verify live

- [x] 4.1 Commit on a feature branch, open PR, merge to `main`, confirm the Pages workflow deploys
- [x] 4.2 Smoke live URLs: `curl` the deployed `robots.txt`, `sitemap-index.xml`, and `/` for the verification tag
- [x] 4.3 Owner (manual, outside repo): verify the GSC property, submit `sitemap-index.xml`, request indexing of `/` and `/es/`
