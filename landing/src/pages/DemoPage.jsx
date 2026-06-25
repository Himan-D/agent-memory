import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { demoApi } from '../utils/api'
import { DASHBOARD_SIGNIN_URL, DASHBOARD_URL } from '../constants'

const STATIC_RESULTS = [
  { mode: 'extraction', reduction: 91, label: '91% reduction' },
  { mode: 'radix', reduction: 78, label: '78% reduction' },
  { mode: 'hybrid', reduction: 85, label: '85% reduction' },
]

function DemoPage() {
  const [results, setResults] = useState(STATIC_RESULTS)
  const [isDemoData, setIsDemoData] = useState(false)
  const user = "Faizan"

  useEffect(() => {
    demoApi.getDashboard()
      .then(data => {
        if (data && Array.isArray(data.compression_results) && data.compression_results.length > 0) {
          setResults(data.compression_results)
          setIsDemoData(false)
        } else {
          setIsDemoData(true)
        }
      })
      .catch(() => {
        setIsDemoData(true)
      })
  }, [])

  return (
    <section className="demo-page section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
          className="demo-header"
        >
          <span className="section-badge">Live Playground</span>
          <h1 className="demo-title">Try Hystersis in Your Browser</h1>
          <p className="demo-description">
            Test compression, search, and knowledge graph algorithms with real data.
            Sign in to the dashboard for the full experience.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="demo-card"
        >
          <div className="demo-preview">
            <div className="preview-header">
              <div className="preview-dots">
                <span className="dot red" />
                <span className="dot yellow" />
                <span className="dot green" />
              </div>
              <span className="preview-url">hystersis.com/demo</span>
              {isDemoData && (
                <span className="demo-data-label">(demo data)</span>
              )}
            </div>
            <div className="preview-body">
              <div className="preview-tabs">
                <span className="preview-tab active">Compression</span>
                <span className="preview-tab">Search</span>
                <span className="preview-tab">Graph</span>
              </div>
              <div className="preview-content">
                <div className="preview-label">Input Text</div>
                <div className="preview-box">
                  machine learning is a subset of artificial intelligence
                  deep learning is a subset of machine learning
                  neural networks are used for learning
                </div>
                <div className="preview-label">Results</div>
                <div className="preview-results">
                  {results.map((r) => (
                    <div key={r.mode} className="preview-result">
                      <div className="result-mode">
                        <span className="result-badge">{r.mode}</span>
                        <span className="result-reduction">{r.label || `${r.reduction}% reduction`}</span>
                      </div>
                      <div className="result-bar" style={{ width: `${r.reduction}%` }} />
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>

          <div className="demo-cta-panel">
            <h3 className="cta-heading">
              {user ? "Continue Your Playground" : "Interactive Playground"}
            </h3>
            <ul className="cta-features">
              <li>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12" /></svg>
                Test extraction, radix, and hybrid compression
              </li>
              <li>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12" /></svg>
                Compare vector vs spreading activation search
              </li>
              <li>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12" /></svg>
                Visualize knowledge graph from your queries
              </li>
              {!user && (
                <li>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12" /></svg>
                  No signup or API key required
                </li>
              )}
            </ul>

            <p className="demo-signin-prompt">Try the full demo on our dashboard</p>
            <a href={DASHBOARD_SIGNIN_URL} className="btn btn-primary"
              style={{
                marginBottom: "10px",
                marginTop: "7px"
              }}
            >
              Sign In to Dashboard
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
            <a href={`${DASHBOARD_URL}/demo`} className="btn btn-secondary">
              Open Live Playground
            </a>
          </div>
        </motion.div>
      </div>

      <style>{`
        .demo-page {
          background: var(--bg-primary);
          min-height: 80vh;
          display: flex;
          align-items: center;
        }

        .section-badge {
          display: inline-block;
          padding: 0.35rem 1rem;
          border-radius: 100px;
          font-size: 0.85rem;
          font-weight: 600;
          background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary));
          color: white;
          margin-bottom: 1rem;
        }

        .demo-header {
          text-align: center;
          max-width: 600px;
          margin: 0 auto 3rem;
        }

        .demo-title {
          font-size: clamp(32px, 5vw, 48px);
          font-weight: 700;
          letter-spacing: -1px;
          margin-bottom: 1rem;
        }

        .demo-description {
          font-size: 18px;
          color: var(--text-secondary);
          line-height: 1.6;
        }

        .user-status {
          margin-top: 1rem;
          display: flex;
          justify-content: center;
        }

        .status-badge {
          background: linear-gradient(135deg, #10b981, #059669);
          color: white;
          padding: 0.5rem 1rem;
          border-radius: 100px;
          font-size: 0.9rem;
          font-weight: 500;
        }

        .demo-card {
          display: grid;
          grid-template-columns: 1.2fr 1fr;
          gap: 2rem;
          align-items: center;
          max-width: 1000px;
          margin: 0 auto;
        }

        .demo-preview {
          background: #0d1117;
          border-radius: 12px;
          overflow: hidden;
          border: 1px solid #30363d;
        }

        .preview-header {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 12px 16px;
          background: #161b22;
          border-bottom: 1px solid #30363d;
        }

        .preview-dots {
          display: flex;
          gap: 6px;
        }

        .dot {
          width: 10px;
          height: 10px;
          border-radius: 50%;
        }

        .dot.red { background: #ff5f56; }
        .dot.yellow { background: #ffbd2e; }
        .dot.green { background: #27c93f; }

        .preview-url {
          font-size: 12px;
          color: #8b949e;
          margin-left: 8px;
        }

        .demo-data-label {
          font-size: 11px;
          color: #8b949e;
          margin-left: auto;
          font-style: italic;
        }

        .preview-tabs {
          display: flex;
          gap: 0;
          padding: 0;
          border-bottom: 1px solid #30363d;
          background: #161b22;
        }

        .preview-tab {
          padding: 10px 20px;
          font-size: 13px;
          color: #8b949e;
        }

        .preview-tab.active {
          color: #c9d1d9;
          border-bottom: 2px solid #2563EB;
        }

        .preview-content {
          padding: 16px;
        }

        .preview-label {
          font-size: 11px;
          font-weight: 600;
          color: #8b949e;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          margin-bottom: 8px;
          margin-top: 16px;
        }

        .preview-label:first-child {
          margin-top: 0;
        }

        .preview-box {
          background: #161b22;
          border: 1px solid #30363d;
          border-radius: 6px;
          padding: 12px;
          font-family: 'SF Mono', monospace;
          font-size: 12px;
          color: #8b949e;
          line-height: 1.6;
        }

        .preview-results {
          display: flex;
          flex-direction: column;
          gap: 10px;
        }

        .preview-result .result-mode {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 4px;
        }

        .result-badge {
          font-size: 11px;
          font-weight: 600;
          padding: 2px 8px;
          border-radius: 4px;
          background: #21262d;
          color: #c9d1d9;
        }

        .result-reduction {
          font-size: 12px;
          font-weight: 600;
          color: #3fb950;
        }

        .result-bar {
          height: 4px;
          background: #3fb950;
          border-radius: 2px;
          transition: width 1s ease;
        }

        .demo-cta-panel {
          padding: 2rem;
        }

        .cta-heading {
          font-size: 24px;
          font-weight: 700;
          margin-bottom: 1.5rem;
        }

        .cta-features {
          list-style: none;
          padding: 0;
          margin: 0 0 2rem;
        }

        .cta-features li {
          display: flex;
          align-items: center;
          gap: 10px;
          font-size: 15px;
          color: var(--text-secondary);
          margin-bottom: 12px;
        }

        .cta-features li svg {
          color: #3fb950;
          flex-shrink: 0;
        }

        .demo-credentials {
          background: var(--bg-secondary);
          border: 1px solid var(--border-light);
          border-radius: 8px;
          padding: 1rem;
          margin-bottom: 1.5rem;
        }

        .demo-label {
          font-size: 12px;
          font-weight: 600;
          color: var(--text-secondary);
          margin-bottom: 8px;
          text-align: center;
        }

        .demo-info {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 4px;
          font-size: 12px;
        }

        .demo-info code {
          background: var(--bg-primary);
          padding: 4px 8px;
          border-radius: 4px;
          border: 1px solid var(--border-light);
          font-family: 'Monaco', 'Menlo', monospace;
        }

        @media (max-width: 768px) {
          .demo-card {
            grid-template-columns: 1fr;
          }

          .demo-page {
            padding: 2rem 0;
          }

          .demo-title {
            font-size: 28px;
          }
        }
      `}</style>
    </section>
  )
}

export default DemoPage
