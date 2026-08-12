/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#F4F7FE',
          100: '#E9EDF7',
          200: '#D5DFED',
          300: '#A3AED0', // Secondary text
          400: '#707EAE', // Muted text
          500: '#4318FF', // Primary blue
          600: '#3911D4', 
          700: '#2B3674', // Heading color light mode
          800: '#111C44', // Card dark mode
          900: '#0B1437', // BG dark mode
        },
      },
      boxShadow: {
        'soft': '0px 18px 40px rgba(112, 144, 176, 0.12)',
        'soft-dark': '0px 18px 40px rgba(0, 0, 0, 0.25)',
      },
      fontFamily: {
        sans: ['Plus Jakarta Sans', 'Inter', 'sans-serif'],
      }
    },
  },
  plugins: [],
}
