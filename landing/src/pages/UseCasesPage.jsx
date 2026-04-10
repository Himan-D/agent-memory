import { motion } from 'framer-motion'

const useCases = [
  {
    id: 'customer-support',
    title: 'Customer Support Bot',
    description: 'Support bots that remember past tickets and customer history.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0"/>
      </svg>
    ),
    problem: 'Support bots repeat themselves and don\'t remember customer history.',
    solution: 'Store every interaction and retrieve relevant context.',
    code: `from agentmemory import AgentMemory

client = AgentMemory("https://api.yourserver.com", api_key="support-key")

# When customer starts a new conversation
session = client.create_session(
    agent_id="support-bot",
    metadata={"customer_id": "CUST-123", "tier": "premium"}
)

# Store each interaction
client.add_message(session["id"], "user", "I can't login to my account")
client.add_message(session["id"], "assistant", "I'll help you. What error do you see?")
client.add_message(session["id"], "user", "It says 'invalid password'")

# Later - when customer returns, find similar issues
past_issues = client.semantic_search("can't login invalid password", limit=5)
# Returns similar issues from other customers`,
    result: 'Bot can say "I see similar login issues were resolved by resetting passwords..."'
  },
  {
    id: 'code-assistant',
    title: 'Code Assistant',
    description: 'Developer tools that understand your codebase patterns.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M16 18l6-6-6-6M8 6l-6 6 6 6"/>
      </svg>
    ),
    problem: 'Developer agents don\'t understand your codebase\'s patterns.',
    solution: 'Index code, docs, and past solutions.',
    code: `# Index codebase entities
client.create_entity(
    name="auth-service",
    type="Service",
    properties={"language": "python", "port": 8080}
)

client.create_entity(
    name="UserService",
    type="Class",
    properties={"file": "services/user.py", "methods": ["login", "logout"]}
)

# Create relationships
client.create_relation("auth-service", "UserService", "USES")

# Later - when developer asks about auth
results = client.semantic_search("how does user authentication work")
# Returns semantically similar code/docs`,
    result: 'Agent understands code structure and can provide relevant context.'
  },
  {
    id: 'research-agent',
    title: 'Research Agent',
    description: 'Agents that connect ideas across papers and notes.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M4 19.5A2.5 2.5 0 016.5 17H20"/>
        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z"/>
      </svg>
    ),
    problem: 'Research agents can\'t connect ideas across papers and notes.',
    solution: 'Build a knowledge graph of research.',
    code: `# Add paper as entity
paper = client.create_entity(
    name="Attention Is All You Need",
    type="Paper",
    properties={
        "authors": ["Vaswani", "Shazeer", "Parmar"],
        "year": 2017
    }
)

# Add concepts
client.create_entity(name="Transformer", type="Concept")
client.create_entity(name="Self-Attention", type="Concept")
client.create_entity(name="Seq2Seq", type="Concept")

# Connect them
client.create_relation("Transformer", "Self-Attention", "USES")
client.create_relation("Transformer", "Seq2Seq", "IMPROVES")

# Find related work
related = client.semantic_search("attention mechanism neural network")`,
    result: 'Build a literature graph and find related papers automatically.'
  },
  {
    id: 'personal-assistant',
    title: 'Personal AI Assistant',
    description: 'Assistants that remember preferences and important dates.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
        <circle cx="12" cy="7" r="4"/>
      </svg>
    ),
    problem: 'Personal assistants forget preferences and important events.',
    solution: 'Store preferences, memories, and context.',
    code: `# Remember preferences
client.create_entity(
    name="user_preferences",
    type="Preference",
    properties={
        "coffee": "black, no sugar",
        "meetings": "mornings preferred",
        "diet": "vegetarian"
    }
)

# Remember important dates
client.create_entity(
    name="anniversary",
    type="Event",
    properties={"date": "2025-06-15", "description": "Wedding anniversary"}
)

# Store conversations about topics
session = client.create_session(agent_id="personal-assistant")
client.add_message(session["id"], "user", "I want to learn Spanish")`,
    result: 'Assistant remembers preferences and important context.'
  },
  {
    id: 'multi-tenant',
    title: 'Multi-Tenant SaaS',
    description: 'Complete data isolation for multi-customer applications.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <path d="M9 3v18M3 9h18"/>
      </svg>
    ),
    problem: 'Need to separate data between customers.',
    solution: 'Use tenant IDs in API keys.',
    code: `# Server config: API_KEYS="key1:tenant1,key2:tenant2"
# Keys: "key1" maps to tenant1, "key2" maps to tenant2

# When customer A makes requests with their key
client_a = AgentMemory("https://api.com", api_key="key1")
session_a = client_a.create_session(agent_id="my-agent")
# Session is automatically tagged with tenant1

# Customer B with different key
client_b = AgentMemory("https://api.com", api_key="key2")
# Their data is completely isolated from customer A`,
    result: 'Complete tenant isolation with zero configuration.'
  },
  {
    id: 'sales-intelligence',
    title: 'Sales Intelligence',
    description: 'Build relationship graphs of accounts and contacts.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/>
        <circle cx="9" cy="7" r="4"/>
        <path d="M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75"/>
      </svg>
    ),
    problem: 'Sales bots don\'t remember deal history or relationships.',
    solution: 'Build a relationship graph of accounts.',
    code: `# Add companies
acme = client.create_entity(name="Acme Corp", type="Company")
competitor = client.create_entity(name="Globex", type="Company")

# Add contacts
alice = client.create_entity(name="Alice (Acme CTO)", type="Person")
bob = client.create_entity(name="Bob (Acme VP)", type="Person")

# Create relationships
client.create_relation("acme", "alice", "HAS_EMPLOYEE")
client.create_relation("acme", "bob", "HAS_EMPLOYEE")
client.create_relation("alice", "bob", "REPORTS_TO")
client.create_relation("acme", "competitor", "COMPETES_WITH")`,
    result: 'Query relationships: "Who do we know at Acme?"'
  },
  {
    id: 'tutoring',
    title: 'Educational Tutoring',
    description: 'Track student progress and adapt to learning styles.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z"/>
        <path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z"/>
      </svg>
    ),
    problem: 'Tutors don\'t adapt to student progress.',
    solution: 'Track learning progress and adapt.',
    code: `# Store student profile
student = client.create_entity(
    name="student-123",
    type="Student",
    properties={"level": "intermediate", "topics_covered": ["algebra", "geometry"]}
)

# Add lesson notes
client.create_entity(
    name="lesson-2024-01-15",
    type="Lesson",
    properties={"topic": "calculus", "duration": "60min", "difficulty": "hard"}
)

# Connect student to lesson
client.create_relation("student-123", "lesson-2024-01-15", "COMPLETED")

# Query: What topics should we review?
# Semantic search for topics marked as "difficult"`,
    result: 'Personalized tutoring based on student history.'
  },
  {
    id: 'gaming',
    title: 'Game AI Companion',
    description: 'NPCs that remember player interactions.',
    icon: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
        <rect x="2" y="6" width="20" height="12" rx="2"/>
        <path d="M6 12h.01M10 12h.01M14 12h.01M18 12h.01"/>
      </svg>
    ),
    problem: 'Game NPCs don\'t remember player interactions.',
    solution: 'Store player history and preferences.',
    code: `# Remember player decisions
session = client.create_session(agent_id="npc-companion")
client.add_message(session["id"], "player", "I'll take the healing potion")
client.add_message(session["id"], "npc", "Good choice! It could save you later")

# Store character preferences
client.create_entity(
    name="player-quest-log",
    type="QuestLog",
    properties={"completed": ["dragon_quest", "forest_quest"], "active": "castle_quest"}
)

# Remember player behavior patterns
results = client.semantic_search("attacked enemies without caution")
# NPC can adapt behavior based on past patterns`,
    result: 'Dynamic NPC behavior based on player history.'
  }
]

function UseCasesPage() {
  return (
    <div className="use-cases-page">
      <motion.div 
        className="page-hero"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5 }}
      >
        <div className="container">
          <span className="section-label">Use Cases</span>
          <h1>Real-world applications</h1>
          <p>See how teams are using Agent Memory to build smarter AI products.</p>
        </div>
      </motion.div>

      <div className="container">
        <div className="use-cases-grid">
          {useCases.map((useCase, index) => (
            <motion.div
              key={useCase.id}
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.1 * index }}
              className="use-case-card"
              id={useCase.id}
            >
              <div className="use-case-icon">{useCase.icon}</div>
              <div className="use-case-content">
                <h3>{useCase.title}</h3>
                <p>{useCase.description}</p>
                
                <div className="use-case-details">
                  <div className="detail">
                    <h4>Problem</h4>
                    <p>{useCase.problem}</p>
                  </div>
                  <div className="detail">
                    <h4>Solution</h4>
                    <p>{useCase.solution}</p>
                  </div>
                  <div className="detail">
                    <h4>Code</h4>
                    <pre><code>{useCase.code}</code></pre>
                  </div>
                  <div className="detail result">
                    <h4>Result</h4>
                    <p>{useCase.result}</p>
                  </div>
                </div>
              </div>
            </motion.div>
          ))}
        </div>
      </div>

      <style>{`
        .use-cases-page {
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

        .use-cases-grid {
          display: flex;
          flex-direction: column;
          gap: 48px;
          padding: 48px 0;
        }

        .use-case-card {
          display: grid;
          grid-template-columns: auto 1fr;
          gap: 32px;
          padding: 40px;
          background: var(--bg-surface);
          border: 1px solid var(--border-light);
          border-radius: 16px;
        }

        .use-case-icon {
          width: 64px;
          height: 64px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: rgba(37, 99, 235, 0.1);
          border-radius: 16px;
        }

        .use-case-icon svg {
          width: 32px;
          height: 32px;
          color: #2563EB;
        }

        .use-case-content h3 {
          font-family: var(--font-display);
          font-size: 24px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .use-case-content > p {
          color: var(--text-secondary);
          margin-bottom: 24px;
        }

        .use-case-details {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
          gap: 20px;
        }

        .detail {
          padding: 20px;
          background: var(--bg-primary);
          border-radius: 12px;
        }

        .detail h4 {
          font-size: 12px;
          font-weight: 600;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          color: #2563EB;
          margin-bottom: 8px;
        }

        .detail p {
          font-size: 14px;
          color: var(--text-secondary);
          line-height: 1.6;
        }

        .detail pre {
          overflow-x: auto;
          margin: 0;
        }

        .detail code {
          font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
          font-size: 12px;
          line-height: 1.6;
          white-space: pre-wrap;
        }

        .result {
          background: rgba(37, 99, 235, 0.05);
          border: 1px solid rgba(37, 99, 235, 0.1);
        }

        .result h4 {
          color: #2563EB;
        }

        @media (max-width: 768px) {
          .use-case-card {
            grid-template-columns: 1fr;
            gap: 20px;
          }

          .use-case-icon {
            width: 48px;
            height: 48px;
          }

          .use-case-icon svg {
            width: 24px;
            height: 24px;
          }
        }
      `}</style>
    </div>
  )
}

export default UseCasesPage