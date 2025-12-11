import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Flash ORM",
  description: "A powerful, database-agnostic ORM built in Go",
  base: '/flash/', // Updated for the correct repo name
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guides', link: '/guides/getting-started' },
      { text: 'Reference', link: '/reference/commands' },
      { text: 'Advanced', link: '/advanced/architecture' }
    ],

    sidebar: {
      '/guides/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/guides/installation' },
            { text: 'Quick Start', link: '/guides/quick-start' },
            { text: 'Configuration', link: '/guides/configuration' }
          ]
        },
        {
          text: 'Language Guides',
          items: [
            { text: 'Go', link: '/guides/go' },
            { text: 'TypeScript/JavaScript', link: '/guides/typescript' },
            { text: 'Python', link: '/guides/python' }
          ]
        },
        {
          text: 'Database Support',
          items: [
            { text: 'PostgreSQL', link: '/guides/postgresql' },
            { text: 'MySQL', link: '/guides/mysql' },
            { text: 'SQLite', link: '/guides/sqlite' }
          ]
        }
      ],
      '/reference/': [
        {
          text: 'CLI Reference',
          items: [
            { text: 'Commands', link: '/reference/commands' },
            { text: 'Flags', link: '/reference/flags' },
            { text: 'Configuration', link: '/reference/configuration' }
          ]
        },
        {
          text: 'API Reference',
          items: [
            { text: 'Go API', link: '/reference/go-api' },
            { text: 'TypeScript API', link: '/reference/typescript-api' },
            { text: 'Python API', link: '/reference/python-api' }
          ]
        }
      ],
      '/advanced/': [
        {
          text: 'Advanced Topics',
          items: [
            { text: 'Architecture', link: '/advanced/architecture' },
            { text: 'Migration System', link: '/advanced/migrations' },
            { text: 'Code Generation', link: '/advanced/code-generation' },
            { text: 'Plugins', link: '/advanced/plugins' }
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/Lumos-Labs-HQ/flash' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024 Lumos Labs HQ'
    },

    search: {
      provider: 'local'
    }
  }
})
