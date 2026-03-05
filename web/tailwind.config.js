/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'claw-dark': '#0a0e27',
        'claw-darker': '#050814',
        'claw-blue': '#3b82f6',
        'claw-green': '#10b981',
        'claw-red': '#ef4444',
        'claw-yellow': '#f59e0b',
      },
    },
  },
  plugins: [],
}
