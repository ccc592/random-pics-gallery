/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      colors: {
        sage: '#A8D5BA',
        cream: '#F8F6F0',
        coral: '#FFB3A7',
        charcoal: '#2D3748'
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif']
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-in-out',
        'scale-hover': 'scaleHover 0.2s ease-in-out'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        scaleHover: {
          '0%': { transform: 'scale(1)' },
          '100%': { transform: 'scale(1.02)' }
        }
      },
      transitionTimingFunction: {
        'out-cubic': 'cubic-bezier(0.4, 0.0, 0.2, 1)'
      }
    }
  },
  plugins: []
};