import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Open Prompt',
  description: 'Desktop AI assistant — đa provider, đa ngôn ngữ',
  lang: 'vi-VN',
  themeConfig: {
    nav: [
      { text: 'Hướng dẫn', link: '/guide/getting-started' },
      { text: 'API', link: '/api/engine-rpc' },
      { text: 'GitHub', link: 'https://github.com/minhtuancn/open-prompt' },
    ],
    sidebar: [
      {
        text: 'Hướng dẫn',
        items: [
          { text: 'Bắt đầu', link: '/guide/getting-started' },
          { text: 'Providers', link: '/guide/providers' },
          { text: 'Phím tắt', link: '/guide/hotkeys' },
          { text: 'Plugins', link: '/guide/plugins' },
        ],
      },
      {
        text: 'API Reference',
        items: [
          { text: 'Engine RPC', link: '/api/engine-rpc' },
        ],
      },
    ],
    footer: {
      message: 'Released under MIT License',
    },
  },
})
