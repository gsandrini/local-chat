/** @type {import('tailwindcss').Config} */
module.exports = {
  // Tailwind scans these files to find which classes are actually used
  // and removes everything else from the output CSS
  content: [
    './index.html',
    './assets/js/**/*.js',
  ],
  darkMode: 'media',
  theme: {
    extend: {
      fontFamily: {
        mono: ['"JetBrains Mono"', 'monospace'],
      },
      colors: {
        ink:  '#0f0f0f',
        snow: '#f7f7f5',
        dim:  '#6b6b68',
      },
    },
  },
  plugins: [],
};
