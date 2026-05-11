import { motion } from 'framer-motion'

function LLMWiki() {
  const operations = [
    {
      id: 'ingest',
      title: 'Ingest',
      description: 'Add a source and the LLM reads it, extracts key information, and integrates it into the wiki — updating entity pages, revising summaries, and flagging contradictions.',
      icon: '📥',
      method: 'POST',
      endpoint: '/wiki/ingest',
      example: `{
  "content": "GPT-4 was released by OpenAI...",
  "title": "GPT-4 Overview",
  "content_type": "text/markdown"
}`,
    },
    {
      id: 'query',
      title: 'Query',
      description: 'Ask questions against the wiki. The LLM searches relevant pages, reads them, and synthesizes an answer with citations. Answers can be filed back as new pages.',
      icon: '🔍',
      method: 'POST',
      endpoint: '/wiki/query',
      example: `{
  "query": "How does spreading activation improve retrieval?",
  "save_as_page": true
}`,
    },
    {
      id: 'lint',
      title: 'Lint',
      description: 'Health-check the wiki: find contradictions, stale claims, orphan pages with no inbound links, and information gaps that need research.',
      icon: '🔧',
      method: 'POST',
      endpoint: '/wiki/lint',
      example: `{
  "check_types": ["contradictions", "stale_claims", "orphans", "gaps"]
}`,
    },
  ]

  const pageTypes = [
    { type: 'Summary', icon: '📄', desc: 'High-level overview of a source or topic' },
    { type: 'Entity', icon: '👤', desc: 'Person, organization, concept, place' },
    { type: 'Comparison', icon: '⚡', desc: 'Side-by-side analysis of related items' },
    { type: 'Timeline', icon: '📅', desc: 'Chronological progression of events' },
    { type: 'Analysis', icon: '🔬', desc: 'Deep analysis of complex topics' },
    { type: 'Synthesis', icon: '🔗', desc: 'Integration of multiple sources' },
  ]

  return (
    <section className="llm-wiki-section" id="llm-wiki">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5 }}
        >
          <div className="section-badge">NEW FEATURE</div>
          <h2 className="section-title">LLM Wiki</h2>
          <p className="section-subtitle">
            A persistent, compounding knowledge base that your AI agents build and maintain.
            Inspired by <a href="https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f" target="_blank" rel="noopener noreferrer">Karpathy's LLM Wiki pattern</a>.
          </p>
        </motion.div>

        <div className="wiki-architecture">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: 0.1 }}
            className="architecture-layers"
          >
            <div className="layer">
              <div className="layer-label">Raw Sources</div>
              <div className="layer-desc">Your curated documents — immutable source of truth</div>
            </div>
            <div className="layer-arrow">↓ LLM processes ↓</div>
            <div className="layer layer-wiki">
              <div className="layer-label">The Wiki</div>
              <div className="layer-desc">LLM-generated interlinked markdown pages — summaries, entities, comparisons, timelines</div>
            </div>
            <div className="layer-arrow">↓ Governed by ↓</div>
            <div className="layer layer-schema">
              <div className="layer-label">Schema</div>
              <div className="layer-desc">Configuration that tells the LLM how to structure, link, and maintain the wiki</div>
            </div>
          </motion.div>
        </div>

        <div className="wiki-operations">
          <h3 className="operations-title">Three Core Operations</h3>
          <div className="operations-grid">
            {operations.map((op, i) => (
              <motion.div
                key={op.id}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.4, delay: i * 0.1 }}
                className="operation-card"
              >
                <div className="operation-header">
                  <span className="operation-icon">{op.icon}</span>
                  <h4>{op.title}</h4>
                  <span className="operation-method">{op.method}</span>
                  <code className="operation-endpoint">{op.endpoint}</code>
                </div>
                <p className="operation-desc">{op.description}</p>
                <div className="operation-example">
                  <code><pre>{op.example}</pre></code>
                </div>
              </motion.div>
            ))}
          </div>
        </div>

        <div className="wiki-page-types">
          <h3 className="operations-title">Wiki Page Types</h3>
          <div className="page-types-grid">
            {pageTypes.map((pt, i) => (
              <motion.div
                key={pt.type}
                initial={{ opacity: 0, scale: 0.9 }}
                whileInView={{ opacity: 1, scale: 1 }}
                viewport={{ once: true }}
                transition={{ duration: 0.3, delay: i * 0.05 }}
                className="page-type-card"
              >
                <span className="page-type-icon">{pt.icon}</span>
                <h5>{pt.type}</h5>
                <p>{pt.desc}</p>
              </motion.div>
            ))}
          </div>
        </div>

        <div className="wiki-key-insight">
          <motion.div
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5 }}
            className="insight-card"
          >
            <h4>The Key Difference</h4>
            <p>
              Unlike RAG, which retrieves raw chunks at query time, the wiki is a <strong>persistent, compounding artifact</strong>.
              Cross-references are already there. Contradictions are already flagged. The synthesis already reflects everything you've read.
              The wiki gets richer with every source and every question.
            </p>
            <div className="insight-comparison">
              <div className="comparison-item comparison-rag">
                <span className="comparison-label">RAG</span>
                <p>Retrieve → Synthesize → Discard</p>
              </div>
              <div className="comparison-arrow">vs</div>
              <div className="comparison-item comparison-wiki">
                <span className="comparison-label">LLM Wiki</span>
                <p>Ingest → Compile → Compound</p>
              </div>
            </div>
          </motion.div>
        </div>

        <div className="wiki-cta">
          <h3>Get Started</h3>
          <div className="wiki-code-example">
            <code><pre>{`# Ingest a source
curl -X POST http://localhost:8080/wiki/ingest \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: your-key" \\
  -d '{"content": "Your document text...", "title": "My Source"}'

# Query the wiki
curl -X POST http://localhost:8080/wiki/query \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: your-key" \\
  -d '{"query": "What are the key insights?"}'

# Lint for issues
curl -X POST http://localhost:8080/wiki/lint \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: your-key" \\
  -d '{"check_types": ["contradictions", "orphans"]}'`}</pre></code>
          </div>
        </div>
      </div>

      <style>{`
        .llm-wiki-section {
          padding: 6rem 0;
          background: var(--bg-secondary, #0d1117);
          border-top: 1px solid var(--border-color, #21262d);
        }

        .section-badge {
          display: inline-block;
          padding: 0.25rem 0.75rem;
          background: linear-gradient(135deg, #6366f1, #8b5cf6);
          color: white;
          border-radius: 9999px;
          font-size: 0.75rem;
          font-weight: 600;
          letter-spacing: 0.05em;
          text-transform: uppercase;
          margin-bottom: 1rem;
        }

        .section-title {
          font-size: 2.5rem;
          font-weight: 700;
          margin-bottom: 1rem;
          background: linear-gradient(135deg, #fff, #a5b4fc);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        .section-subtitle {
          font-size: 1.125rem;
          color: var(--text-secondary, #8b949e);
          max-width: 640px;
          margin: 0 auto 3rem;
          line-height: 1.6;
        }

        .section-subtitle a {
          color: #818cf8;
          text-decoration: underline;
        }

        .architecture-layers {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
          max-width: 600px;
          margin: 0 auto 3rem;
        }

        .layer {
          padding: 1.25rem 1.5rem;
          border-radius: 12px;
          border: 1px solid var(--border-color, #21262d);
          background: var(--bg-primary, #161b22);
          text-align: center;
        }

        .layer-wiki {
          border-color: #6366f1;
          background: linear-gradient(135deg, rgba(99, 102, 241, 0.1), rgba(139, 92, 246, 0.05));
        }

        .layer-schema {
          border-color: #8b5cf6;
        }

        .layer-label {
          font-weight: 700;
          font-size: 1.1rem;
          margin-bottom: 0.25rem;
        }

        .layer-desc {
          font-size: 0.875rem;
          color: var(--text-secondary, #8b949e);
        }

        .layer-arrow {
          text-align: center;
          color: var(--text-secondary, #8b949e);
          font-size: 0.8rem;
          padding: 0.25rem 0;
        }

        .operations-title {
          font-size: 1.5rem;
          font-weight: 600;
          text-align: center;
          margin-bottom: 2rem;
        }

        .operations-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 1.5rem;
          margin-bottom: 3rem;
        }

        .operation-card {
          background: var(--bg-primary, #161b22);
          border: 1px solid var(--border-color, #21262d);
          border-radius: 16px;
          padding: 1.5rem;
          transition: border-color 0.2s;
        }

        .operation-card:hover {
          border-color: #6366f1;
        }

        .operation-header {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          flex-wrap: wrap;
          margin-bottom: 0.75rem;
        }

        .operation-icon {
          font-size: 1.5rem;
        }

        .operation-header h4 {
          font-size: 1.1rem;
          font-weight: 600;
          margin: 0;
        }

        .operation-method {
          background: #238636;
          color: white;
          padding: 0.125rem 0.5rem;
          border-radius: 4px;
          font-size: 0.7rem;
          font-weight: 600;
          font-family: monospace;
        }

        .operation-endpoint {
          font-size: 0.75rem;
          color: var(--text-secondary, #8b949e);
          font-family: monospace;
        }

        .operation-desc {
          font-size: 0.875rem;
          color: var(--text-secondary, #8b949e);
          line-height: 1.5;
          margin-bottom: 1rem;
        }

        .operation-example {
          background: #0d1117;
          border-radius: 8px;
          overflow: hidden;
        }

        .operation-example code pre {
          margin: 0;
          padding: 0.75rem;
          font-size: 0.75rem;
          color: #e6edf3;
          overflow-x: auto;
          white-space: pre-wrap;
        }

        .page-types-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
          gap: 1rem;
          margin-bottom: 3rem;
        }

        .page-type-card {
          background: var(--bg-primary, #161b22);
          border: 1px solid var(--border-color, #21262d);
          border-radius: 12px;
          padding: 1.25rem;
          text-align: center;
          transition: border-color 0.2s, transform 0.2s;
        }

        .page-type-card:hover {
          border-color: #8b5cf6;
          transform: translateY(-2px);
        }

        .page-type-icon {
          font-size: 2rem;
          display: block;
          margin-bottom: 0.5rem;
        }

        .page-type-card h5 {
          font-size: 0.9rem;
          font-weight: 600;
          margin: 0 0 0.25rem;
        }

        .page-type-card p {
          font-size: 0.75rem;
          color: var(--text-secondary, #8b949e);
          margin: 0;
        }

        .insight-card {
          background: linear-gradient(135deg, rgba(99, 102, 241, 0.08), rgba(139, 92, 246, 0.04));
          border: 1px solid rgba(99, 102, 241, 0.3);
          border-radius: 16px;
          padding: 2rem;
          margin-bottom: 3rem;
          text-align: center;
        }

        .insight-card h4 {
          font-size: 1.25rem;
          font-weight: 700;
          margin-bottom: 0.75rem;
        }

        .insight-card p {
          font-size: 0.95rem;
          color: var(--text-secondary, #8b949e);
          line-height: 1.7;
          max-width: 640px;
          margin: 0 auto 1.5rem;
        }

        .insight-comparison {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 1.5rem;
          flex-wrap: wrap;
        }

        .comparison-item {
          padding: 1rem 1.5rem;
          border-radius: 12px;
          text-align: center;
          min-width: 180px;
        }

        .comparison-rag {
          background: rgba(248, 81, 73, 0.1);
          border: 1px solid rgba(248, 81, 73, 0.3);
        }

        .comparison-wiki {
          background: rgba(35, 134, 54, 0.1);
          border: 1px solid rgba(35, 134, 54, 0.3);
        }

        .comparison-label {
          font-weight: 700;
          font-size: 1rem;
          display: block;
          margin-bottom: 0.25rem;
        }

        .comparison-rag .comparison-label { color: #f85149; }
        .comparison-wiki .comparison-label { color: #3fb950; }

        .comparison-item p {
          font-size: 0.85rem;
          margin: 0;
          color: var(--text-secondary, #8b949e);
          font-family: monospace;
        }

        .comparison-arrow {
          font-size: 1rem;
          color: var(--text-secondary, #8b949e);
          font-weight: 600;
        }

        .wiki-cta h3 {
          text-align: center;
          font-size: 1.5rem;
          margin-bottom: 1.5rem;
        }

        .wiki-code-example {
          background: #0d1117;
          border: 1px solid var(--border-color, #21262d);
          border-radius: 12px;
          overflow-x: auto;
        }

        .wiki-code-example code pre {
          margin: 0;
          padding: 1.5rem;
          font-size: 0.8rem;
          color: #e6edf3;
          line-height: 1.6;
        }
      `}</style>
    </section>
  )
}

export default LLMWiki