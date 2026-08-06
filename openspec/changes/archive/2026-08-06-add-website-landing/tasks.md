# Tasks: add-website-landing

## 1. Demo videos

- [x] 1.1 Add `mp4` and `webm` `Output` lines to the four tapes (`hero`, `vulns`, `add-path`, `select`) targeting `website/public/demos/`
- [x] 1.2 Run `make demos` and verify GIFs regenerate in `assets/demo/` and videos land in `website/public/demos/` (8 files)
- [x] 1.3 Update `docs/DEMOS.md` to document the video outputs and their website destination

## 2. Astro scaffold

- [x] 2.1 Scaffold Astro project in `website/` (static output, `site` + `base: '/lazyncu'`, i18n config `en` default / `es`, lockfile committed)
- [x] 2.2 Add `src/i18n/en.ts` and `src/i18n/es.ts` translation dictionaries with all landing copy
- [x] 2.3 Add base layout: dark terminal theme tokens, meta/OG tags, `lang` attribute, `hreflang` alternate links per locale

## 3. Landing content

- [x] 3.1 Hero section: product name, one-line value proposition, shields.io release badge, autoplaying muted looping hero video (`mp4`+`webm` sources)
- [x] 3.2 Features section: multi-path dashboard, semver-classified updates, vulnerability audit, update commands — with secondary demo videos
- [x] 3.3 Install section: copyable commands for Homebrew, `go install`, and release binaries download link
- [x] 3.4 Footer/header: GitHub repo link, license, language switcher (`/` ↔ `/es/`)
- [x] 3.5 Spanish page `src/pages/es/index.astro` rendering the same components with the `es` dictionary

## 4. Deploy

- [x] 4.1 Add `.github/workflows/pages.yml`: push-to-main with `website/**` paths filter + `workflow_dispatch`, configure-pages → npm build in `website/` → upload-pages-artifact → deploy-pages, `pages: write` + `id-token: write` permissions, `pages` concurrency group
- [x] 4.2 Enable GitHub Pages with source "GitHub Actions" in repo settings (done via gh api, build_type=workflow)

## 5. Verification

- [x] 5.1 Local check: `npm run build` in `website/` succeeds; `astro preview` — both locales render, videos autoplay, install commands copy, switcher and badge work with the `/lazyncu` base path
- [x] 5.2 Go toolchain untouched: `go vet ./...`, `go build ./...`, `go test ./...` all pass with no new files in the module
- [ ] 5.3 Post-merge smoke: Pages workflow green, `https://luchrv.github.io/lazyncu/` and `/es/` live with working videos (user)
- [x] 5.4 Update README with a link to the website
