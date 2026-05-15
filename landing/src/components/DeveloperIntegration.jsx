import { motion } from 'framer-motion'

function DeveloperIntegration() {
  const examples = [
    {
      language: 'Python',
      icon: '🐍',
      code: `from hystersis import HystersisClient

# Initialize client
client = HystersisClient(api_key="your-api-key")

# Store and retrieve memories
client.add_memory("User prefers Python over JavaScript")
memories = client.search("programming preferences")`,
      description: '3 lines to get started with Python SDK'
    },
    {
      language: 'Node.js',
      icon: '🟢',
      code: `import { Hystersis } from '@hystersis/sdk'

// Initialize client
const client = new Hystersis({ apiKey: 'your-api-key' })

// Store and retrieve memories
await client.add('User prefers Python over JavaScript')
const memories = await client.search('programming preferences')`,
      description: '3 lines to get started with Node.js SDK'
    },
    {
      language: 'LangChain',
      icon: '🔗',
      code: `from langchain.memory import HystersisMemory

# Initialize memory
memory = HystersisMemory(api_key="your-api-key")

# Use in your agent
agent = create_agent(memory=memory)`,
      description: 'Seamless LangChain integration'
    }
  ]

  const features = [
    {
      title: 'Zero Configuration',
      description: 'Auto-detects your environment and optimizes settings',
      icon: '⚙️'
    },
    {
      title: 'TypeScript Support',
      description: 'Full type definitions and IntelliSense support',
      icon: '📝'
    },
    {
      title: 'React Components',
      description: 'Pre-built React components for quick integration',
      icon: '⚛️'
    },
    {
      title: 'Streaming Support',
      description: 'Real-time memory streaming and sync',
      icon: '🔄'
    }
  ]

  return (
    <section className="developer-section section" id="developer">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Developer Experience</span>
          <h2 className="section-title">Integrate in 3 Lines of Code</h2>
          <p className="section-description">
            Powerful SDKs with comprehensive documentation, examples, and 24/7 support.
            Start building intelligent agents in minutes.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="code-examples"
        >
          {examples.map((example, index) => (
            <motion.div
              key={index}
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className="code-example"
            >
              <div className="example-header">
                <span className="language-icon">{example.icon}</span>
                <span className="language-name">{example.language}</span>
              </div>
              <div className="code-container">
                <pre>
                  <code>{example.code}</code>
                </pre>
              </div>
              <div className="example-description">{example.description}</div>
            </motion.div>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="sdk-features"
        >
          <h3 className="features-title">SDK Features</h3>
          <div className="features-grid">
            {features.map((feature, index) => (
              <motion.div
                key={index}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.5, delay: index * 0.1 }}
                className="feature-card"
              >
                <div className="feature-icon">{feature.icon}</div>
                <h4 className="feature-title">{feature.title}</h4>
                <p className="feature-description">{feature.description}</p>
              </motion.div>
            ))}
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.6 }}
          className="installation-steps"
        >
          <h3 className="steps-title">Quick Install</h3>
          <div className="steps-container">
            <div className="step">
              <div className="step-number">1</div>
              <div className="step-content">
                <h4>Choose Your Package Manager</h4>
                <div className="package-options">
                  <div className="package-option">
                    <code>npm install @hystersis/sdk</code>
                  </div>
                  <div className="package-option">
                    <code>pip install hystersis</code>
                  </div>
                  <div className="package-option">
                    <code>go get github.com/hystersis/sdk-go</code>
                  </div>
                </div>
              </div>
            </div>
            <div className="step">
              <div className="step-number">2</div>
              <div className="step-content">
                <h4>Get Your API Key</h4>
                <p>Sign up for free at dashboard.hystersis.ai to get your API key</p>
                <div className="api-key-demo">
                  <code>sk-1234567890abcdef</code>
                </div>
              </div>
            </div>
            <div className="step">
              <div className="step-number">3</div>
              <div className="step-content">
                <h4>Start Building</h4>
                <p>Check out our comprehensive documentation and examples</p>
                <div className="quick-links">
                  <a href="https://docs.hystersis.ai" className="btn btn-secondary">
                    Read Docs
                  </a>
                  <a href="https://github.com/Himan-D/agent-memory" className="btn btn-secondary">
                    View Examples
                  </a>
                </div>
              </div>
            </div>
          </div>
        </motion.div>
      </div>

      <style>{`
        .developer-section {
          background: var(--bg-primary);
          padding: 80px 0;
          border-top: 1px solid var(--border-light);
        }

        .section-header {
          text-align: center;
          margin-bottom: 60px;
        }

        .section-label {
          display: inline-block;
          font-size: 12px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 2px;
          color: var(--text-secondary);
          margin-bottom: 16px;
        }

        .section-title {
          font-size: clamp(32px, 5vw, 48px);
          font-weight: 700;
          letter-spacing: -1px;
          margin-bottom: 16px;
        }

        .section-description {
          font-size: 18px;
          color: var(--text-secondary);
          line-height: 1.6;
          max-width: 600px;
          margin: 0 auto;
        }

        .code-examples {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
          gap: 24px;
          margin-bottom: 60px;
        }

        .code-example {
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          overflow: hidden;
        }

        .example-header {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 16px 20px;
          background: var(--bg-secondary);
          border-bottom: 1px solid var(--border-light);
        }

        .language-icon {
          font-size: 20px;
        }

        .language-name {
          font-size: 14px;
          font-weight: 600;
          color: var(--text-primary);
        }

        .code-container {
          position: relative;
        }

        .code-container pre {
          margin: 0;
          padding: 20px;
          background: #1e1e1e;
          color: #d4d4d4;
          font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
          font-size: 13px;
          line-height: 1.5;
          overflow-x: auto;
        }

        .code-container code {
          display: block;
        }

        .example-description {
          padding: 16px 20px;
          font-size: 14px;
          color: var(--text-secondary);
          background: var(--bg-secondary);
          text-align: center;
        }

        .sdk-features {
          max-width: 800px;
          margin: 0 auto 60px;
        }

        .features-title {
          text-align: center;
          font-size: 24px;
          font-weight: 600;
          margin-bottom: 32px;
        }

        .features-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 24px;
        }

        .feature-card {
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          padding: 24px;
          text-align: center;
          transition: all 0.3s ease;
        }

        .feature-card:hover {
          transform: translateY(-4px);
          border-color: var(--accent);
        }

        .feature-icon {
          font-size: 32px;
          margin-bottom: 16px;
        }

        .feature-title {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 8px;
          color: var(--text-primary);
        }

        .feature-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
        }

        .installation-steps {
          max-width: 1000px;
          margin: 0 auto;
        }

        .steps-title {
          text-align: center;
          font-size: 24px;
          font-weight: 600;
          margin-bottom: 40px;
        }

        .steps-container {
          display: flex;
          flex-direction: column;
          gap: 24px;
        }

        .step {
          display: flex;
          gap: 20px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          overflow: hidden;
        }

        .step-number {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 60px;
          font-size: 24px;
          font-weight: 700;
          background: var(--accent);
          color: white;
          flex-shrink: 0;
        }

        .step-content {
          padding: 24px;
          flex: 1;
        }

        .step-content h4 {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 12px;
          color: var(--text-primary);
        }

        .step-content p {
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 16px;
          line-height: 1.6;
        }

        .package-options {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .package-option {
          background: var(--bg-secondary);
          border: 1px solid var(--border-light);
          border-radius: 6px;
          padding: 8px 12px;
        }

        .package-option code {
          font-size: 13px;
          color: var(--text-primary);
        }

        .api-key-demo {
          background: var(--bg-secondary);
          border: 1px solid var(--border-light);
          border-radius: 6px;
          padding: 12px;
          margin-top: 12px;
        }

        .api-key-demo code {
          font-size: 14px;
          color: var(--text-primary);
          font-family: monospace;
        }

        .quick-links {
          display: flex;
          gap: 12px;
          margin-top: 16px;
        }

        .quick-links .btn {
          flex: 1;
          justify-content: center;
        }

        @media (max-width: 768px) {
          .developer-section {
            padding: 60px 0;
          }

          .code-examples {
            grid-template-columns: 1fr;
            gap: 16px;
          }

          .step {
            flex-direction: column;
          }

          .step-number {
            width: 40px;
            height: 40px;
            font-size: 18px;
          }

          .quick-links {
            flex-direction: column;
          }
        }
      `}</style>
    </section>
  )
}

export default DeveloperIntegration
