/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      typography: {
        DEFAULT: {
          css: {
            table: {
              borderCollapse: 'collapse',
              width: '100%',
            },
            'th,td': {
              padding: '0.5rem',
              borderWidth: '1px',
              borderColor: 'var(--tw-prose-td-borders)',
            },
            th: {
              backgroundColor: 'var(--tw-prose-thead-bg)',
            },
          },
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}