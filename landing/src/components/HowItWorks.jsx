import { motion } from 'framer-motion'

const steps = [
  {
    num: '01',
    title: 'Store',
    description: 'Agents store messages, entities, and relationships via API or SDK',
    code: `client.memories.create({
  content: "User prefers dark mode",
  agent_id: "my-agent"
})`
  },
  {
    num: '02',
    title: 'Compress',
    description: 'ProMem extraction compresses memory 85-93% while retaining 97%+ accuracy',
    code: `result = playground.test_compression(
  text="...",
  modes=["extraction", "radix"]
)
# Best: 91% reduction, 97% accuracy`
  },
  {
    num: '03',
    title: 'Search',
    description: 'Spreading activation traverses knowledge graph for multi-hop reasoning',
    code: `results = client.search_enhanced(
  "user preferences",
  mode="spreading"  # +23% vs vector
)`
  },
  {
    num: '04',
    title: 'Retrieve',
    description: 'Context-ranked results with entity extraction and skill synthesis',
    code: `context = client.sessions.get_context(
  session_id="...",
  include_entities=true,
  include_skills=true
)`
  }
]

function HowItWorks() {
  return (
    <section className="how-section section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">How It Works</span>
          <h2 className="section-title">Four steps to persistent memory</h2>
          <p className="section-description">
            Store, compress, search, and retrieve — with proprietary algorithms that outperform pure vector search.
          </p>
        </motion.div>

        <div className="steps-container">
          {steps.map((step, index) => (
            <motion.div
              key={step.num}
              className="step-card"
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6, delay: index * 0.15 }}
            >
              <div className="step-number">{step.num}</div>
              <h3 className="step-title">{step.title}</h3>
              <p className="step-description">{step.description}</p>
              <div className="step-code">
                <pre><code>{step.code}</code></pre>
              </div>
            </motion.div>
          ))}
        </div>
      </div>

      <style>{`
        .how-section {
          background: var(--bg-secondary);
          border-top: 1px solid var(--border-light);
        }

        .section-header {
          text-align: center;
          margin-bottom: 64px;
          max-width: 700px;
          margin-left: auto;
          margin-right: auto;
        }

        .section-description {
          font-size: 16px;
          color: var(--text-secondary);
          line-height: 1.7;
        }

        .steps-container {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: 24px;
          max-width: 960px;
          margin: 0 auto;
        }

        .step-card {
          padding: 32px 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          text-align: left;
          position: relative;
          transition: all 0.3s ease;
        }

        .step-card:hover {
          border-color: var(--text-primary);
          transform: translateY(-2px);
        }

        .step-number {
          font-size: 13px;
          font-weight: 700;
          color: var(--accent-primary, #2563EB);
          margin-bottom: 12px;
          letter-spacing: 1px;
        }

        .step-title {
          font-size: 22px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .step-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
          margin-bottom: 16px;
        }

        .step-code {
          background: var(--bg-primary, #0d1117);
          border-radius: 8px;
          padding: 16px;
          overflow-x: auto;
        }

        .step-code pre {
          margin: 0;
          font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
          font-size: 12.5px;
          line-height: 1.5;
          color: var(--text-primary);
        }

        .step-code code {
          color: inherit;
        }

        @media (max-width: 768px) {
          .steps-container {
            grid-template-columns: 1fr;
          }

          .step-card {
            max-width: 100%;
          }
        }
      `}</style>
    </section>
  )
}

export default HowItWorks