import { useEffect } from 'react'

/**
 * Redirect /docs to Mintlify documentation.
 * Production uses workers/site.js proxy; this handles client-side navigation.
 */
function DocsPage() {
  useEffect(() => {
    const path = window.location.pathname.replace(/^\/docs/, '') || '/'
    const target = `https://docs.hystersis.com${path}${window.location.search}${window.location.hash}`
    window.location.replace(target)
  }, [])

  return (
    <div className="docs-redirect">
      <p>Redirecting to documentation…</p>
      <p>
        <a href="https://docs.hystersis.com">Open docs.hystersis.com</a>
      </p>
      <style>{`
        .docs-redirect {
          min-height: 60vh;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 12px;
          padding: 48px 24px;
          text-align: center;
        }
        .docs-redirect a {
          color: var(--color-primary, #6366f1);
          text-decoration: underline;
        }
      `}</style>
    </div>
  )
}

export default DocsPage
