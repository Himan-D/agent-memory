import { motion } from 'framer-motion'

const features = [
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        <path d="M10 7v6M7 10h6"/>
      </svg>
    ),
    title: 'Semantic Search',
    description: 'Vector-based similarity search using OpenAI embeddings. Natural language queries return contextually relevant memories.',
    stats: '~100ms query'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <circle cx="12" cy="12" r="3"/>
        <circle cx="19" cy="5" r="2"/>
        <circle cx="5" cy="5" r="2"/>
        <circle cx="19" cy="19" r="2"/>
        <circle cx="5" cy="19" r="2"/>
        <path d="M12 9V5M9 12H5M12 15v4M15 12h4"/>
      </svg>
    ),
    title: 'Knowledge Graph',
    description: 'Neo4j-powered entity relationships. Connect concepts, traverse connections, and discover hidden patterns.',
    stats: 'Cypher queries'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
      </svg>
    ),
    title: 'Memory Compaction',
    description: 'LLM-powered deduplication and summarization. Compress memories to 15% while preserving meaning.',
    stats: '85% compression'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
      </svg>
    ),
    title: 'Skill Extraction',
    description: 'Auto-extract reusable skills from conversations. Synthesize patterns into actionable procedures.',
    stats: 'Auto-synthesis'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
      </svg>
    ),
    title: 'Multi-Agent Sync',
    description: 'Real-time pub/sub memory sharing between agents. Collaborate without context loss.',
    stats: 'Real-time pub/sub'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
    ),
    title: 'Memory Versioning',
    description: 'Full history tracking with version restore. Track changes, rollback mistakes, audit trails.',
    stats: 'Full history'
  }
]

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.1 }
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

function Features() {
  return (
    <section className="features-section section" id="features">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Features</span>
          <h2 className="section-title">Everything your AI agents need to remember</h2>
          <p className="section-description">
            Six memory systems working together. From semantic search to skill extraction, 
            Hystersis gives your agents the memory of a decade.
          </p>
        </motion.div>

        <motion.div
          variants={containerVariants}
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-100px" }}
          className="features-grid"
        >
          {features.map((feature, index) => (
            <motion.div key={index} variants={itemVariants} className="feature-card">
              <div className="feature-icon">{feature.icon}</div>
              <h3 className="feature-title">{feature.title}</h3>
              <p className="feature-description">{feature.description}</p>
              <span className="feature-stats">{feature.stats}</span>
            </motion.div>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.3 }}
          className="features-cta"
        >
          <a href="https://docs.hystersis.ai" className="btn btn-secondary">
            Read the Docs
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M5 12h14M12 5l7 7-7 7"/>
            </svg>
          </a>
        </motion.div>
      </div>

      <style>{`
        .features-section {
          background: var(--bg-primary);
          border-top: 1px solid var(--border-light);
        }
        .section-header {
          text-align: center;
          margin-bottom: 64px;
          max-width: 700px;
          margin-left: auto;
          margin-right: auto;
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
          font-size: clamp(28px, 4vw, 42px);
          font-weight: 700;
          letter-spacing: -1px;
          margin-bottom: 16px;
        }
        .section-description {
          font-size: 16px;
          color: var(--text-secondary);
          line-height: 1.7;
        }
        .feature-card {
          padding: 32px;
          border: 1px solid var(--border-light);
          border-radius: 12px;
          background: var(--card-bg);
          transition: all 0.3s ease;
          display: flex;
          flex-direction: column;
        }
        .feature-card:hover {
          border-color: var(--text-primary);
          transform: translateY(-4px);
        }
        .feature-icon {
          width: 48px;
          height: 48px;
          display: flex;
          align-items: center;
          justify-content: center;
          margin-bottom: 20px;
          background: var(--bg-secondary);
          border-radius: 8px;
        }
        .feature-icon svg {
          width: 24px;
          height: 24px;
          color: var(--text-primary);
        }
        .feature-title {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 12px;
        }
        .feature-description {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.7;
          flex-grow: 1;
          margin-bottom: 16px;
        }
        .feature-stats {
          font-size: 12px;
          font-weight: 600;
          color: var(--text-primary);
          background: var(--bg-secondary);
          padding: 6px 12px;
          border-radius: 6px;
          width: fit-content;
        }
        .features-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 24px;
        }
        .features-cta {
          text-align: center;
          margin-top: 48px;
        }
        .features-cta .btn {
          display: inline-flex;
          align-items: center;
          gap: 8px;
        }
        @media (max-width: 640px) {
          .features-grid { grid-template-columns: 1fr; }
        }
      `}</style>
    </section>
  )
}

export default Features