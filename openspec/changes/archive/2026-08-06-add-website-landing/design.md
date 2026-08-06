# Design: add-website-landing

## Context

See `proposal.md` — Why. Current state: the repo is a Go module with no web assets beyond README GIFs rendered by VHS tapes (`assets/tapes/*.tape` → `assets/demo/*.gif` via `make demos`). Four tapes exist: `hero`, `vulns`, `add-path`, `select`. CI is a single `release.yml` (goreleaser on `v*` tags). GitHub Pages is not yet enabled on the repo.

Constraints already locked (2026-08-06 exploration):
- Site lives in this repo at `website/` (`docs/` is taken by contributor docs; separate repo rejected — it would change the Pages URL).
- Astro, static output, landing page only.
- Bilingual EN (`/`) + ES (`/es/`), terminal aesthetic, dark by default.
- Videos re-rendered from tapes as `mp4`/`webm`, not converted from GIFs.
- Version shown via shields.io badge, no build-time injection.

## Goals / Non-Goals

**Goals:**
- One-page static site, zero runtime backend, fully self-hosted assets.
- Deployment fully automated after a one-time Pages settings toggle.
- Website toolchain (Node/Astro) isolated from the Go module — no impact on `go build`, `make check`, or goreleaser.

**Non-Goals:**
- Documentation site, blog, or multi-page content (landing only).
- Custom domain, analytics, or SEO beyond basic meta/`hreflang`.
- Light-theme design work beyond an acceptable default (dark is the design target).
- CI checks for the website beyond the Pages build itself.

## Decisions

### D1: Astro with static output and manual Pages workflow
Astro (`output: 'static'`, default) with `site: 'https://luchrv.github.io'` and `base: '/lazyncu'`. Deploy via the standard three-step Pages workflow (`actions/configure-pages` → build → `actions/upload-pages-artifact` + `actions/deploy-pages`) rather than `withastro/action`, keeping the workflow explicit and pin-able like the existing `release.yml`.
- *Alternative — plain HTML/CSS:* viable for one page, but Astro gives components shared across the two locales (hero, features, install) without duplicating markup, plus scoped CSS and asset hashing for free.
- *Alternative — Hugo:* Go-native tooling is appealing in a Go repo, but the team decision was Astro; its component model fits the per-locale duplication better.

### D2: i18n via Astro built-in routing + translation dictionaries
Use Astro's built-in i18n config (`defaultLocale: 'en'`, `locales: ['en','es']`, no prefix for default). Pages are thin: `src/pages/index.astro` and `src/pages/es/index.astro` both render the same `Landing` component tree, passing a locale; all strings live in `src/i18n/en.ts` / `es.ts` dictionaries. Guarantees structural equivalence between languages by construction; the switcher links `/lazyncu/` ↔ `/lazyncu/es/` and both pages emit `hreflang` alternates.
- *Alternative — i18n library (astro-i18next etc.):* dependency overhead for two locales on one page; YAGNI.

### D3: Videos rendered by tapes directly into `website/public/demos/`
VHS supports multiple `Output` lines per tape, including `.mp4` and `.webm`. Each tape gains two `Output` lines targeting `website/public/demos/<name>.mp4|.webm`; GIF outputs stay in `assets/demo/` for the README. `make demos` stays the single regeneration command — no copy step, no new target.
- *Alternative — output to `assets/demo/` and copy into the site at build time:* extra build plumbing for no benefit; `public/` is Astro's canonical passthrough directory.
- *Alternative — ffmpeg GIF→mp4 conversion:* explicitly rejected in exploration (quality; extra tool dependency).

### D4: Version badge is a plain shields.io `<img>`
`https://img.shields.io/github/v/release/luchrv/lazyncu` embedded as an image with fixed dimensions (no layout shift). This is the one exception to "fully self-hosted assets": accepted because it is exactly what keeps the version current without redeploys (see spec requirement), and it degrades to a broken-image-alt gracefully.

### D5: Workflow shape
`.github/workflows/pages.yml`: trigger `push` to `main` with `paths: ['website/**', '.github/workflows/pages.yml']` plus `workflow_dispatch`; permissions `pages: write`, `id-token: write`; concurrency group `pages`; Node LTS with npm cache scoped to `website/package-lock.json`; build runs in `working-directory: website`.

## Risks / Trade-offs

- [Committed videos bloat the repo] → keep demos short loops; webm/mp4 at terminal resolution are typically well under the GIFs they replace; four demos ≈ single-digit MB total. If it grows, revisit with Git LFS.
- [Pages not enabled → first deploy fails] → one-time manual step documented in tasks; workflow failure message is self-explanatory (`Pages site not found`).
- [Autoplay blocked by browsers] → videos are `muted playsinline loop`, which all modern browsers allow to autoplay; `controls` as fallback affordance.
- [`base: '/lazyncu'` path bugs (assets/links 404 on Pages but work locally)] → use Astro's `base`-aware URL helpers everywhere; verify with `astro preview` against the built output.
- [Node toolchain drift in a Go repo] → `website/` owns its own lockfile; nothing in the Go workflow references it; Pages workflow paths-filtered so Go PRs never wait on it.

## Migration Plan

1. Merge to `main` with Pages source set to "GitHub Actions" (repo settings, one-time, before or right after merge).
2. First workflow run publishes the site; rollback = revert the commit (site redeploys previous state) or disable Pages.
