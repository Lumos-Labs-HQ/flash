import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Flash ORM",
  description: "A powerful, database-agnostic ORM built in Go with Prisma-like functionality",
  
  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }],
  ],

  themeConfig: {
    logo: '/logo.png',
    
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Getting Started', link: '/getting-started' },
      { 
        text: 'Guides', 
        items: [
          { text: 'Go Guide', link: '/guides/go' },
          { text: 'TypeScript Guide', link: '/guides/typescript' },
          { text: 'Python Guide', link: '/guides/python' },
        ]
      },
      { 
        text: 'Reference', 
        items: [
          { text: 'CLI Commands', link: '/reference/cli' },
          { text: 'Configuration', link: '/reference/configuration' },
          { text: 'Schema Definition', link: '/reference/schema' },
        ]
      },
      { 
        text: 'Advanced',   
        items: [
          { text: 'How It Works', link: '/advanced/how-it-works' },
          { text: 'Plugin System', link: '/advanced/plugins' },
          { text: 'Technology Stack', link: '/advanced/technology-stack' },
        ]
      },
      {
        text: 'v1.0.0',
        items: [
          { text: 'Release Notes', link: '/releases' },
          { text: 'Contributing', link: '/contributing' },
        ]
      }
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'What is Flash ORM?', link: '/introduction/what-is-flash' },
          { text: 'Why Flash ORM?', link: '/introduction/why-flash' },
          { text: 'Quick Start', link: '/getting-started' },
        ]
      },
      {
        text: 'Guides',
        items: [
          { text: 'Go', link: '/guides/go' },
          { text: 'TypeScript/JavaScript', link: '/guides/typescript' },
          { text: 'Python', link: '/guides/python' },
        ]
      },
      {
        text: 'Core Concepts',
        items: [
          { text: 'Schema Definition', link: '/concepts/schema' },
          { text: 'Migrations', link: '/concepts/migrations' },
          { text: 'Code Generation', link: '/concepts/code-generation' },
          { text: 'Database Studio', link: '/concepts/studio' },
          { text: 'Data Export', link: '/concepts/export' },
          { text: 'Branching', link: '/concepts/branching' },
        ]
      },
      {
        text: 'Database Support',
        items: [
          { text: 'PostgreSQL', link: '/databases/postgresql' },
          { text: 'MySQL', link: '/databases/mysql' },
          { text: 'SQLite', link: '/databases/sqlite' },
          { text: 'MongoDB', link: '/databases/mongodb' },
        ]
      },
      {
        text: 'Reference',
        items: [
          { text: 'CLI Commands', link: '/reference/cli' },
          { text: 'Configuration File', link: '/reference/configuration' },
          { text: 'Schema Syntax', link: '/reference/schema' },
          { text: 'Query API', link: '/reference/query-api' },
        ]
      },
      {
        text: 'Advanced',
        items: [
          { text: 'How It Works', link: '/advanced/how-it-works' },
          { text: 'Plugin System', link: '/advanced/plugins' },
          { text: 'Technology Stack', link: '/advanced/technology-stack' },
          { text: 'Performance', link: '/advanced/performance' },
        ]
      },
      {
        text: 'Community',
        items: [
          { text: 'Contributing', link: '/contributing' },
          { text: 'Release Notes', link: '/releases' },
          { text: 'Beta Workflows', link: '/beta-workflows' },
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/Lumos-Labs-HQ/flash' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024-present Lumos Labs'
    },

    search: {
      provider: 'local'
    },

    editLink: {
      pattern: 'https://github.com/Lumos-Labs-HQ/flash/edit/main/docs/:path',
      text: 'Edit this page on GitHub'
    }
  }
})
