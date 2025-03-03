import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import react from '@astrojs/react';

// https://astro.build/config
export default defineConfig({
  outDir: './dist',
  site: 'https://docs.centralmind.ai',
  build: {
    assets: 'app_assets',
  },
  integrations: [
    starlight({
      title: 'CentralMind',
      logo: { dark: './src/assets/logo-dark.svg', light: './src/assets/logo-light.svg' },
      customCss: ['./src/styles/custom.css'],
      social: {
        github: 'https://github.com/centralmind/gateway',
      },
      sidebar: [
        { label: 'Introduction', slug: '' },
        {
          label: 'Getting Started',
          items: [
            { label: 'Installation', slug: 'docs/content/getting-started/installation' },
            { label: 'Generating an API', slug: 'docs/content/getting-started/generating-api' },
            { label: 'Launching an API', slug: 'docs/content/getting-started/launching-api' },
          ],
        },
        {
          label: 'Deployment',
          items: [
            { label: 'Docker Compose', link: '/example/simple' },
            { label: 'Plugin Integrations', link: '/example/complex' },
            { label: 'Kubernetes Example', link: '/example/k8s' },
            { label: 'Helm Installation', link: '/helm/gateway' },
          ],
        },
        {
          label: 'Integration',
          items: [
            { label: 'ChatGPT', slug: 'docs/content/integration/chatgpt' },
            { label: 'LangChain', slug: 'docs/content/integration/langchain' },
            { label: 'Claude Desktop', slug: 'docs/content/integration/claude-desktop' },
            { label: 'Local Running Models', slug: 'docs/content/integration/local-running-models' },
          ],
        },
        { label: 'CLI (Command Line Interface)', link: '/cli' },
        {
          label: 'Database Connectors',
          autogenerate: {
            directory: 'connectors',
          },
        },
        {
          label: 'Plugins',
          autogenerate: {
            directory: 'plugins',
          },
        },
        {
          label: 'Terms of Service',
          items: [
            { label: 'Terms of Service', slug: 'docs/content/legal/terms' },
            { label: 'Privacy Policy', slug: 'docs/content/legal/privacy' },
            { label: 'Cookie Policy', slug: 'docs/content/legal/cookie' },
          ],
        },
      ],
    }),
    react(),
  ],
});
