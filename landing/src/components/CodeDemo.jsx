import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'

const codeExample = `from agentmemory import AgentMemory

# Connect to your agent memory server
client = AgentMemory("https://api.yourserver.com", api_key="your-key")

# Create a session for your agent
session = client.create_session(agent_id="assistant-bot")

# Add conversation messages
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great! What type?")
client.add_message(session["id"], "user", "Especially neural networks")

# Later, search semantically
results = client.semantic_search("deep learning transformers")
# Returns: [{"score": 0.92, "content": "I love machine learning!..."}]`

function CodeDemo() {
  const [displayedCode, setDisplayedCode] = useState('')
  const [isTyping, setIsTyping] = useState(true)
  const codeRef = useRef(null)

  useEffect(() => {
    let index = 0
    const totalChars = codeExample.length
    
    const typeCode = () => {
      if (index < totalChars) {
        setDisplayedCode(codeExample.slice(0, index + 1))
        index += Math.random() > 0.85 ? 2 : 1
        setTimeout(typeCode, Math.random() > 0.7 ? 30 : 15)
      } else {
        setIsTyping(false)
      }
    }

    setTimeout(typeCode, 500)

    return () => {
      index = totalChars
    }
  }, [])

  const highlightCode = (code) => {
    let result = code
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
    
    const keywords = ['from', 'import', 'def', 'class', 'in', 'return', 'if', 'else', 'with', 'as']
    keywords.forEach(kw => {
      const regex = new RegExp('\\b(' + kw + ')\\b', 'g')
      result = result.replace(regex, '<span class="kw">$1</span>')
    })
    
    const stringRegex = /(["'])(?:[^\\]|\\.)*?\1/g
    result = result.replace(stringRegex, '<span class="str">$&</span>')
    
    const commentRegex = /#.*/g
    result = result.replace(commentRegex, '<span class="cm">$&</span>')
    
    const numRegex = /\b(\d+(?:\.\d+)?)\b/g
    result = result.replace(numRegex, '<span class="num">$1</span>')
    
    return result
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
              <span className="code-filename">example.py</span>
              <div className="code-header-spacer" />
            </div>
            <div className="code-body">
              <pre>
                <code 
                  ref={codeRef}
                  dangerouslySetInnerHTML={{ 
                    __html: highlightCode(displayedCode) + (isTyping ? '<span class="cursor">|</span>' : '')
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
          <div className="stat">
            <span className="stat-value">~100ms</span>
            <span className="stat-label">vector search</span>
          </div>
          <div className="stat">
            <span className="stat-value">50</span>
            <span className="stat-label">connection pool</span>
          </div>
          <div className="stat">
            <span className="stat-value">100</span>
            <span className="stat-label">req/min rate limit</span>
          </div>
        </motion.div>
      </div>

      <style>{`
        .demo-section {
          background: linear-gradient(180deg, var(--bg-primary) 0%, var(--bg-surface) 100%);
        }

        .section-header {
          text-align: center;
          margin-bottom: 48px;
        }

        .section-title {
          font-family: var(--font-display);
          font-size: clamp(28px, 5vw, 40px);
          font-weight: 700;
          margin-bottom: 16px;
          letter-spacing: -1px;
        }

        .section-description {
          font-size: 16px;
          color: var(--text-secondary);
        }

        .code-container {
          max-width: 720px;
          margin: 0 auto 48px;
        }

        .code-window {
          background: #0a0a0a;
          border: 1px solid rgba(255, 255, 255, 0.1);
          border-radius: 12px;
          overflow: hidden;
          box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
        }

        .code-header {
          display: flex;
          align-items: center;
          padding: 14px 16px;
          background: rgba(255, 255, 255, 0.03);
          border-bottom: 1px solid rgba(255, 255, 255, 0.06);
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

        .code-filename {
          flex: 1;
          text-align: center;
          font-size: 13px;
          color: var(--text-muted);
        }

        .code-header-spacer {
          width: 36px;
        }

        .code-body {
          padding: 20px;
          overflow-x: auto;
        }

        .code-body pre {
          margin: 0;
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 14px;
          line-height: 1.7;
        }

        .code-body code {
          color: #e8e8e8;
        }

        .kw { color: #ff79c6; }
        .str { color: #f1fa8c; }
        .cm { color: #6272a4; font-style: italic; }
        .num { color: #bd93f9; }

        .cursor {
          color: var(--color-primary);
          animation: blink 1s step-end infinite;
        }

        @keyframes blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0; }
        }

        .demo-stats {
          display: flex;
          justify-content: center;
          gap: 48px;
          flex-wrap: wrap;
        }

        .stat {
          text-align: center;
        }

        .stat-value {
          display: block;
          font-family: var(--font-display);
          font-size: 28px;
          font-weight: 700;
          color: var(--color-primary);
          margin-bottom: 4px;
        }

        .stat-label {
          font-size: 13px;
          color: var(--text-muted);
        }

        @media (max-width: 640px) {
          .demo-stats {
            gap: 32px;
          }

          .stat-value {
            font-size: 24px;
          }
        }
      `}</style>
    </section>
  )
}

export default CodeDemo