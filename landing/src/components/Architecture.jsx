import { motion } from 'framer-motion'

function Architecture() {
  return (
    <section className="architecture-section section" id="architecture">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="section-header"
        >
          <span className="section-label">Technical Architecture</span>
          <h2 className="section-title">Built for Production-Grade AI Agents</h2>
          <p className="section-description">
            Proprietary compression engine with ProMem extraction and spreading activation retrieval.
            Open source, enterprise-ready, and battle-tested.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="architecture-diagram"
        >
          <div className="architecture-layers">
            <div className="layer api-layer">
              <div className="layer-header">
                <h3>API Layer</h3>
                <span>RESTful + GraphQL</span>
              </div>
              <div className="layer-components">
                <div className="component">Memory API</div>
                <div className="component">Search API</div>
                <div className="component">Skills API</div>
                <div className="component">Analytics API</div>
              </div>
            </div>

            <div className="layer compression-layer">
              <div className="layer-header">
                <h3>Compression Engine</h3>
                <span>Proprietary</span>
              </div>
              <div className="layer-components">
                <div className="component">
                  <div className="component-name">ProMem Extractor</div>
                  <div className="component-desc">97% accuracy fact extraction</div>
                </div>
                <div className="component">
                  <div className="component-name">Spreading Activation</div>
                  <div className="component-desc">+23% multi-hop reasoning</div>
                </div>
                <div className="component">
                  <div className="component-name">Async Pipeline</div>
                  <div className="component-desc">&lt;5ms write latency</div>
                </div>
              </div>
            </div>

            <div className="layer storage-layer">
              <div className="layer-header">
                <h3>Storage Layer</h3>
                <span>Tiered Memory System</span>
              </div>
              <div className="layer-components">
                <div className="component hot-tier">
                  <div className="component-name">Hot Tier</div>
                  <div className="component-desc">Redis &lt;20ms</div>
                </div>
                <div className="component working-tier">
                  <div className="component-name">Working</div>
                  <div className="component-desc">&lt;5ms in-memory</div>
                </div>
                <div className="component cold-tier">
                  <div className="component-name">Cold Tier</div>
                  <div className="component-desc">Neo4j+Qdrant &lt;100ms</div>
                </div>
                <div className="component archive-tier">
                  <div className="component-name">Archive</div>
                  <div className="component-desc">Object storage &gt;1s</div>
                </div>
              </div>
            </div>

            <div className="layer infrastructure-layer">
              <div className="layer-header">
                <h3>Infrastructure</h3>
                <span>Open Source Stack</span>
              </div>
              <div className="layer-components">
                <div className="component">Neo4j Graph</div>
                <div className="component">Qdrant Vector</div>
                <div className="component">Redis Cache</div>
                <div className="component">PostgreSQL</div>
                <div className="component">Docker/K8s</div>
              </div>
            </div>
          </div>

          <div className="architecture-stats">
            <div className="stat">
              <div className="stat-number">97%</div>
              <div className="stat-label">Accuracy Retention</div>
            </div>
            <div className="stat">
              <div className="stat-number">85-93%</div>
              <div className="stat-label">Token Reduction</div>
            </div>
            <div className="stat">
              <div className="stat-number">&lt;100ms</div>
              <div className="stat-label">Query Latency</div>
            </div>
            <div className="stat">
              <div className="stat-number">23%</div>
              <div className="stat-label">Multi-hop Gain</div>
            </div>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="architecture-features"
        >
          <div className="feature-grid">
            <div className="feature">
              <div className="feature-icon">🔒</div>
              <h4>Enterprise Security</h4>
              <p>SSO, RBAC, audit logs, and end-to-end encryption</p>
            </div>
            <div className="feature">
              <div className="feature-icon">⚡</div>
              <h4>High Performance</h4>
              <p>Async processing, caching, and optimized queries</p>
            </div>
            <div className="feature">
              <div className="feature-icon">🔧</div>
              <h4>Developer Friendly</h4>
              <p>REST APIs, SDKs, comprehensive docs, and examples</p>
            </div>
            <div className="feature">
              <div className="feature-icon">🌐</div>
              <h4>Cloud Native</h4>
              <p>Containerized, scalable, and cloud-agnostic</p>
            </div>
          </div>
        </motion.div>
      </div>

      <style>{`
        .architecture-section {
          background: var(--bg-secondary);
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

        .architecture-diagram {
          max-width: 1200px;
          margin: 0 auto 60px;
        }

        .architecture-layers {
          display: flex;
          flex-direction: column;
          gap: 16px;
          margin-bottom: 40px;
        }

        .layer {
          border: 1px solid var(--border-light);
          border-radius: 12px;
          overflow: hidden;
          background: var(--card-bg);
        }

        .layer-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 16px 24px;
          background: var(--bg-primary);
          border-bottom: 1px solid var(--border-light);
        }

        .layer-header h3 {
          font-size: 18px;
          font-weight: 600;
          margin: 0;
        }

        .layer-header span {
          font-size: 12px;
          color: var(--text-secondary);
          background: var(--bg-secondary);
          padding: 4px 12px;
          border-radius: 20px;
        }

        .layer-components {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
          padding: 20px;
        }

        .component {
          flex: 1;
          min-width: 180px;
          padding: 16px;
          background: var(--bg-secondary);
          border: 1px solid var(--border-light);
          border-radius: 8px;
          text-align: center;
        }

        .compression-layer .component {
          background: linear-gradient(135deg, #f0f9ff, #e0f2fe);
          border-color: #0284c7;
        }

        .component-name {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 4px;
          color: var(--text-primary);
        }

        .component-desc {
          font-size: 12px;
          color: var(--text-secondary);
        }

        .architecture-stats {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 24px;
          margin-bottom: 60px;
        }

        .stat {
          text-align: center;
          padding: 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
        }

        .stat-number {
          font-size: 32px;
          font-weight: 700;
          color: var(--accent);
          margin-bottom: 8px;
        }

        .stat-label {
          font-size: 14px;
          color: var(--text-secondary);
          font-weight: 500;
        }

        .architecture-features {
          max-width: 1000px;
          margin: 0 auto;
        }

        .feature-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
          gap: 24px;
        }

        .feature {
          text-align: center;
          padding: 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          transition: all 0.3s ease;
        }

        .feature:hover {
          transform: translateY(-4px);
          border-color: var(--accent);
        }

        .feature-icon {
          font-size: 32px;
          margin-bottom: 16px;
        }

        .feature h4 {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 8px;
          color: var(--text-primary);
        }

        .feature p {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
        }

        @media (max-width: 768px) {
          .architecture-section {
            padding: 60px 0;
          }

          .layer-components {
            padding: 16px;
          }

          .component {
                min-width: 140px;
            padding: 12px;
          }

          .architecture-stats {
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 16px;
          }

          .stat {
            padding: 16px;
          }

          .stat-number {
            font-size: 24px;
          }
        }
      `}</style>
    </section>
  )
}

export default Architecture
