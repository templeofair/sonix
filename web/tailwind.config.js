/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // Semantic colour tokens — use these instead of ad-hoc greys in new UI.
        accent: { DEFAULT: '#5b7c99', light: '#7a9bb5' },
        surface: '#f8f9fa',
        card: '#ffffff',
        muted: { DEFAULT: '#6b7280', subtle: '#9ca3af' },
        border: { DEFAULT: '#e5e7eb', strong: '#d1d5db' },
        success: { DEFAULT: '#166534', soft: '#ecfdf5', border: '#a7f3d0' },
        warning: { DEFAULT: '#92400e', soft: '#fffbeb', border: '#fde68a' },
        danger: { DEFAULT: '#991b1b', soft: '#fef2f2', border: '#fecaca' },
      },
      borderRadius: {
        card: '0.75rem',
        btn: '0.5rem',
      },
      boxShadow: {
        card: '0 1px 3px 0 rgb(0 0 0 / 0.06), 0 1px 2px -1px rgb(0 0 0 / 0.06)',
        nav: '0 1px 0 0 rgb(0 0 0 / 0.06)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
