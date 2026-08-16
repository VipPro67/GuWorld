module.exports = {
  darkMode: 'class',
  content: ['./src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: 'var(--bg-primary)',
        surface: 'var(--bg-surface)',
        elevated: 'var(--bg-elevated)',
        text: {
          primary: 'var(--text-primary)',
          secondary: 'var(--text-secondary)',
          muted: 'var(--text-muted)'
        },
        border: {
          DEFAULT: 'var(--border-default)',
          hover: 'var(--border-hover)'
        },
        brand: {
          DEFAULT: 'var(--brand-primary)',
          dim: 'var(--brand-dim)'
        },
        success: 'var(--success)',
        warning: 'var(--warning)',
        error: 'var(--error)',
        info: 'var(--info)'
      }
    }
  },
  plugins: []
};
