import { motion } from 'framer-motion'

function DemoPage() {
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
            No signup required.
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
              <span className="preview-url">hystersis.ai/demo</span>
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
                  <div className="preview-result">
                    <div className="result-mode">
                      <span className="result-badge">extraction</span>
                      <span className="result-reduction">91% reduction</span>
                    </div>
                    <div className="result-bar" style={{ width: '91%' }} />
                  </div>
                  <div className="preview-result">
                    <div className="result-mode">
                      <span className="result-badge">radix</span>
                      <span className="result-reduction">78% reduction</span>
                    </div>
                    <div className="result-bar" style={{ width: '78%' }} />
                  </div>
                  <div className="preview-result">
                    <div className="result-mode">
                      <span className="result-badge">hybrid</span>
                      <span className="result-reduction">85% reduction</span>
                    </div>
                    <div className="result-bar" style={{ width: '85%' }} />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="demo-cta-panel">
            <h3 className="cta-heading">Interactive Playground</h3>
            <ul className="cta-features">
              <li>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12"/></svg>
                Test extraction, radix, and hybrid compression
              </li>
              <li>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12"/></svg>
                Compare vector vs spreading activation search
              </li>
              <li>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12"/></svg>
                Visualize knowledge graph from your queries
              </li>
              <li>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="20 6 9 17 4 12"/></svg>
                No signup or API key required
              </li>
            </ul>
            <a href="https://dashboard.hystersis.ai/demo" className="btn btn-primary">
              Open Live Playground
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7"/>
              </svg>
            </a>
            <a href="https://github.com/Himan-D/agent-memory" className="btn btn-secondary" target="_blank" rel="noopener noreferrer">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
              </svg>
              Star on GitHub
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