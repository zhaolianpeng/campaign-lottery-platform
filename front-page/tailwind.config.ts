import type { Config } from 'tailwindcss';
import forms from '@tailwindcss/forms';
import typography from '@tailwindcss/typography';

const config: Config = {
  content: [
    './app/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#fff7ed',
          500: '#f97316',
          600: '#ea580c',
          900: '#7c2d12',
        },
      },
      boxShadow: {
        glow: '0 20px 60px rgba(249, 115, 22, 0.22)',
      },
    },
  },
  plugins: [forms, typography],
};

export default config;
