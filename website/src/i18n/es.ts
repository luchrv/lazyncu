import type { Dictionary } from './types';

export const es: Dictionary = {
  meta: {
    title: 'lazyncu — dashboard de terminal para actualizaciones npm',
    description:
      'Un dashboard de terminal de solo lectura para npm-check-updates. Mira de un vistazo qué proyectos necesitan actualizaciones y qué tan urgentes son.',
  },
  nav: {
    github: 'GitHub',
    install: 'Instalar',
    languageSwitch: 'English',
    languageSwitchHref: '/',
  },
  hero: {
    tagline: '¿Cuáles de mis proyectos necesitan actualizaciones, y qué tan urgentes son?',
    subtitle:
      'lazyncu es un dashboard de terminal de solo lectura para npm-check-updates: escanea tus paquetes globales y cada proyecto registrado en paralelo, y responde esa pregunta de un vistazo.',
    ctaInstall: 'Instalar',
    ctaGithub: 'Ver en GitHub',
    videoFallback: 'Tu navegador no soporta videos embebidos.',
  },
  features: {
    heading: 'Características',
    items: [
      {
        title: 'Un dashboard, todos tus proyectos',
        description:
          'Escanea los paquetes globales y cada ruta registrada en paralelo al iniciar. Detecta automáticamente proyectos individuales, monorepos o carpetas de proyectos — cero configuración por ruta.',
        demo: 'select',
      },
      {
        title: 'Severidad a la vista',
        description:
          'Cada actualización se clasifica como major, minor o patch con colores y contadores por proyecto. Las sugerencias respetan el engines.node de cada proyecto: solo ves versiones que realmente puede instalar.',
      },
      {
        title: 'Vulnerabilidades, con la cadena que las arrastra',
        description:
          'npm audit / pnpm audit corren junto al escaneo de versiones: contadores por severidad, detalle del paquete vulnerable y la cadena de dependencias detrás de cada hallazgo.',
        demo: 'vulns',
      },
      {
        title: 'Solo lectura, por diseño',
        description:
          'lazyncu nunca modifica nada. Muestra el comando exacto de actualización o corrección para la selección actual y lo copia a tu portapapeles.',
        demo: 'add-path',
      },
    ],
  },
  install: {
    heading: 'Instalación',
    copyLabel: 'Copiar',
    copiedLabel: '¡Copiado!',
    methods: [
      {
        label: 'Homebrew (macOS/Linux) — instala npm-check-updates automáticamente',
        command: 'brew install luchrv/tap/lazyncu',
      },
      {
        label: 'Go',
        command: 'go install github.com/luchrv/lazyncu@latest',
      },
    ],
    binaries: {
      label: 'Binarios precompilados para Linux, macOS y Windows (amd64/arm64):',
      linkText: 'página de releases',
    },
  },
  footer: {
    license: 'Licencia MIT',
    madeWith: 'Hecho con Go, Bubble Tea y npm-check-updates.',
  },
};
