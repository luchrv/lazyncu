# website Specification

## Purpose
Public landing page for lazyncu: a static, bilingual site that shows the TUI in action through video demos, presents the key features, and funnels visitors to installation — deployed automatically to GitHub Pages from this repository.
## Requirements
### Requirement: Landing page is published on GitHub Pages
The repository SHALL contain a static site under `website/` that is built and deployed to GitHub Pages at `https://luchrv.github.io/lazyncu/` by a dedicated GitHub Actions workflow. The workflow SHALL run on pushes to `main` that touch `website/**` and SHALL be manually triggerable. The release workflow SHALL remain unaffected.

#### Scenario: Website change lands on main
- **WHEN** a commit touching `website/` is pushed to `main`
- **THEN** the Pages workflow builds the site and the updated page is served at `https://luchrv.github.io/lazyncu/`

#### Scenario: Non-website change lands on main
- **WHEN** a commit touching only Go sources is pushed to `main`
- **THEN** the Pages workflow does not run

### Requirement: Landing page is bilingual with English default
The site SHALL serve English content at the root path (`/`) and Spanish content at `/es/`, with equivalent structure and meaning in both languages. Every page SHALL offer a visible language switcher linking to its counterpart, and each language version SHALL declare the proper `lang` attribute and `hreflang` alternate links.

#### Scenario: Visitor switches to Spanish
- **WHEN** a visitor on the English landing page activates the language switcher
- **THEN** the Spanish version of the same page loads at `/es/`

#### Scenario: Search engine crawls the site
- **WHEN** a crawler fetches either language version
- **THEN** the page declares its language and links the alternate language via `hreflang`

### Requirement: Landing page presents the product with a terminal aesthetic
The landing page SHALL use a terminal-inspired dark visual theme by default and SHALL include: a hero section identifying lazyncu with a primary demo video, a features section covering the core capabilities (multi-path dashboard, semver-classified updates, vulnerability audit, update commands), an installation section with copyable commands for Homebrew, `go install`, and release binaries, and links to the GitHub repository.

#### Scenario: Visitor lands on the page
- **WHEN** a visitor opens the landing page
- **THEN** they see a dark terminal-styled hero with the product name, a one-line value proposition, a playing demo, and a visible path to installation instructions

#### Scenario: Visitor copies an install command
- **WHEN** a visitor uses the copy control on an install command
- **THEN** the exact command text is placed on their clipboard

### Requirement: Demos are embedded as native video
The landing page SHALL embed the demo recordings as `mp4`/`webm` video elements sourced from files committed in the repository — not as animated GIFs and not hotlinked from external hosts. Demo videos SHALL autoplay muted, loop, and play inline.

#### Scenario: Visitor views the hero demo
- **WHEN** the landing page loads in a modern browser
- **THEN** the hero demo video plays automatically, muted and looping, without visitor interaction

### Requirement: Displayed release version is always current
The landing page SHALL display the latest lazyncu release version via a dynamic badge (shields.io GitHub release badge) so the shown version updates without rebuilding or redeploying the site.

#### Scenario: New release is published
- **WHEN** a new version tag is released and the site has not been redeployed
- **THEN** the landing page badge shows the new version


### Requirement: Site is discoverable by search engines
The site SHALL expose the standard search-engine discovery artifacts: a build-time-generated XML sitemap listing every published page in both languages, a `robots.txt` at the site root that allows all crawlers and declares the absolute URL of the sitemap index, and a Google Search Console site-verification meta tag in the `<head>` of every page. All URLs in these artifacts SHALL be absolute and use the canonical site origin and base path.

#### Scenario: Crawler fetches robots.txt
- **WHEN** a crawler requests `https://luchrv.github.io/lazyncu/robots.txt`
- **THEN** it receives a policy that allows crawling and a `Sitemap:` line with the absolute sitemap index URL

#### Scenario: Crawler fetches the sitemap
- **WHEN** a crawler requests the sitemap index URL declared in `robots.txt`
- **THEN** it receives a valid XML sitemap whose entries resolve to the English (`/`) and Spanish (`/es/`) pages with absolute URLs

#### Scenario: Search Console verifies ownership
- **WHEN** Google Search Console fetches any page of the site during verification
- **THEN** the page `<head>` contains the `google-site-verification` meta tag with the owner's token
