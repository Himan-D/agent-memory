import { useState } from 'react'
import { motion } from 'framer-motion'

const apiEndpoints = [
  {
    method: 'POST',
    path: '/sessions',
    description: 'Create a new agent session',
    body: `{ "agent_id": "my-agent", "metadata": {} }`,
    response: `{ "id": "session-123", "agent_id": "my-agent", "created_at": "2025-01-15T10:00:00Z" }`
  },
  {
    method: 'POST',
    path: '/sessions/{id}/messages',
    description: 'Add a message to a session',
    body: `{ "role": "user", "content": "Hello!" }`,
    response: `{ "id": "msg-456", "session_id": "session-123", "role": "user", "content": "Hello!" }`
  },
  {
    method: 'GET',
    path: '/sessions/{id}/messages',
    description: 'Get session message history',
    body: null,
    response: `[{ "id": "msg-1", "role": "user", "content": "Hello!" }, { "id": "msg-2", "role": "assistant", "content": "Hi!" }]`
  },
  {
    method: 'POST',
    path: '/entities',
    description: 'Create a knowledge graph entity',
    body: `{ "name": "UserService", "type": "Class", "properties": { "file": "user.py" } }`,
    response: `{ "id": "entity-789", "name": "UserService", "type": "Class" }`
  },
  {
    method: 'GET',
    path: '/entities/{id}',
    description: 'Get entity with relationships',
    body: null,
    response: `{ "id": "entity-789", "name": "UserService", "type": "Class", "relations": [] }`
  },
  {
    method: 'POST',
    path: '/relations',
    description: 'Create a relationship between entities',
    body: `{ "from": "entity-1", "to": "entity-2", "type": "USES" }`,
    response: `{ "from": "entity-1", "to": "entity-2", "type": "USES" }`
  },
  {
    method: 'GET',
    path: '/search',
    description: 'Semantic vector search',
    body: null,
    response: `[{ "id": "msg-123", "content": "I love ML!", "score": 0.92 }]`
  },
  {
    method: 'POST',
    path: '/graph/query',
    description: 'Raw Cypher query (admin only)',
    body: `{ "query": "MATCH (n) RETURN n LIMIT 5" }`,
    response: `[{ "n": { "id": "1", "name": "test" } }]`
  }
]

const quickstartSteps = [
  {
    step: '1',
    title: 'Install the SDK',
    code: 'pip install agentmemory'
  },
  {
    step: '2',
    title: 'Connect to your server',
    code: `from agentmemory import AgentMemory

client = AgentMemory(
    "https://api.yourserver.com", 
    api_key="your-key"
)`
  },
  {
    step: '3',
    title: 'Create a session',
    code: `session = client.create_session(
    agent_id="my-bot"
)`
  },
  {
    step: '4',
    title: 'Add messages',
    code: `client.add_message(
    session["id"], 
    "user", 
    "Hello!"
)

client.add_message(
    session["id"],
    "assistant",
    "Hi there!"
)`
  },
  {
    step: '5',
    title: 'Search memories',
    code: `results = client.semantic_search(
    "machine learning"
)`
  }
]

function DocsPage() {
  const [activeTab, setActiveTab] = useState('quickstart')

  return (
    <div className="docs-page">
      <motion.div 
        className="page-hero"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <div className="container">
          <span className="section-label">Docs</span>
          <h1>API Reference</h1>
          <p>Everything you need to integrate Agent Memory into your applications.</p>
        </div>
      </motion.div>

      <div className="container">
        <div className="docs-tabs">
          <button 
            className={`tab ${activeTab === 'quickstart' ? 'active' : ''}`}
            onClick={() => setActiveTab('quickstart')}
          >
            Quick Start
          </button>
          <button 
            className={`tab ${activeTab === 'api' ? 'active' : ''}`}
            onClick={() => setActiveTab('api')}
          >
            API Endpoints
          </button>
        </div>

        {activeTab === 'quickstart' && (
          <motion.div 
            className="quickstart-section"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.3 }}
          >
            <div className="steps-list">
              {quickstartSteps.map((step) => (
                <div key={step.step} className="step-item">
                  <div className="step-header">
                    <span className="step-number">{step.step}</span>
                    <span className="step-title">{step.title}</span>
                  </div>
                  <pre><code>{step.code}</code></pre>
                </div>
              ))}
            </div>
          </motion.div>
        )}

        {activeTab === 'api' && (
          <motion.div 
            className="api-section"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.3 }}
          >
            <div className="endpoints-list">
              {apiEndpoints.map((endpoint, index) => (
                <div key={index} className="endpoint-card">
                  <div className="endpoint-header">
                    <span className={`method ${endpoint.method.toLowerCase()}`}>
                      {endpoint.method}
                    </span>
                    <span className="path">{endpoint.path}</span>
                  </div>
                  <p className="endpoint-description">{endpoint.description}</p>
                  
                  {endpoint.body && (
                    <div className="endpoint-body">
                      <h4>Request Body</h4>
                      <pre><code>{endpoint.body}</code></pre>
                    </div>
                  )}
                  
                  <div className="endpoint-response">
                    <h4>Response</h4>
                    <pre><code>{endpoint.response}</code></pre>
                  </div>
                </div>
              ))}
            </div>
          </motion.div>
        )}
      </div>

      <style>{`
        .docs-page {
          padding-bottom: 80px;
        }

        .page-hero {
          padding: 80px 0 60px;
          text-align: center;
          border-bottom: 1px solid var(--border-light);
        }

        .page-hero h1 {
          font-family: var(--font-display);
          font-size: clamp(36px, 6vw, 56px);
          font-weight: 800;
          margin-bottom: 16px;
          letter-spacing: -2px;
        }

        .page-hero p {
          font-size: 18px;
          color: var(--text-secondary);
          max-width: 500px;
          margin: 0 auto;
        }

        .docs-tabs {
          display: flex;
          gap: 8px;
          padding: 24px 0;
          border-bottom: 1px solid var(--border-light);
          margin-bottom: 32px;
        }

        .tab {
          padding: 12px 24px;
          background: none;
          border: none;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-secondary);
          cursor: pointer;
          border-radius: 8px;
          transition: all 0.2s ease;
        }

        .tab:hover {
          background: var(--bg-surface);
        }

        .tab.active {
          background: var(--color-primary);
          color: white;
        }

        .steps-list {
          display: flex;
          flex-direction: column;
          gap: 24px;
        }

        .step-item {
          padding: 24px;
          background: var(--bg-surface);
          border: 1px solid var(--border-light);
          border-radius: 12px;
        }

        .step-header {
          display: flex;
          align-items: center;
          gap: 12px;
          margin-bottom: 16px;
        }

        .step-number {
          width: 28px;
          height: 28px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: #2563EB;
          color: white;
          font-size: 14px;
          font-weight: 600;
          border-radius: 50%;
        }

        .step-title {
          font-family: var(--font-display);
          font-size: 16px;
          font-weight: 600;
        }

        .step-item pre {
          margin: 0;
          overflow-x: auto;
        }

        .step-item code {
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 14px;
          color: var(--text-secondary);
        }

        .endpoints-list {
          display: flex;
          flex-direction: column;
          gap: 24px;
        }

        .endpoint-card {
          padding: 24px;
          background: var(--bg-surface);
          border: 1px solid var(--border-light);
          border-radius: 12px;
        }

        .endpoint-header {
          display: flex;
          align-items: center;
          gap: 12px;
          margin-bottom: 12px;
        }

        .method {
          padding: 6px 12px;
          font-size: 12px;
          font-weight: 600;
          border-radius: 6px;
        }

        .method.post {
          background: #DCFCE7;
          color: #166534;
        }

        .method.get {
          background: #DBEAFE;
          color: #1E40AF;
        }

        .path {
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 14px;
          font-weight: 500;
        }

        .endpoint-description {
          color: var(--text-secondary);
          margin-bottom: 16px;
        }

        .endpoint-body h4,
        .endpoint-response h4 {
          font-size: 12px;
          font-weight: 600;
          margin-bottom: 8px;
          color: var(--text-muted);
        }

        .endpoint-body pre,
        .endpoint-response pre {
          margin: 0 0 16px;
          overflow-x: auto;
        }

        .endpoint-body code,
        .endpoint-response code {
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 13px;
          color: var(--text-secondary);
        }
      `}</style>
    </div>
  )
}

export default DocsPage