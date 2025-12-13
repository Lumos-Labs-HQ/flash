import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Flash ORM",
  description: "A powerful, database-agnostic ORM built in Go",
  base: '/flash/',
  vite: {
    plugins: [],
  },
  themeConfig: {
    nav: [
      { text: 'Home', link: '/' },
      {
        text: 'Docs',
        items: [
          { text: 'Getting Started', link: '/getting-started' },
          { text: 'Guides', link: '/guides/go' },
          { text: 'Concepts', link: '/concepts/schema' },
          { text: 'Reference', link: '/reference/cli' },
          { text: 'Advanced', link: '/advanced/how-it-works' }
        ]
      },
      { text: 'Studio', link: '/concepts/studio' }
    ],

    sidebar: {
      '/guides/': [
        {
          text: 'Language Guides',
          items: [
            { text: 'Go', link: '/guides/go' },
            { text: 'TypeScript/JavaScript', link: '/guides/typescript' },
            { text: 'Python', link: '/guides/python' }
          ]
        }
      ],
      '/concepts/': [
        {
          text: 'Core Concepts',
          items: [
            { text: 'Schema Definition', link: '/concepts/schema' },
            { text: 'Migrations', link: '/concepts/migrations' },
            { text: 'Code Generation', link: '/concepts/code-generation' },
            { text: 'FlashORM Studio', link: '/concepts/studio' },
            { text: 'Data Export', link: '/concepts/export' },
            { text: 'Branching', link: '/concepts/branching' }
          ]
        }
      ],
      '/reference/': [
        {
          text: 'CLI Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/cli' },
            { text: 'Configuration', link: '/reference/configuration' },
            { text: 'Schema Syntax', link: '/reference/schema' },
            { text: 'Query API', link: '/reference/query-api' }
          ]
        }
      ],
      '/advanced/': [
        {
          text: 'Advanced Topics',
          items: [
            { text: 'How It Works', link: '/advanced/how-it-works' },
            { text: 'Plugin System', link: '/advanced/plugins' },
            { text: 'Technology Stack', link: '/advanced/technology-stack' },
            { text: 'Performance', link: '/advanced/performance' }
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
    },

    editLink: {
      pattern: 'https://github.com/Lumos-Labs-HQ/flash-orm/edit/documentation/docs/:path',
      text: 'Edit this page on GitHub'
    },

    lastUpdated: {
      text: 'Last updated',
      formatOptions: {
        dateStyle: 'full',
        timeStyle: 'medium'
      }
    },

    returnToTopLabel: 'Return to top',
    sidebarMenuLabel: 'Menu',
    darkModeSwitchLabel: 'Appearance',
    darkModeSwitchTitle: 'Switch to dark mode',
    lightModeSwitchTitle: 'Switch to light mode'
  }
})
