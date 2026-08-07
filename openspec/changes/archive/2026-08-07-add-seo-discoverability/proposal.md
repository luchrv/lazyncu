# Proposal: add-seo-discoverability

## Why

Searching "lazyncu" on Google returns zero results for the project (checked 2026-08-07): the site went live one day ago, has no sitemap or robots.txt, and no property registered in Google Search Console, so search engines have no discovery signal for it. Without these, indexing of a GitHub Pages subpath with near-zero backlinks can take weeks or never happen.

## What Changes

- Add the Google Search Console verification meta tag to the site `<head>` (token already issued by the owner).
- Generate a sitemap at build time via the `@astrojs/sitemap` integration, covering both language versions.
- Add a `robots.txt` that allows all crawlers and points to the sitemap index.
- No content, layout, or promotion changes — technical SEO only (scope locked with the owner: no external backlink work).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `website`: new requirement — the site must be discoverable by search engines (verification meta tag, build-time sitemap, robots.txt referencing the sitemap).

## Impact

- `website/src/layouts/Base.astro`: one `<meta name="google-site-verification">` line in `<head>`.
- `website/astro.config.mjs`: add `@astrojs/sitemap` integration.
- `website/package.json` / lockfile: new dev dependency `@astrojs/sitemap`.
- `website/public/robots.txt`: new file.
- No Go code, no release pipeline, no Pages workflow changes (existing workflow already builds `website/**`).
- Post-merge manual steps (owner, outside repo): verify the property in Search Console, submit `sitemap-index.xml`, request indexing of `/` and `/es/`.
