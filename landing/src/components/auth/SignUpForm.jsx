import { DASHBOARD_SIGNIN_URL, DASHBOARD_SIGNUP_URL } from '../../constants'

function SignUpForm() {
  return (
    <section className="auth-redirect-card" aria-labelledby="signup-title">
      <p className="auth-redirect-eyebrow">Hystersis dashboard</p>
      <h1 id="signup-title">Create your account in the dashboard</h1>
      <p>
        Account creation is handled by the secure dashboard app so your API keys,
        workspace settings, and memory data stay in one place.
      </p>
      <div className="auth-redirect-actions">
        <a href={DASHBOARD_SIGNUP_URL} className="btn btn-primary">
          Create account
        </a>
        <a href={DASHBOARD_SIGNIN_URL} className="btn btn-secondary">
          Sign in
        </a>
      </div>
    </section>
  )
}

export default SignUpForm
