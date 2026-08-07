# Design: add-seo-discoverability

## Context

See proposal.md — Why. Current state: Astro 7 static site under `website/`, `site: 'https://luchrv.github.io'` + `base: '/lazyncu'` in `astro.config.mjs`, shared `<head>` in `src/layouts/Base.astro` (already has canonical + hreflang + og tags), static assets served from `website/public/`, Pages deploy via the existing workflow on `main` pushes touching `website/**`. The GSC verification token is already issued: `1S6KXVsP3OOR7Ira87pAqFJdq5udANZewR753rL3Uow` (verification tokens are public by design — they appear in the served HTML of every verified site).

## Goals / Non-Goals

**Goals:**
- Every published page carries the GSC verification meta tag.
- Sitemap generated at build time — never hand-maintained.
- `robots.txt` served at the site root with the absolute sitemap index URL.

**Non-Goals:**
- External promotion / backlinks (explicitly descoped by owner).
- Structured data (JSON-LD), Open Graph images, or content changes.
- Bing or other webmaster-tools registration (owner may import from GSC later; no repo change needed).
- Automating the Search Console steps themselves (property registration, sitemap submission, indexing requests are manual owner actions).

## Decisions

**D1 — `@astrojs/sitemap` integration over a hand-written sitemap.**
The official integration walks all rendered routes at build time, honors `site` + `base`, and emits `sitemap-index.xml` + `sitemap-0.xml` into `dist/`. A static file in `public/` would go stale the moment a page is added and would duplicate the i18n URL logic. Install with `npx astro add sitemap` (adds the dependency and the `integrations: [sitemap()]` entry in one step); pass the `i18n` option (`defaultLocale: 'en'`, `locales: {en: 'en-US', es: 'es-ES'}`) so entries carry `xhtml:link` hreflang alternates, matching the hreflang tags already in `Base.astro`.

**D2 — static `robots.txt` in `website/public/` over a generated one.**
Content is two lines and changes never; a `robots.txt.ts` endpoint would be machinery for nothing (YAGNI). The `Sitemap:` line must be the absolute URL `https://luchrv.github.io/lazyncu/sitemap-index.xml` because relative sitemap references are not part of the robots.txt protocol. Note the file lands at `/lazyncu/robots.txt`, not the origin root — crawlers resolve robots.txt per-origin, but GSC's URL-prefix property and the sitemap discovery line both work at the subpath, which is the standard GitHub Pages project-site situation and explicitly acceptable here.

**D3 — verification meta tag hardcoded in `Base.astro`.**
One line next to the existing meta tags. An env var or config indirection adds a secret-management smell to a value that is intentionally public and permanent. `Base.astro` is the single shared layout, so every page (`/` and `/es/`) gets it — GSC only needs it on `/`, but blanket coverage is free and harmless.

## Risks / Trade-offs

- [robots.txt at `/lazyncu/robots.txt` is not at the origin root, so generic crawlers fetching `luchrv.github.io/robots.txt` won't see it] → GSC discovery does not depend on it: the sitemap is submitted manually in Search Console, and the `Sitemap:` line is a redundancy, not the primary channel.
- [Sitemap URLs wrong if `site`/`base` ever change] → integration derives everything from `astro.config.mjs`; a build-output check is part of the tasks.
- [Indexing still takes days after merge] → expected; owner requests indexing manually per the agreed runbook (guide already delivered in conversation).

## Migration Plan

Ship in one PR; Pages workflow deploys on merge. No rollback concerns — all three artifacts are additive and inert for existing visitors. Post-merge owner runbook (outside repo): verify property → submit `sitemap-index.xml` → request indexing of `/` and `/es/`.
