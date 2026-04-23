import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import AgentChat from '../components/AgentChat'
import DemoDashboard from '../components/DemoDashboard'
import { demoApi } from '../utils/api'

function DemoPage() {
  const [withMemorySession, setWithMemorySession] = useState('')
  const [withoutMemorySession, setWithoutMemorySession] = useState('')
  const [messagesWithMemory, setMessagesWithMemory] = useState([])
  const [messagesWithoutMemory, setMessagesWithoutMemory] = useState([])
  const [dashboardData, setDashboardData] = useState(null)
  const [isTyping, setIsTyping] = useState(false)
  const [inputValue, setInputValue] = useState('')
  const inputRef = useRef(null)

  // Initialize sessions on mount
  useEffect(() => {
    const initSessions = async () => {
      try {
        const withResp = await fetch('/demo/session', { method: 'POST' })
        const withData = await withResp.json()
        setWithMemorySession(withData.session_id)

        const withoutResp = await fetch('/demo/session', { method: 'POST' })
        const withoutData = await withoutResp.json()
        setWithoutMemorySession(withoutData.session_id)

        // Load initial messages for both sessions
        loadMessages(withData.session_id, true)
        loadMessages(withoutData.session_id, false)
      } catch (err) {
        console.error('Failed to init sessions:', err)
      }
    }

    initSessions()
    const dashboardInterval = setInterval(fetchDashboard, 10000)
    fetchDashboard()
    return () => clearInterval(dashboardInterval)
  }, [])

  // Load messages for a session
  const loadMessages = async (sessionId, withMemory) => {
    try {
      const resp = await fetch(`/demo/session/${sessionId}`)
      const data = await resp.json()
      if (withMemory) {
        setMessagesWithMemory(data.messages)
      } else {
        setMessagesWithoutMemory(data.messages)
      }
    } catch (err) {
      console.error('Failed to load messages:', err)
    }
  }

  // Fetch dashboard data
  const fetchDashboard = async () => {
    try {
      const resp = await fetch('/demo/dashboard')
      const data = await resp.json()
      setDashboardData(data)
    } catch (err) {
      console.error('Failed to fetch dashboard:', err)
    }
  }

  // Handle sending message
  const handleSendMessage = async (e) => {
    e.preventDefault()
    if (!inputValue.trim()) return

    const message = inputValue.trim()
    setInputValue('')
    setIsTyping(true)

    try {
      // Send to both agents in parallel
      const [withResp, withoutResp] = await Promise.all([
        fetch('/demo/chat', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            message,
            session_id: withMemorySession,
            with_memory: true,
          }),
        }),
        fetch('/demo/chat', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            message,
            session_id: withoutMemorySession,
            with_memory: false,
          }),
        }),
      ])

      const withData = await withResp.json()
      const withoutData = await withoutResp.json()

      // Update message lists
      setMessagesWithMemory(prev => [
        ...prev,
        { role: 'user', content: message },
        { role: 'assistant', content: withData.response, memories: withData.retrieved_memories || [] }
      ])

      setMessagesWithoutMemory(prev => [
        ...prev,
        { role: 'user', content: message },
        { role: 'assistant', content: withoutData.response }
      ])

      // Scroll to bottom
      setTimeout(() => {
        const withChat = document.getElementById('with-memory-chat')
        const withoutChat = document.getElementById('without-memory-chat')
        if (withChat) withChat.scrollTop = withChat.scrollHeight
        if (withoutChat) withoutChat.scrollTop = withoutChat.scrollHeight
      }, 100)
    } catch (err) {
      console.error('Failed to send message:', err)
    } finally {
      setIsTyping(false)
    }
  }

  // Clear chat
  const handleClearChat = async () => {
    try {
      await fetch(`/demo/session/${withMemorySession}`, { method: 'DELETE' })
      await fetch(`/demo/session/${withoutMemorySession}`, { method: 'DELETE' })
      
      // Create new sessions
      const [withResp, withoutResp] = await Promise.all([
        fetch('/demo/session', { method: 'POST' }),
        fetch('/demo/session', { method: 'POST' })
      ])
      
      const withData = await withResp.json()
      const withoutData = await withoutResp.json()
      
      setWithMemorySession(withData.session_id)
      setWithoutMemorySession(withoutData.session_id)
      setMessagesWithMemory([])
      setMessagesWithoutMemory([])
    } catch (err) {
      console.error('Failed to clear chat:', err)
    }
  }

  return (
    <section className="demo-page section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          className="demo-header"
        >
          <h1 className="demo-title">Agent Memory Comparison</h1>
          <p className="demo-description">
            See how AI agents perform with vs without persistent memory
          </p>
        </motion.div>

        <div className="demo-grid">
          {/* WITH MEMORY Agent */}
          <motion.div
            initial={{ opacity: 0, x: -50 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="agent-panel"
          >
            <div className="agent-header">
              <h2 className="agent-title">🤖 Agent WITH Memory</h2>
              <div className="agent-status">
                <span className="status-indicator" />
                <span className="status-text">Active</span>
              </div>
            </div>

            <AgentChat
              sessionId={withMemorySession}
              messages={messagesWithMemory}
              isTyping={isTyping}
              title="WITH MEMORY"
              onClear={handleClearChat}
            />

            <div className="agent-actions">
              <button className="btn btn-secondary" onClick={handleClearChat}>
                Clear Chat
              </button>
            </div>
          </motion.div>

          {/* WITHOUT MEMORY Agent */}
          <motion.div
            initial={{ opacity: 0, x: 50 }}
            whileInView={{ opacity: 1, x: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="agent-panel"
          >
            <div className="agent-header">
              <h2 className="agent-title">🤖 Agent WITHOUT Memory</h2>
              <div className="agent-status">
                <span className="status-indicator" />
                <span className="status-text">Stateless</span>
              </div>
            </div>

            <AgentChat
              sessionId={withoutMemorySession}
              messages={messagesWithoutMemory}
              isTyping={isTyping}
              title="WITHOUT MEMORY"
              onClear={handleClearChat}
            />

            <div className="agent-actions">
              <button className="btn btn-secondary" onClick={handleClearChat}>
                Clear Chat
              </button>
            </div>
          </motion.div>
        </div>

        {/* Dashboard */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.4 }}
          className="demo-dashboard-panel"
        >
          <DemoDashboard data={dashboardData} />
        </motion.div>

        {/* Message Input */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6, delay: 0.5 }}
          className="message-input-container"
        >
          <form onSubmit={handleSendMessage} className="message-input-form">
            <div className="message-input-group">
              <input
                ref={inputRef}
                type="text"
                placeholder="Ask both agents a question..."
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                disabled={isTyping}
                className="message-input"
              />
              <button
                type="submit"
                disabled={isTyping || !inputValue.trim()}
                className="message-btn"
              >
                {isTyping ? 'Sending...' : 'Send'}
              </button>
            </div>
            <p className="input-hint">
              Try asking: "What's my preferred programming language?" or 
              "Do you remember what we talked about earlier?"
            </p>
          </form>
        </motion.div>
      </div>
    </section>
  )
}

export default DemoPage