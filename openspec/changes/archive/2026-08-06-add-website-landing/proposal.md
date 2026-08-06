# Proposal: add-website-landing

## Why

lazyncu has no web presence beyond the GitHub README: there is no page to link from social posts, package listings, or search results, and the README cannot show autoplaying video demos or serve a Spanish-speaking audience. A landing page gives the project a shareable home at `https://luchrv.github.io/lazyncu` that demonstrates the TUI visually and funnels visitors to installation.

## What Changes

- Add a static landing page under `website/` in this repository, built with Astro.
- Bilingual content: English at `/` and Spanish at `/es/`, with a language switcher.
- Terminal-aesthetic design, dark theme by default.
- Demo videos rendered from the existing VHS tapes as `mp4`/`webm` (not converted from the README GIFs) and embedded in the page.
- Current release version displayed via a shields.io badge (no build-time version injection).
- New GitHub Actions workflow `.github/workflows/pages.yml` deploying the site to GitHub Pages on push to `main` with a `website/**` paths filter; Pages source set to GitHub Actions.
- Extend the demo tooling so `make demos` can also render the video variants used by the website.

## Capabilities

### New Capabilities

- `website`: the public landing page — content structure, bilingual routing, terminal aesthetic, embedded video demos, install instructions, version badge, and the GitHub Pages deployment workflow.

### Modified Capabilities

- `demo-recording`: demo tapes additionally render `mp4`/`webm` video outputs consumed by the website, alongside the existing README GIFs.

## Impact

- New directory `website/` (Astro project with its own `package.json`; not part of the Go module or its build).
- New workflow `.github/workflows/pages.yml`; existing `release.yml` untouched.
- `Makefile` demo targets and `assets/tapes/` outputs extended for video rendering.
- Repository settings: GitHub Pages must be enabled with source "GitHub Actions" (one-time manual step).
- No changes to the Go CLI, its behavior, or the release pipeline.
