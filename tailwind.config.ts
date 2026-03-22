import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        surface: '#1a1a2e',
        accent: '#6366f1',
      },
    },
  },
  plugins: [],
} satisfies Config
