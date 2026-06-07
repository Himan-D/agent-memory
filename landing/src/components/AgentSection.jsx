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
    description: '25 Model Context Protocol tools for Claude Code, Cursor, OpenCode, and Windsurf.',
    stat: '25 MCP tools'
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
curl "https://api.hystersis.com/search/enhanced?\\
mode=spreading&query=user+pref" \\
  -H "X-API-Key: your-key"

# Compress memory
curl -X POST https://api.hystersis.com/playground/compress \\
  -H "X-API-Key: your-key" \\
  -H "Content-Type: application/json" \\
  -d '{"text": "...", "modes": ["extraction", "radix"]}'

# Get compression metrics
curl https://api.hystersis.com/metrics/compression \\
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

function AgentSection() {
  const [activeTab, setActiveTab] = useState(0)

  return (
    <section className="agent-section section" id="for-agents">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-badge">For Developers</span>
          <h2 className="section-title">Built for the tools <span className="gradient-text">agents use</span></h2>
          <p className="section-subtitle">
            95+ endpoints, 25 MCP tools, typed SDKs, and llms.txt.
            Everything your AI agent needs — in any language, any framework.
          </p>
        </motion.div>

        <motion.div
          className="agent-grid"
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true }}
        >
          {agentFeatures.map((feature, index) => (
            <motion.div key={index} className="agent-card" variants={itemVariants}>
              <div className="agent-card-icon">{feature.icon}</div>
              <h3 className="agent-card-title">{feature.title}</h3>
              <p className="agent-card-description">{feature.description}</p>
              <span className="agent-card-stat">{feature.stat}</span>
            </motion.div>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.3 }}
          className="agent-code-examples"
        >
          <div className="code-tabs">
            {codeExamples.map((example, index) => (
              <button
                key={index}
                className={`code-tab ${activeTab === index ? 'active' : ''}`}
                onClick={() => setActiveTab(index)}
              >
                {example.language}
              </button>
            ))}
          </div>
          <div className="code-window">
            <div className="code-header">
              <div className="window-dots">
                <span className="dot red" />
                <span className="dot yellow" />
                <span className="dot green" />
              </div>
            </div>
            <div className="code-body">
              <pre><code>{codeExamples[activeTab].code}</code></pre>
            </div>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="agent-cta"
        >
          <a href="https://github.com/Himan-D/agent-memory" className="btn btn-primary" target="_blank" rel="noopener noreferrer">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
            </svg>
            View on GitHub
          </a>
          <a href="/demo" className="btn btn-secondary">
            Try Live Playground
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M5 12h14M12 5l7 7-7 7"/>
            </svg>
          </a>
          <a href="/agents.md" className="btn btn-outline" target="_blank" rel="noopener noreferrer">
            Agent Reference
          </a>
        </motion.div>
      </div>

      <style>{`
        .agent-section {
          background: var(--bg-primary);
          padding: 6rem 0;
          border-top: 1px solid var(--border-light);
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

        .agent-grid {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 1.5rem;
          margin: 3rem 0;
        }

        .agent-card {
          background: var(--card-bg);
          border: 1px solid var(--border-subtle);
          border-radius: 12px;
          padding: 1.5rem;
          transition: transform 0.2s, box-shadow 0.2s;
        }

        .agent-card:hover {
          transform: translateY(-2px);
          box-shadow: 0 8px 30px rgba(0, 0, 0, 0.1);
          border-color: var(--accent-primary);
        }

        .agent-card-icon {
          width: 40px;
          height: 40px;
          margin-bottom: 1rem;
          color: var(--accent-primary);
        }

        .agent-card-icon svg {
          width: 100%;
          height: 100%;
        }

        .agent-card-title {
          font-size: 1.1rem;
          font-weight: 600;
          margin-bottom: 0.5rem;
          color: var(--text-primary);
        }

        .agent-card-description {
          font-size: 0.9rem;
          color: var(--text-secondary);
          line-height: 1.5;
          margin-bottom: 0.75rem;
        }

        .agent-card-stat {
          font-size: 0.8rem;
          font-weight: 600;
          color: var(--accent-primary);
          background: var(--bg-secondary);
          padding: 0.25rem 0.75rem;
          border-radius: 100px;
        }

        .agent-code-examples {
          margin: 3rem 0;
          max-width: 720px;
          margin-left: auto;
          margin-right: auto;
        }

        .code-tabs {
          display: flex;
          gap: 0;
          border-bottom: 1px solid var(--border-subtle);
          background: #161b22;
          border-radius: 12px 12px 0 0;
          padding: 0 0.5rem;
        }

        .code-tab {
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

        .code-tab.active {
          color: #c9d1d9;
          border-bottom-color: var(--accent-primary, #2563EB);
        }

        .code-tab:hover {
          color: #c9d1d9;
        }

        .code-window {
          background: #0d1117;
          border-radius: 0 0 12px 12px;
          overflow: hidden;
        }

        .code-header {
          display: flex;
          align-items: center;
          padding: 14px 16px;
          background: #161b22;
          border-bottom: 1px solid #30363d;
        }

        .window-dots {
          display: flex;
          gap: 8px;
        }

        .dot {
          width: 12px;
          height: 12px;
          border-radius: 50%;
        }

        .dot.red { background: #ff5f56; }
        .dot.yellow { background: #ffbd2e; }
        .dot.green { background: #27c93f; }

        .code-body {
          padding: 20px;
          overflow-x: auto;
        }

        .code-body pre {
          margin: 0;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
          font-size: 13px;
          line-height: 1.6;
          color: #c9d1d9;
        }

        .code-body code {
          color: inherit;
        }

        .agent-cta {
          display: flex;
          gap: 1rem;
          justify-content: center;
          margin-top: 2rem;
          flex-wrap: wrap;
        }

        .btn-outline {
          background: transparent;
          border: 1px solid var(--accent-primary);
          color: var(--accent-primary);
          padding: 0.75rem 1.5rem;
          border-radius: 8px;
          font-weight: 600;
          text-decoration: none;
          transition: all 0.2s;
        }

        .btn-outline:hover {
          background: var(--accent-primary);
          color: white;
        }

        @media (max-width: 768px) {
          .agent-grid {
            grid-template-columns: 1fr;
          }

          .agent-cta {
            flex-direction: column;
            align-items: center;
          }
        }
      `}</style>
    </section>
  )
}

export default AgentSection