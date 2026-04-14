import { useState } from 'react'
import { motion } from 'framer-motion'
import Prism from 'prismjs'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-json'

const codeExamples = [
  {
    title: 'Python',
    language: 'python',
    code: `from hystersis import Hystersis

client = Hystersis(
    "https://api.hystersis.ai",
    api_key="your-key"
)

# Create a session for your agent
session = client.create_session(
    agent_id="assistant-bot"
)

# Add conversation messages
client.add_message(session["id"], "user", 
    "I love machine learning!")
client.add_message(session["id"], "assistant",
    "That's great! What type?")

# Store a semantic memory
memory = client.create_memory(
    content="User interested in ML and AI",
    user_id="user-123",
    category="preferences"
)

# Search semantically
results = client.search("deep learning transformers")
# Returns: [{score: 0.92, content: "User interested..."}]`
  },
  {
    title: 'TypeScript',
    language: 'typescript',
    code: `import { Hystersis } from 'hystersis';

const client = new Hystersis({
  baseUrl: 'https://api.hystersis.ai',
  apiKey: 'your-api-key'
});

// Create a session
const session = await client.sessions.create({
  agentId: 'assistant-bot'
});

// Add messages
await client.messages.add(
  session.id, 'user', 
  'I love machine learning!'
);

// Store a semantic memory
const memory = await client.memories.create({
  content: 'User interested in ML and AI',
  userId: 'user-123',
  category: 'preferences'
});

// Search semantically
const results = await client.memories.search({
  query: 'deep learning transformers',
  limit: 10
});`
  },
{
    title: 'CLI',
    language: 'bash',
    code: `# Install SDK
pip install hystersis

# Or with npm
npm install hystersis

# CLI usage
hystersis init --api-key your-key

# Add memory
hystersis memory add "User loves coffee" \\
  --category preferences

# Search
hystersis search "beverages"

# Start server
hystersis server start`
  },
  {
    title: 'API Keys',
    language: 'bash',
    code: `# Create API key with scope
curl -X POST https://api.hystersis.ai/admin/api-keys \\
  -H "Authorization: Bearer YOUR_ADMIN_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "label": "Production App",
    "scope": "write",
    "expires_in_hours": 720
  }'

# Response:
# {"id":"key_abc123","key":"am_xxxxx","label":"Production App","scope":"write"}

# List your API keys
curl https://api.hystersis.ai/admin/api-keys \\
  -H "Authorization: Bearer YOUR_API_KEY"

# Delete API key (requires admin scope)
curl -X DELETE https://api.hystersis.ai/admin/api-keys/key_abc123 \\
  -H "Authorization: Bearer YOUR_ADMIN_KEY"

# Scope levels:
# - read: GET requests only
# - write: GET + POST/PUT/DELETE
# - admin: Full access + key management`
  }
]

function CodeDemo() {
  const [activeTab, setActiveTab] = useState(0)

  const highlightCode = (code, language) => {
    try {
      return Prism.highlight(code, Prism.languages[language], language)
    } catch (e) {
      return code
    }
  }

  return (
    <section className="demo-section section" id="demo">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">How It Works</span>
          <h2 className="section-title">Simple SDK, powerful memory</h2>
          <p className="section-description">
            Add persistent memory to your AI agents in minutes.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 40 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="code-container"
        >
          <div className="code-window">
            <div className="code-header">
              <div className="window-dots">
                <span className="dot red" />
                <span className="dot yellow" />
                <span className="dot green" />
              </div>
              <div className="code-tabs">
                {codeExamples.map((example, index) => (
                  <button
                    key={index}
                    className={`code-tab ${activeTab === index ? 'active' : ''}`}
                    onClick={() => setActiveTab(index)}
                  >
                    {example.title}
                  </button>
                ))}
              </div>
              <div className="code-header-spacer" />
            </div>
            <div className="code-body">
              <pre>
                <code 
                  className={`language-${codeExamples[activeTab].language}`}
                  dangerouslySetInnerHTML={{
                    __html: highlightCode(
                      codeExamples[activeTab].code,
                      codeExamples[activeTab].language
                    )
                  }}
                />
              </pre>
            </div>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="demo-stats"
        >
          <div className="demo-stat">
            <span className="demo-stat-value">~100ms</span>
            <span className="demo-stat-label">vector search</span>
          </div>
          <div className="demo-stat">
            <span className="demo-stat-value">85%</span>
            <span className="demo-stat-label">compression</span>
          </div>
          <div className="demo-stat">
            <span className="demo-stat-value">Real-time</span>
            <span className="demo-stat-label">pub/sub sync</span>
          </div>
        </motion.div>
      </div>

      <style>{`
        .demo-section {
          background: var(--bg-primary);
        }

        .section-header {
          text-align: center;
          margin-bottom: 48px;
        }

        .code-container {
          max-width: 720px;
          margin: 0 auto 48px;
        }

        .code-window {
          background: #0d1117;
          border-radius: 12px;
          overflow: hidden;
          box-shadow: 0 4px 24px rgba(0,0,0,0.12);
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

        .code-tabs {
          display: flex;
          gap: 4px;
          margin-left: 20px;
        }

        .code-tab {
          padding: 6px 12px;
          font-size: 12px;
          font-weight: 500;
          color: #8b949e;
          background: transparent;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          transition: all 0.2s ease;
        }

        .code-tab:hover {
          color: #c9d1d9;
        }

        .code-tab.active {
          color: #c9d1d9;
          background: #21262d;
        }

        .code-header-spacer {
          flex: 1;
        }

        .code-body {
          padding: 20px;
          overflow-x: auto;
        }

        .code-body pre {
          margin: 0;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
          font-size: 13px;
          line-height: 1.6;
        }

        .code-body code {
          color: #c9d1d9;
        }

        /* Prism.js token colors */
        .token.comment { color: #8b949e; font-style: italic; }
        .token.keyword { color: #ff7b72; }
        .token.function { color: #d2a8ff; }
        .token.string { color: #a5d6ff; }
        .token.number { color: #79c0ff; }
        .token.operator { color: #ff7b72; }
        .token.class-name { color: #ffa657; }
        .token.builtin { color: #ffa657; }
        .token.operator { color: #79c0ff; }
        .token.punctuation { color: #c9d1d9; }
        .token.property { color: #79c0ff; }
        .token.boolean { color: #ff7b72; }

        .demo-stats {
          display: flex;
          justify-content: center;
          gap: 48px;
          flex-wrap: wrap;
        }

        .demo-stat {
          text-align: center;
        }

        .demo-stat-value {
          display: block;
          font-size: 28px;
          font-weight: 700;
          margin-bottom: 4px;
        }

        .demo-stat-label {
          font-size: 13px;
          color: var(--text-secondary);
        }

        @media (max-width: 640px) {
          .code-body {
            padding: 16px;
          }

          .code-body pre {
            font-size: 12px;
            line-height: 1.5;
          }

          .code-tabs {
            display: none;
          }

          .code-window {
            border-radius: 8px;
          }
        }
      `}</style>
    </section>
  )
}

export default CodeDemo
