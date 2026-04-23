import { useState, useEffect, useRef } from 'react'
import { motion } from 'framer-motion'

function AgentChat({ sessionId, messages, isTyping, title, onClear }) {
  const messagesEndRef = useRef(null)
  const [agentMessages, setAgentMessages] = useState([])

  // Sync incoming messages
  useEffect(() => {
    setAgentMessages(messages)
  }, [messages])

  // Scroll to bottom when messages update
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [agentMessages])

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6 }}
      className="agent-chat-container"
    >
      <div className="agent-chat-header">
        <h3 className="agent-chat-title">{title}</h3>
        <div className="agent-chat-status">
          <div className="status-dot" />
          <span className="status-text">{isTyping ? 'Typing...' : 'Ready'}</span>
        </div>
      </div>

      <div className="agent-chat-messages" ref={messagesEndRef}>
        {agentMessages.map((msg, index) => (
          <motion.div
            key={index}
            initial={{ opacity: 0, x: msg.role === 'assistant' ? -20 : 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.3 }}
            className={`message-bubble ${msg.role === 'assistant' ? 'assistant-msg' : 'user-msg'}`}
          >
            <div className="message-content">
              <p>{msg.content}</p>
              {msg.memories && msg.memories.length > 0 && (
                <div className="memories-used">
                  <span className="memories-label">💡 Used {msg.memories.length} memory facts:</span>
                  <ul className="memories-list">
                    {msg.memories.map((mem, idx) => (
                      <li key={idx} className="memory-item">
                        "{mem.Content}" <span className="memory-score">(relevance: {Math.round(mem.Score * 100)}%)</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          </motion.div>
        ))}
      </div>

      <div className="agent-chat-footer">
        <button 
          className="btn btn-outline btn-sm" 
          onClick={onClear}
          disabled={isTyping}
        >
          Clear Chat
        </button>
      </div>
    </motion.div>
  )
}

export default AgentChat