import { useTheme } from '../context/ThemeContext'

function LogoMark({ size = 28, className = '' }) {
  const { theme } = useTheme()
  const fill = theme === 'dark' ? '#ffffff' : '#000000'

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 128 128"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden="true"
    >
      <path
        fill={fill}
        d="M24 34V88c0 2.2 1.8 4 4 4h6c2.2 0 4-1.8 4-4V34H24zm56 0v54c0 2.2 1.8 4 4 4h6c2.2 0 4-1.8 4-4V34H80z"
      />
      <path fill={fill} d="M44 58h40v12H44V58z" />
      <path fill={fill} d="M56 42h16v16H56V42z" />
      <path fill={fill} d="M30 24h68v12H30V24z" />
      <path
        fill={fill}
        d="M20 92c8-10 18-14 30-14h38c12 0 22 4 30 14l-8 8c-6-8-14-12-24-12H52c-10 0-18 4-24 12l-8-8z"
      />
    </svg>
  )
}

export function BrandLogo({ size = 28, showWordmark = true, className = '' }) {
  return (
    <span className={`brand-logo ${className}`.trim()} style={{ display: 'inline-flex', alignItems: 'center', gap: '10px' }}>
      <LogoMark size={size} />
      {showWordmark && (
        <span
          className="brand-logo-text"
          style={{ fontSize: size >= 28 ? '18px' : '16px', fontWeight: 600, color: 'var(--text-primary)' }}
        >
          Hystersis
        </span>
      )}
    </span>
  )
}

export default BrandLogo
