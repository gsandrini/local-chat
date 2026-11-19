/** @type {import('tailwindcss').Config} */
module.exports = {
  // Tailwind scans these files to find which classes are actually used
  // and removes everything else from the output CSS
  content: [
    './index.html',
    './assets/js/**/*.js',
  ],
  // `dark:` variants follow the same data-theme attribute as the CSS tokens
  darkMode: ['selector', '[data-theme="dark"]'],
  theme: {
    extend: {
      fontSize: {
        sm: '13px',
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
};
