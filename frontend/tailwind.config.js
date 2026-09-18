/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        background: '#090d16',
        surface: '#111827',
        surfaceLight: '#1f2937',
        accent: '#3b82f6',
        accentGlow: '#60a5fa',
        neonPurple: '#a855f7',
        neonCyan: '#06b6d4',
        neonPink: '#ec4899',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'glow': 'glow 2s ease-in-out infinite alternate',
      },
      keyframes: {
        glow: {
          '0%': { boxShadow: '0 0 10px rgba(59, 130, 246, 0.5)' },
          '100%': { boxShadow: '0 0 25px rgba(168, 85, 247, 0.8)' },
        },
      },
    },
  },
  plugins: [],
}
