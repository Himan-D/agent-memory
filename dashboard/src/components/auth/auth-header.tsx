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
          <circle cx="64" cy="64" r="12" fill="url(#logoGradAuth)"/>
          <circle cx="32" cy="40" r="7" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="96" cy="40" r="7" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="32" cy="88" r="7" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="96" cy="88" r="7" fill="url(#logoGradAuth)" opacity="0.9"/>
          <circle cx="64" cy="24" r="5" fill="url(#logoGradAuth)" opacity="0.7"/>
          <circle cx="64" cy="104" r="5" fill="url(#logoGradAuth)" opacity="0.7"/>
          <line x1="64" y1="52" x2="32" y2="40" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
          <line x1="64" y1="52" x2="96" y2="40" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
          <line x1="64" y1="52" x2="32" y2="88" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
          <line x1="64" y1="52" x2="96" y2="88" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
          <line x1="64" y1="52" x2="64" y2="24" stroke="#6366f1" strokeWidth="2" opacity="0.4"/>
          <line x1="64" y1="76" x2="64" y2="104" stroke="#6366f1" strokeWidth="2" opacity="0.4"/>
        </svg>
      </div>
      <span className="text-2xl font-bold">Hystersis</span>
    </div>
  );
}