import { motion } from 'framer-motion'

const features = [
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
        <path d="M10 7v6M7 10h6"/>
      </svg>
    ),
    title: 'Semantic Memory',
    description: 'Store and retrieve memories by meaning, not keywords. ML-powered similarity finds "neural networks" without an exact match.',
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
    description: 'Neo4j-powered entity relationships with 30+ edge types. Multi-hop traversal finds connections Mem0 can\'t.',
    stats: 'Cypher queries'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
      </svg>
    ),
    title: 'Spreading Activation',
    description: 'SYNAPSE-inspired graph propagation with temporal decay. +23% multi-hop reasoning accuracy over standard retrieval.',
    stats: '+23% multi-hop'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
      </svg>
    ),
    title: 'ProMem Compression',
    description: 'Two-pass extraction with self-questioning and verification. 80–85% token reduction at 97% accuracy.',
    stats: '85% compression'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
    ),
    title: 'Temporal Reasoning',
    description: 'Phase rotation preserves history instead of deleting it. Volatile facts decay fast; stable facts persist indefinitely.',
    stats: 'Phase decay'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
      </svg>
    ),
    title: 'Memory Worth Scoring',
    description: 'Outcome-linked importance via success/failure counters. Memories that help get stronger; bad ones fade away.',
    stats: 'Adaptive scoring'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/>
      </svg>
    ),
    title: 'Conflict Resolution',
    description: 'Auditable validity framework: superseded memories are kept as historical facts, not silently overwritten.',
    stats: 'Full audit trail'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
      </svg>
    ),
    title: 'Sleep Consolidation',
    description: 'Auto-Dreamer background consolidation produces 12x memory reduction without losing information.',
    stats: '12x reduction'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
      </svg>
    ),
    title: 'Adaptive Retrieval',
    description: 'Routes queries to the optimal strategy: direct lookup, parallel decomposition, or iterative narrowing.',
    stats: 'Auto-routing'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2M7 4h10M7 4l-2 16h14L17 4"/>
        <path d="M10 8v8M14 8v8"/>
      </svg>
    ),
    title: 'Provenance Tracking',
    description: 'DAG traces which memories influenced which. TD(λ) credit flows back through full derivation chains.',
    stats: 'TD(λ) credit'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
      </svg>
    ),
    title: 'Multi-Agent Sync',
    description: 'Redis-backed real-time memory sharing across agent groups with pub/sub. Collaborate without context loss.',
    stats: 'Real-time pub/sub'
  },
  {
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <path d="M3 9h18M9 21V9"/>
      </svg>
    ),
    title: '12 Vector Providers',
    description: 'Qdrant, Pinecone, Weaviate, Chroma, Pgvector, Milvus, Elastic, Vespa, Redis, MongoDB, Azure, OpenSearch.',
    stats: '12 providers'
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
          <h2 className="section-title">The Complete Memory Stack</h2>
          <p className="section-description">
            Twelve memory systems working together. From semantic search to sleep consolidation,
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
          <a href="/docs" className="btn btn-secondary">
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
