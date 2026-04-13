import { useState } from 'react'
import { motion } from 'framer-motion'

const codeExamples = [
  {
    title: 'Memory & Search',
    code: `from agentmemory import AgentMemory

client = AgentMemory("https://api.yourserver.com", api_key="your-key")

# Create agent and session
agent = client.create_agent(name="assistant", config={})
session = client.create_session(agent_id=agent["id"])

# Add messages
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great!")

# Semantic search with 85% compression
results = client.semantic_search("deep learning transformers")`
  },
  {
    title: 'Skills & Agents',
    code: `# Extract skills from agent interactions
skill = client.extract_skills(
  agent_id=agent["id"],
  interaction="User asked about ML, bot explained neural networks"
)

# Suggest skills for new tasks
suggestions = client.suggest_skills(task="explain transformers")
# Returns: [{"skill": "transformer_explanation", "confidence": 0.92}]

# Use extracted skill
client.use_skill(skill["id"], context={"topic": "attention mechanism"})`
  },
  {
    title: 'Multi-Agent Groups',
    code: `# Create agent group for collaboration
group = client.create_agent_group(
  name="research-team",
  policy={"shared_memory": True, "sync_interval_ms": 1000}
)

# Add agents to group
client.add_agent_to_group(group["id"], agent["id"], role="researcher")

# Share memory across agents
client.share_memory_to_group(
  group_id=group["id"],
  memory_id=memory["id"],
  shared_by=agent["id"]
)

# Real-time sync with Redis pub/sub
events = client.subscribe_to_group(group["id"])`
  }
]

function CodeDemo() {
  const [activeTab, setActiveTab] = useState(0)

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
          <h2 className="section-title">Simple Python SDK</h2>
          <p className="section-description">
            Just a few lines of code to add memory to your AI agents.
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
                <code>{codeExamples[activeTab].code}</code>
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
          padding: 16px;
          overflow-x: auto;
        }

        .code-body pre {
          font-size: 13px;
          white-space: pre;
          overflow-x: auto;
        }

        @media (max-width: 640px) {
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

        .code-body pre {
          margin: 0;
          font-family: 'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace;
          font-size: 14px;
          line-height: 1.6;
          color: #c9d1d9;
        }

        .code-body code {
          color: #c9d1d9;
        }

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
      `}</style>
    </section>
  )
}

export default CodeDemo
