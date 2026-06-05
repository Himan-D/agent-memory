import { useState } from 'react'
import { motion } from 'framer-motion'

const agentFeatures = [
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/>
      </svg>
    ),
    title: 'API-First',
    description: '95+ REST endpoints with full CRUD for memories, entities, skills, sessions, and agents.',
    stat: '95+ endpoints'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
      </svg>
    ),
    title: 'MCP Server',
    description: '6 Model Context Protocol tools for Claude Code, Cursor, OpenCode, and Windsurf.',
    stat: '6 MCP tools'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/>
      </svg>
    ),
    title: 'Python & Node SDKs',
    description: 'Typed SDKs with memory CRUD, search, entity management, and compression.',
    stat: 'pip install hystersis'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M13 10V3L4 14h7v7l9-11h-7z"/>
      </svg>
    ),
    title: 'Proprietary Compression',
    description: 'ProMem extraction reaches 97%+ accuracy at 85-93% token reduction.',
    stat: '97% accuracy'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
      </svg>
    ),
    title: 'Enterprise SSO',
    description: 'OIDC, SAML, and LDAP authentication out of the box. RBAC with admin, editor, viewer roles.',
    stat: '3 SSO providers'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
      </svg>
    ),
    title: 'Knowledge Graph',
    description: 'Neo4j-powered entity relationships with multi-hop traversal and Cypher queries.',
    stat: 'Cypher + traverse'
  }
]

const codeExamples = [
  {
    language: 'Python',
    code: `from hystersis import Hystersis

client = Hystersis(api_key="your-key")

# Add memory
client.memories.add(
    content="User prefers dark mode",
    agent_id="my-agent"
)

# Spreading activation search
results = client.search_enhanced(
    "user preferences",
    mode="spreading"
)

# Extract skills from content
skills = client.skills.extract(
    content="When the server returns 429..."
)`
  },
  {
    language: 'JavaScript',
    code: `import { Hystersis } from 'hystersis';

const client = new Hystersis({
  apiKey: 'your-key'
});

// Add memory
await client.memories.add({
  content: 'User prefers dark mode',
  agentId: 'my-agent'
});

// Hybrid search
const results = await client.searchHybrid({
  query: 'user preferences',
  semanticWeight: 0.7
});

// Create entity
const entity = await client.entities.create({
  name: 'UserService',
  type: 'Class'
});`
  },
  {
    language: 'cURL',
    code: `# Search with spreading activation
curl "https://api.hystersis.ai/search/enhanced?\\
  mode=spreading&query=user+pref" \\
  -H "X-API-Key: your-key"

# Compress memory
curl -X POST https://api.hystersis.ai/playground/compress \\
  -H "X-API-Key: your-key" \\
  -H "Content-Type: application/json" \\
  -d '{"text": "...", "modes": ["extraction", "radix"]}'

# Get compression metrics
curl https://api.hystersis.ai/metrics/compression \\
  -H "X-API-Key: your-key"`
  }
]

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.08 }
  }
}

const itemVariants = {
  hidden: { opacity: 0, y: 30 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.5, ease: [0.25, 0.46, 0.45, 0.94] }
  }
}

function ForAgentsPage() {
  const [activeTab, setActiveTab] = useState(0)
  const [copied, setCopied] = useState(false)

  const [installCmd, setInstallCmd] = useState('curl')

  const handleCopyInstall = () => {
    const cmd = installCmd === 'curl'
      ? 'curl -fsSL https://hystersis.ai/install.sh | bash'
      : 'pip install hystersis'
    navigator.clipboard.writeText(cmd)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="for-agents-page">
      {/* Hero Section */}
      <section className="fa-hero">
        <div className="container">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="fa-hero-content"
          >
            <span className="fa-badge">For AI Agents</span>
            <h1 className="fa-hero-title">
              Memory Infrastructure for <span className="gradient-text">AI Agents</span>
            </h1>
            <p className="fa-hero-subtitle">
              Persistent memory, knowledge graphs, and multi-hop reasoning — in 5 minutes.
            </p>
            <div className="fa-hero-actions">
              <div className="fa-install-group">
                <div className="fa-install-tabs">
                  <button
                    className={`fa-install-tab ${installCmd === 'curl' ? 'active' : ''}`}
                    onClick={() => setInstallCmd('curl')}
                  >
                    curl
                  </button>
                  <button
                    className={`fa-install-tab ${installCmd === 'pip' ? 'active' : ''}`}
                    onClick={() => setInstallCmd('pip')}
                  >
                    pip
                  </button>
                </div>
                <button className="fa-install-btn" onClick={handleCopyInstall}>
                  {copied ? (
                    <>
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <polyline points="20 6 9 17 4 12"/>
                      </svg>
                      Copied!
                    </>
                  ) : (
                    <>
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                        <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/>
                      </svg>
                      {installCmd === 'curl'
                        ? 'curl -fsSL https://hystersis.ai/install.sh | bash'
                        : 'pip install hystersis'
                      }
                    </>
                  )}
                </button>
              </div>
              <a href="/demo" className="fa-btn fa-btn-secondary">
                Try Live Playground
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M5 12h14M12 5l7 7-7 7"/>
                </svg>
              </a>
            </div>
          </motion.div>
        </div>
      </section>

      {/* Core Capabilities */}
      <section className="fa-capabilities section">
        <div className="container">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6 }}
            className="section-header"
          >
            <span className="section-badge">Core Capabilities</span>
            <h2 className="section-title">Everything your agent needs</h2>
            <p className="section-subtitle">
              95+ endpoints, typed SDKs, MCP server, and proprietary compression.
              Built for the tools agents actually use.
            </p>
          </motion.div>

          <motion.div
            className="fa-grid"
            variants={containerVariants}
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
          >
            {agentFeatures.map((feature, index) => (
              <motion.div key={index} className="fa-card" variants={itemVariants}>
                <div className="fa-card-icon">{feature.icon}</div>
                <h3 className="fa-card-title">{feature.title}</h3>
                <p className="fa-card-description">{feature.description}</p>
                <span className="fa-card-stat">{feature.stat}</span>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* Code Examples */}
      <section className="fa-code section">
        <div className="container">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6 }}
            className="section-header"
          >
            <span className="section-badge">Quick Start</span>
            <h2 className="section-title">Integrate in minutes</h2>
            <p className="section-subtitle">
              Python, JavaScript, or raw API — choose your language.
            </p>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 30 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="fa-code-examples"
          >
            <div className="fa-code-tabs">
              {codeExamples.map((example, index) => (
                <button
                  key={index}
                  className={`fa-code-tab ${activeTab === index ? 'active' : ''}`}
                  onClick={() => setActiveTab(index)}
                >
                  {example.language}
                </button>
              ))}
            </div>
            <div className="fa-code-window">
              <div className="fa-code-header">
                <div className="fa-window-dots">
                  <span className="fa-dot fa-dot-red" />
                  <span className="fa-dot fa-dot-yellow" />
                  <span className="fa-dot fa-dot-green" />
                </div>
              </div>
              <div className="fa-code-body">
                <pre><code>{codeExamples[activeTab].code}</code></pre>
              </div>
            </div>
          </motion.div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="fa-cta section">
        <div className="container">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6 }}
            className="fa-cta-content"
          >
            <h2 className="fa-cta-title">Ready to Give Your Agent Memory?</h2>
            <p className="fa-cta-subtitle">
              Open source. Self-hostable. Get started free.
            </p>
            <div className="fa-cta-actions">
              <a href="https://github.com/Himan-D/agent-memory" className="fa-btn fa-btn-primary" target="_blank" rel="noopener noreferrer">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
                </svg>
                View on GitHub
              </a>
              <a href="/demo" className="fa-btn fa-btn-secondary">
                Try Live Playground
              </a>
              <a href="https://docs.hystersis.ai" className="fa-btn fa-btn-outline" target="_blank" rel="noopener noreferrer">
                Read the Docs
              </a>
            </div>
          </motion.div>
        </div>
      </section>

      <style>{`
        .for-agents-page {
          background: var(--bg-primary);
          min-height: 100vh;
        }

        /* Hero Section */
        .fa-hero {
          padding: 8rem 0 6rem;
          text-align: center;
          background: linear-gradient(180deg, var(--bg-secondary) 0%, var(--bg-primary) 100%);
          border-bottom: 1px solid var(--border-light);
        }

        .fa-badge {
          display: inline-block;
          padding: 0.35rem 1rem;
          border-radius: 100px;
          font-size: 0.85rem;
          font-weight: 600;
          background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary));
          color: white;
          margin-bottom: 1.5rem;
        }

        .fa-hero-title {
          font-size: 3.5rem;
          font-weight: 700;
          line-height: 1.1;
          margin-bottom: 1.5rem;
          color: var(--text-primary);
        }

        .fa-hero-subtitle {
          font-size: 1.25rem;
          color: var(--text-secondary);
          max-width: 600px;
          margin: 0 auto 2rem;
          line-height: 1.6;
        }

        .fa-hero-actions {
          display: flex;
          gap: 1rem;
          justify-content: center;
          flex-wrap: wrap;
        }

        .fa-install-group {
          display: flex;
          flex-direction: column;
          gap: 4px;
        }

        .fa-install-tabs {
          display: flex;
          gap: 2px;
          padding: 3px;
          background: rgba(255,255,255,0.06);
          border-radius: 8px;
          width: fit-content;
        }

        .fa-install-tab {
          padding: 3px 12px;
          font-size: 12px;
          font-weight: 500;
          border: none;
          border-radius: 6px;
          background: transparent;
          color: rgba(255,255,255,0.5);
          cursor: pointer;
          transition: all 0.2s;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
        }

        .fa-install-tab.active {
          background: rgba(255,255,255,0.1);
          color: #fff;
        }

        .fa-install-btn {
          display: inline-flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.875rem 1.5rem;
          font-size: 1rem;
          font-weight: 600;
          color: var(--text-primary);
          background: var(--card-bg);
          border: 1px solid var(--border-medium);
          border-radius: 8px;
          cursor: pointer;
          transition: all 0.2s;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
        }

        .fa-install-btn:hover {
          border-color: var(--accent-primary);
          box-shadow: 0 0 0 2px var(--accent-primary);
        }

        .fa-btn {
          display: inline-flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.875rem 1.5rem;
          font-size: 1rem;
          font-weight: 600;
          border-radius: 8px;
          text-decoration: none;
          transition: all 0.2s;
          cursor: pointer;
          border: none;
        }

        .fa-btn-primary {
          color: white;
          background: var(--btn-primary-bg);
        }

        .fa-btn-primary:hover {
          opacity: 0.9;
          transform: translateY(-1px);
        }

        .fa-btn-secondary {
          color: var(--text-primary);
          background: var(--card-bg);
          border: 1px solid var(--border-medium);
        }

        .fa-btn-secondary:hover {
          border-color: var(--accent-primary);
        }

        .fa-btn-outline {
          color: var(--accent-primary);
          background: transparent;
          border: 1px solid var(--accent-primary);
        }

        .fa-btn-outline:hover {
          background: var(--accent-primary);
          color: white;
        }

        /* Capabilities Section */
        .fa-capabilities {
          padding: 6rem 0;
        }

        .fa-grid {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 1.5rem;
          margin-top: 3rem;
        }

        .fa-card {
          background: var(--card-bg);
          border: 1px solid var(--border-subtle);
          border-radius: 12px;
          padding: 1.5rem;
          transition: transform 0.2s, box-shadow 0.2s;
        }

        .fa-card:hover {
          transform: translateY(-2px);
          box-shadow: 0 8px 30px rgba(0, 0, 0, 0.1);
          border-color: var(--accent-primary);
        }

        .fa-card-icon {
          width: 40px;
          height: 40px;
          margin-bottom: 1rem;
          color: var(--accent-primary);
        }

        .fa-card-icon svg {
          width: 100%;
          height: 100%;
        }

        .fa-card-title {
          font-size: 1.1rem;
          font-weight: 600;
          margin-bottom: 0.5rem;
          color: var(--text-primary);
        }

        .fa-card-description {
          font-size: 0.9rem;
          color: var(--text-secondary);
          line-height: 1.5;
          margin-bottom: 0.75rem;
        }

        .fa-card-stat {
          font-size: 0.8rem;
          font-weight: 600;
          color: var(--accent-primary);
          background: var(--bg-secondary);
          padding: 0.25rem 0.75rem;
          border-radius: 100px;
        }

        /* Code Section */
        .fa-code {
          padding: 6rem 0;
          background: var(--bg-secondary);
          border-top: 1px solid var(--border-light);
          border-bottom: 1px solid var(--border-light);
        }

        .fa-code-examples {
          max-width: 720px;
          margin: 3rem auto 0;
        }

        .fa-code-tabs {
          display: flex;
          gap: 0;
          border-bottom: 1px solid var(--border-subtle);
          background: #161b22;
          border-radius: 12px 12px 0 0;
          padding: 0 0.5rem;
        }

        .fa-code-tab {
          padding: 0.75rem 1.25rem;
          font-size: 0.85rem;
          font-weight: 500;
          color: #8b949e;
          background: transparent;
          border: none;
          border-bottom: 2px solid transparent;
          cursor: pointer;
          transition: all 0.2s;
        }

        .fa-code-tab.active {
          color: #c9d1d9;
          border-bottom-color: var(--accent-primary, #2563EB);
        }

        .fa-code-tab:hover {
          color: #c9d1d9;
        }

        .fa-code-window {
          background: #0d1117;
          border-radius: 0 0 12px 12px;
          overflow: hidden;
        }

        .fa-code-header {
          display: flex;
          align-items: center;
          padding: 14px 16px;
          background: #161b22;
          border-bottom: 1px solid #30363d;
        }

        .fa-window-dots {
          display: flex;
          gap: 8px;
        }

        .fa-dot {
          width: 12px;
          height: 12px;
          border-radius: 50%;
        }

        .fa-dot-red { background: #ff5f56; }
        .fa-dot-yellow { background: #ffbd2e; }
        .fa-dot-green { background: #27c93f; }

        .fa-code-body {
          padding: 20px;
          overflow-x: auto;
        }

        .fa-code-body pre {
          margin: 0;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
          font-size: 13px;
          line-height: 1.6;
          color: #c9d1d9;
        }

        .fa-code-body code {
          color: inherit;
        }

        /* CTA Section */
        .fa-cta {
          padding: 6rem 0;
          text-align: center;
        }

        .fa-cta-content {
          max-width: 700px;
          margin: 0 auto;
        }

        .fa-cta-title {
          font-size: 2.5rem;
          font-weight: 700;
          margin-bottom: 1rem;
          color: var(--text-primary);
        }

        .fa-cta-subtitle {
          font-size: 1.125rem;
          color: var(--text-secondary);
          margin-bottom: 2rem;
        }

        .fa-cta-actions {
          display: flex;
          gap: 1rem;
          justify-content: center;
          flex-wrap: wrap;
        }

        /* Responsive */
        @media (max-width: 1024px) {
          .fa-grid {
            grid-template-columns: repeat(2, 1fr);
          }
        }

        @media (max-width: 768px) {
          .fa-hero {
            padding: 6rem 0 4rem;
          }

          .fa-hero-title {
            font-size: 2.5rem;
          }

          .fa-hero-subtitle {
            font-size: 1.1rem;
          }

          .fa-grid {
            grid-template-columns: 1fr;
          }

          .fa-cta-title {
            font-size: 2rem;
          }

          .fa-cta-actions {
            flex-direction: column;
            align-items: center;
          }

          .fa-hero-actions {
            flex-direction: column;
            align-items: center;
          }
        }
      `}</style>
    </div>
  )
}

export default ForAgentsPage
