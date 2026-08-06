export interface FeatureCopy {
  title: string;
  description: string;
  /** Basename of the demo video under public/demos/, without extension. */
  demo?: 'hero' | 'vulns' | 'add-path' | 'select';
}

export interface InstallMethodCopy {
  label: string;
  command: string;
}

export interface Dictionary {
  meta: {
    title: string;
    description: string;
  };
  nav: {
    github: string;
    install: string;
    languageSwitch: string;
    languageSwitchHref: string;
  };
  hero: {
    tagline: string;
    subtitle: string;
    ctaInstall: string;
    ctaGithub: string;
    videoFallback: string;
  };
  features: {
    heading: string;
    items: FeatureCopy[];
  };
  install: {
    heading: string;
    copyLabel: string;
    copiedLabel: string;
    methods: InstallMethodCopy[];
    binaries: {
      label: string;
      linkText: string;
    };
  };
  footer: {
    license: string;
    madeWith: string;
  };
}
