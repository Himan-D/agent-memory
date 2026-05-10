// AUTH_PAGE: Reusable logo header for all authentication pages
// Ensures consistent branding across signin, signup, error pages
export function AuthHeader() {
  return (
    <div className="flex items-center justify-center gap-3 mb-8">
      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary">
        <svg viewBox="0 0 128 128" className="h-7 w-7" fill="none" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <linearGradient id="logoGradAuth" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#6366f1"/>
              <stop offset="100%" stopColor="#8b5cf6"/>
            </linearGradient>
          </defs>
          <circle cx="64" cy="64" r="10" fill="url(#logoGradAuth)"/>
          <circle cx="36" cy="44" r="5" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="92" cy="44" r="5" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="36" cy="84" r="5" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="92" cy="84" r="5" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="64" cy="30" r="3" fill="url(#logoGradAuth)" opacity="0.7"/>
          <circle cx="64" cy="98" r="3" fill="url(#logoGradAuth)" opacity="0.7"/>
        </svg>
      </div>
      <span className="text-2xl font-bold">Hystersis</span>
    </div>
  );
}