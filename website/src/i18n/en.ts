import type { Dictionary } from './types';

export const en: Dictionary = {
  meta: {
    title: 'lazyncu — terminal dashboard for npm updates',
    description:
      'A read-only terminal dashboard for npm-check-updates. See which of your projects need updates — and how urgent they are — at a glance.',
  },
  nav: {
    github: 'GitHub',
    install: 'Install',
    languageSwitch: 'Español',
    languageSwitchHref: '/es/',
  },
  hero: {
    tagline: 'Which of my projects need updates, and how urgent are they?',
    subtitle:
      'lazyncu is a read-only terminal dashboard for npm-check-updates: it scans your global packages and every registered project in parallel, and answers that question at a glance.',
    ctaInstall: 'Install',
    ctaGithub: 'View on GitHub',
    videoFallback: 'Your browser does not support embedded videos.',
  },
  features: {
    heading: 'Features',
    items: [
      {
        title: 'One dashboard, all your projects',
        description:
          'Scans global packages and every registered path in parallel on launch. Auto-detects single projects, monorepos, or folders of projects — zero per-path configuration.',
        demo: 'select',
      },
      {
        title: 'Severity you can see',
        description:
          'Every upgrade is classified as major, minor, or patch with color coding and per-project counters. Suggestions respect each project’s engines.node, so you only see versions it can actually install.',
      },
      {
        title: 'Vulnerabilities, with the chain that drags them in',
        description:
          'npm audit / pnpm audit run alongside the version scan: severity counters, vulnerable-package detail, and the dependency chain behind each finding.',
        demo: 'vulns',
      },
      {
        title: 'Read-only, by design',
        description:
          'lazyncu never modifies anything. It shows the exact update or fix command for the current selection and copies it to your clipboard.',
        demo: 'add-path',
      },
    ],
  },
  install: {
    heading: 'Install',
    copyLabel: 'Copy',
    copiedLabel: 'Copied!',
    methods: [
      {
        label: 'Homebrew (macOS/Linux) — installs npm-check-updates automatically',
        command: 'brew install luchrv/tap/lazyncu',
      },
      {
        label: 'Go',
        command: 'go install github.com/luchrv/lazyncu@latest',
      },
    ],
    binaries: {
      label: 'Prebuilt binaries for Linux, macOS, and Windows (amd64/arm64):',
      linkText: 'releases page',
    },
  },
  footer: {
    license: 'MIT License',
    madeWith: 'Built with Go, Bubble Tea, and npm-check-updates.',
  },
};
