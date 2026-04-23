# Hystersis Skills

> CLI tool and SDK for managing Hystersis memory skills - add, list, suggest, and use skills for AI agents.

## Installation

### Via NPX (Recommended)

```bash
npx @hystersis/skills install Hyman-D/hystersis-skills
```

### Via NPM

```bash
npm install -g @hystersis/skills
```

## Usage

### CLI

```bash
# Install skills from GitHub
npx @hystersis/skills install Hyman-D/hystersis-skills

# Add a skill
npx @hystersis/skills add

# List all skills
npx @hystersis/skills list

# Search for skills
npx @hystersis/skills search "code review"

# Get suggestions
npx @hystersis/skills suggest "I need help with debugging"
```

### SDK

```javascript
const { SkillsClient } = require('@hystersis/skills');

const client = new SkillsClient('http://localhost:8080');

// Add a skill
const skill = await client.addSkill('my-skill', 'Code Review', 'development', 0.8);

// List all skills
const skills = await client.listSkills();

// Search for skills
const results = await client.searchSkills('code review');

// Get suggestions
const suggestions = await client.suggestSkills('I need help debugging', 'coding context', 5);

// Extract skills from content
const extracted = await client.extractSkills('This is about Python programming...', 'user123');

// Execute a skill
const result = await client.executeSkill('skill123', { context: 'debugging' });
```

## Built-in Skills

| Skill | Description |
|-------|-------------|
| api-designer | API design and RESTful services |
| devops-engineer | DevOps, CI/CD, deployment |
| prompt-engineer | LLM prompt optimization |
| security-pro | Security auditing |
| sql-expert | SQL and database optimization |
| testing-pro | Testing strategies |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /skills | List all skills |
| POST | /skills | Add a new skill |
| GET | /skills/:id | Get skill by ID |
| PUT | /skills/:id | Update skill |
| DELETE | /skills/:id | Delete skill |
| GET | /skills/search | Search skills |
| POST | /skills/suggest | Get skill suggestions |
| POST | /skills/extract | Extract skills from content |
| POST | /skills/:id/execute | Execute a skill |
| GET | /skills/review | Get pending reviews |
| POST | /skills/review | Submit a review |

## License

MIT

## Author

Himan-D <himan@hystersis.ai>
