import { DASHBOARD_SIGNIN_URL, DASHBOARD_SIGNUP_URL } from '../constants'

export function DashboardAuthLinks({ variant = 'navbar' }) {
  if (variant === 'navbar') {
    return (
      <a href={DASHBOARD_SIGNIN_URL} className="nav-cta nav-signin">
        Sign In
      </a>
    )
  }

  if (variant === 'mobile') {
    return (
      <div className="mobile-auth-links">
        <a href={DASHBOARD_SIGNIN_URL} className="mobile-cta">
          Sign In
        </a>
        <a href={DASHBOARD_SIGNUP_URL} className="mobile-link-external">
          Create Account
        </a>
      </div>
    )
  }

  return (
    <a href={DASHBOARD_SIGNIN_URL} className="nav-cta">
      Sign In
    </a>
  )
}
