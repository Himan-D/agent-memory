export const demoApi = {
  // Chat with agents
  chat: async (message, sessionId, withMemory) => {
    const response = await fetch('/demo/chat', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        message,
        session_id: sessionId,
        with_memory: withMemory,
      }),
    })
    
    if (!response.ok) {
      throw new Error(`Chat failed: ${response.status}`)
    }
    
    return response.json()
  },
  
  // Get dashboard data
  getDashboard: async () => {
    const response = await fetch('/demo/dashboard')
    if (!response.ok) {
      throw new Error(`Dashboard failed: ${response.status}`)
    }
    return response.json()
  },
  
  // Create a new demo session
  createSession: async () => {
    const response = await fetch('/demo/session', {
      method: 'POST',
    })
    
    if (!response.ok) {
      throw new Error(`Create session failed: ${response.status}`)
    }
    
    return response.json()
  },
  
  // Get session messages
  getSession: async (sessionId) => {
    const response = await fetch(`/demo/session/${sessionId}`)
    if (!response.ok) {
      throw new Error(`Get session failed: ${response.status}`)
    }
    return response.json()
  },
  
  // Clear/delete a session
  clearSession: async (sessionId) => {
    const response = await fetch(`/demo/session/${sessionId}`, {
      method: 'DELETE',
    })
    
    if (!response.ok) {
      throw new Error(`Clear session failed: ${response.status}`)
    }
    
    return response.json()
  }
}