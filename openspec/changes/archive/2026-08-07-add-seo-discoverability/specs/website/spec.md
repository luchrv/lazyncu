## ADDED Requirements

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
