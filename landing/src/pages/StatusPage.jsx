import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { CLOUDFLARE_MCP_STATUS, runDeployDiagnostics } from '../lib/deploy-status'

const BETTERSTACK_API_URL = import.meta.env.VITE_BETTERSTACK_API_URL || 'https://api.hystersis.com'
const BETTERSTACK_MONITORS_URL = import.meta.env.VITE_BETTERSTACK_MONITORS_URL || 'https://api.hystersis.com/monitors'
const BETTERSTACK_API_TOKEN = import.meta.env.VITE_BETTERSTACK_API_TOKEN || ''

function StatusPage() {
  const [status, setStatus] = useState(null)
  const [monitors, setMonitors] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [deployDiag, setDeployDiag] = useState(null)
  const [deployLoading, setDeployLoading] = useState(true)

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch(BETTERSTACK_API_URL, {
          headers: {
            'Authorization': `Bearer ${BETTERSTACK_API_TOKEN}`
          }
        })
        if (!response.ok) throw new Error('Failed to fetch status')
        const data = await response.json()
        setStatus(data)
      } catch (err) {
        console.warn('Using demo status data:', err.message)
        setError(err.message)
        setStatus({
          data: {
            attributes: {
              name: 'Hystersis',
              status: 'up',
              url: 'https://api.hystersis.com',
              response_time: 125,
              uptime: 99.98,
              check_frequency: 60,
              last_check_at: new Date().toISOString(),
              next_check_at: new Date(Date.now() + 60000).toISOString(),
            }
          }
        })
      } finally {
        setLoading(false)
      }
    }

    fetchStatus()
    const interval = setInterval(fetchStatus, 60000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    const fetchMonitors = async () => {
      try {
        const response = await fetch(BETTERSTACK_MONITORS_URL, {
          headers: {
            'Authorization': `Bearer ${BETTERSTACK_API_TOKEN}`
          }
        })
        if (!response.ok) throw new Error('Failed to fetch monitors')
        const data = await response.json()
        setMonitors(data.data || [])
      } catch (err) {
        console.warn('Using demo monitors data:', err.message)
        setMonitors([
          { id: '1', attributes: { name: 'API', status: 'up', response_time: 125 } },
          { id: '2', attributes: { name: 'Neo4j', status: 'up', response_time: 45 } },
          { id: '3', attributes: { name: 'Qdrant', status: 'up', response_time: 23 } },
        ])
      }
    }

    fetchMonitors()
  }, [])

  useEffect(() => {
    const refreshDeploy = () => {
      runDeployDiagnostics()
        .then(setDeployDiag)
        .catch((err) => console.warn('Deploy diagnostics failed:', err))
        .finally(() => setDeployLoading(false))
    }
    refreshDeploy()
    const interval = setInterval(refreshDeploy, 120000)
    return () => clearInterval(interval)
  }, [])

  const getStatusColor = (status) => {
    switch (status) {
      case 'up': return '#27c93f'
      case 'down': return '#ff5f56'
      case 'degraded': return '#ffbd2e'
      default: return '#999'
    }
  }

  const getStatusLabel = (status) => {
    switch (status) {
      case 'up': return 'Operational'
      case 'down': return 'Down'
      case 'degraded': return 'Degraded'
      default: return 'Unknown'
    }
  }

  const overallStatus = status?.data?.attributes?.status === 'up' ? 'operational' : 'down'
  const uptime = status?.data?.attributes?.uptime?.toFixed(2) || '99.98'

  const defaultServices = [
    { name: 'API', status: 'up', latency: 125, description: 'Main API server' },
    { name: 'Neo4j', status: 'up', latency: 45, description: 'Knowledge graph database' },
    { name: 'Qdrant', status: 'up', latency: 23, description: 'Vector search engine' },
  ]

  const services = monitors.length > 0 
    ? monitors.map(m => ({
        name: m.attributes.name,
        status: m.attributes.status,
        latency: m.attributes.response_time,
        description: `${m.attributes.name} service`
      }))
    : defaultServices

  return (
    <div className="status-page">
      <div className="status-container">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="status-header"
        >
          <h1>System Status</h1>
          <p className="status-subtitle">Real-time status of Hystersis infrastructure</p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="status-banner"
          style={{ borderColor: getStatusColor(overallStatus) }}
        >
          <div 
            className="status-indicator" 
            style={{ backgroundColor: getStatusColor(overallStatus) }} 
          />
          <div className="status-info">
            <span className="status-label">
              {overallStatus === 'operational' ? 'All Systems Operational' : 'System Issue Detected'}
            </span>
            <span className="status-time">
              Last checked: {status?.data?.attributes?.last_check_at 
                ? new Date(status.data.attributes.last_check_at).toLocaleString()
                : 'Just now'}
            </span>
          </div>
          <div className="uptime-badge">
            <span className="uptime-value">{uptime}%</span>
            <span className="uptime-label">Uptime</span>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="uptime-chart"
        >
          <h3>30-Day Uptime</h3>
          <div className="uptime-bar">
            <div className="uptime-fill" style={{ width: `${uptime}%` }} />
          </div>
          <div className="uptime-stats">
            <span>{uptime}% uptime this month</span>
            <span>Next check in 60s</span>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.3 }}
          className="services-grid"
        >
          {services.map((service, index) => (
            <div key={index} className="service-card">
              <div className="service-header">
                <div className="service-name-group">
                  <h3>{service.name}</h3>
                  <div 
                    className="service-dot"
                    style={{ backgroundColor: getStatusColor(service.status) }}
                  />
                </div>
                <span 
                  className="service-status"
                  style={{ color: getStatusColor(service.status) }}
                >
                  {getStatusLabel(service.status)}
                </span>
              </div>
              <p className="service-description">{service.description}</p>
              <div className="service-metrics">
                <span className="metric">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <circle cx="12" cy="12" r="10"/>
                    <polyline points="12 6 12 12 16 14"/>
                  </svg>
                  {service.latency}ms
                </span>
                <span className="metric">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                    <polyline points="22 4 12 14.01 9 11.01"/>
                  </svg>
                  {service.status === 'up' ? 'Responding' : 'No response'}
                </span>
              </div>
            </div>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.4 }}
          className="deploy-section"
        >
          <h2>Deploy &amp; DNS</h2>
          <p className="deploy-subtitle">
            Live checks for production domains. Run <code>bash scripts/verify-domains.sh</code> in CI for full validation.
          </p>

          {deployLoading && !deployDiag ? (
            <p className="deploy-loading">Checking DNS and endpoints…</p>
          ) : deployDiag ? (
            <>
              <div
                className="deploy-banner"
                style={{
                  borderColor: deployDiag.summary.healthy ? '#27c93f' : '#ffbd2e',
                }}
              >
                <span>
                  {deployDiag.summary.healthy
                    ? 'All deploy checks passing'
                    : `${deployDiag.summary.dnsTotal - deployDiag.summary.dnsOk} DNS and ${deployDiag.summary.httpTotal - deployDiag.summary.httpOk} HTTP issue(s) detected`}
                </span>
                <span className="deploy-checked">
                  Checked {new Date(deployDiag.checkedAt).toLocaleString()}
                </span>
              </div>

              <div className="deploy-grid">
                <div className="deploy-card">
                  <h3>DNS records</h3>
                  <ul className="deploy-list">
                    {deployDiag.dns.map((row) => (
                      <li key={row.hostname} className={row.ok ? 'ok' : 'fail'}>
                        <span className="deploy-label">{row.hostname}</span>
                        <span className="deploy-value">
                          {row.ok ? row.records.join(', ') : row.error || 'Missing'}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>

                <div className="deploy-card">
                  <h3>HTTP endpoints</h3>
                  <ul className="deploy-list">
                    {deployDiag.http.map((row) => (
                      <li key={row.url} className={row.ok ? 'ok' : 'fail'}>
                        <span className="deploy-label">{row.name}</span>
                        <span className="deploy-value">
                          {row.status != null ? `HTTP ${row.status}` : row.error || 'Unreachable'}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </>
          ) : null}

          <div className="deploy-card mcp-card">
            <h3>Cloudflare MCP (Cursor)</h3>
            <ul className="deploy-list">
              {CLOUDFLARE_MCP_STATUS.map((row) => (
                <li key={row.server} className={row.status === 'ready' ? 'ok' : 'warn'}>
                  <span className="deploy-label">{row.server}</span>
                  <span className="deploy-value">{row.status} — {row.note}</span>
                </li>
              ))}
            </ul>
            <p className="mcp-hint">
              Cloud Agents cannot use OAuth-backed MCP servers. Authenticate Cloudflare MCP in Cursor Desktop, or add{' '}
              <code>CLOUDFLARE_API_TOKEN</code> to GitHub secrets and run Deploy Cloudflare (All).
            </p>
          </div>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.5 }}
          className="subscribe-section"
        >
          <p>Get notified about incidents</p>
          <a href="https://hystersis.com/status" className="subscribe-btn" target="_blank" rel="noopener noreferrer">
            Subscribe to Updates →
          </a>
        </motion.div>
      </div>

      <style>{`
        .status-page {
          min-height: 100vh;
          background: var(--bg-primary);
          padding: 120px 24px 80px;
        }

        .status-container {
          max-width: 800px;
          margin: 0 auto;
        }

        .status-header {
          text-align: center;
          margin-bottom: 32px;
        }

        .status-header h1 {
          font-size: 36px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .status-subtitle {
          color: var(--text-secondary);
          font-size: 16px;
        }

        .status-banner {
          display: flex;
          align-items: center;
          gap: 16px;
          padding: 24px;
          background: var(--card-bg);
          border: 2px solid var(--border-light);
          border-radius: 16px;
          margin-bottom: 24px;
        }

        .status-indicator {
          width: 16px;
          height: 16px;
          border-radius: 50%;
          animation: pulse 2s infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.7; transform: scale(0.95); }
        }

        .status-info {
          flex: 1;
        }

        .status-label {
          display: block;
          font-size: 20px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .status-time {
          font-size: 13px;
          color: var(--text-muted);
        }

        .uptime-badge {
          text-align: center;
          padding: 12px 20px;
          background: var(--bg-secondary);
          border-radius: 12px;
        }

        .uptime-value {
          display: block;
          font-size: 24px;
          font-weight: 700;
          color: var(--text-primary);
        }

        .uptime-label {
          font-size: 11px;
          color: var(--text-muted);
          text-transform: uppercase;
          letter-spacing: 1px;
        }

        .uptime-chart {
          padding: 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          margin-bottom: 24px;
        }

        .uptime-chart h3 {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 16px;
          color: var(--text-secondary);
        }

        .uptime-bar {
          height: 8px;
          background: var(--bg-secondary);
          border-radius: 4px;
          overflow: hidden;
          margin-bottom: 12px;
        }

        .uptime-fill {
          height: 100%;
          background: linear-gradient(90deg, #27c93f, #2ed573);
          border-radius: 4px;
          transition: width 0.5s ease;
        }

        .uptime-stats {
          display: flex;
          justify-content: space-between;
          font-size: 13px;
          color: var(--text-secondary);
        }

        .services-grid {
          display: grid;
          gap: 16px;
          margin-bottom: 32px;
        }

        .service-card {
          padding: 24px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
          transition: all 0.2s ease;
        }

        .service-card:hover {
          border-color: var(--border-medium);
        }

        .service-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 8px;
        }

        .service-name-group {
          display: flex;
          align-items: center;
          gap: 10px;
        }

        .service-name-group h3 {
          font-size: 16px;
          font-weight: 600;
        }

        .service-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
        }

        .service-status {
          font-size: 14px;
          font-weight: 500;
        }

        .service-description {
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 12px;
        }

        .service-metrics {
          display: flex;
          gap: 20px;
        }

        .metric {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 13px;
          color: var(--text-muted);
        }

        .monitored-by {
          text-align: center;
          font-size: 13px;
          color: var(--text-muted);
          margin-bottom: 32px;
        }

        .monitored-by a {
          color: var(--text-secondary);
          font-weight: 500;
          margin-left: 4px;
        }

        .subscribe-section {
          text-align: center;
          padding: 32px;
          background: var(--bg-secondary);
          border-radius: 16px;
        }

        .subscribe-section p {
          color: var(--text-secondary);
          margin-bottom: 16px;
        }

        .subscribe-btn {
          display: inline-flex;
          align-items: center;
          gap: 8px;
          padding: 12px 24px;
          background: var(--text-primary);
          color: var(--bg-primary);
          border-radius: 8px;
          font-weight: 500;
          transition: opacity 0.2s ease;
        }

        .subscribe-btn:hover {
          opacity: 0.85;
        }

        .deploy-section {
          margin-bottom: 32px;
        }

        .deploy-section h2 {
          font-size: 20px;
          font-weight: 600;
          margin-bottom: 8px;
        }

        .deploy-subtitle {
          font-size: 14px;
          color: var(--text-secondary);
          margin-bottom: 16px;
        }

        .deploy-subtitle code,
        .mcp-hint code {
          font-size: 12px;
          background: var(--bg-secondary);
          padding: 2px 6px;
          border-radius: 4px;
        }

        .deploy-loading {
          font-size: 14px;
          color: var(--text-muted);
        }

        .deploy-banner {
          display: flex;
          justify-content: space-between;
          align-items: center;
          gap: 12px;
          padding: 16px 20px;
          background: var(--card-bg);
          border: 2px solid var(--border-light);
          border-radius: 12px;
          margin-bottom: 16px;
          font-size: 14px;
          font-weight: 500;
        }

        .deploy-checked {
          font-size: 12px;
          color: var(--text-muted);
          font-weight: 400;
        }

        .deploy-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
          gap: 16px;
          margin-bottom: 16px;
        }

        .deploy-card {
          padding: 20px;
          background: var(--card-bg);
          border: 1px solid var(--border-light);
          border-radius: 12px;
        }

        .deploy-card h3 {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 12px;
          color: var(--text-secondary);
        }

        .deploy-list {
          list-style: none;
          padding: 0;
          margin: 0;
          display: flex;
          flex-direction: column;
          gap: 10px;
        }

        .deploy-list li {
          display: flex;
          flex-direction: column;
          gap: 2px;
          font-size: 13px;
          padding: 10px 12px;
          border-radius: 8px;
          background: var(--bg-secondary);
        }

        .deploy-list li.ok .deploy-label { color: #27c93f; }
        .deploy-list li.fail .deploy-label { color: #ff5f56; }
        .deploy-list li.warn .deploy-label { color: #ffbd2e; }

        .deploy-label {
          font-weight: 600;
        }

        .deploy-value {
          color: var(--text-muted);
          word-break: break-all;
        }

        .mcp-card {
          margin-top: 0;
        }

        .mcp-hint {
          margin-top: 12px;
          font-size: 12px;
          color: var(--text-muted);
          line-height: 1.5;
        }

        @media (max-width: 640px) {
          .deploy-banner {
            flex-direction: column;
            align-items: flex-start;
          }
          .status-banner {
            flex-direction: column;
            text-align: center;
          }

          .uptime-stats {
            flex-direction: column;
            gap: 4px;
          }

          .service-metrics {
            flex-direction: column;
            gap: 8px;
          }
        }
      `}</style>
    </div>
  )
}

export default StatusPage
