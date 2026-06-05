#!/usr/bin/env node
/**
 * Hystersis Skills CLI
 * Usage: skills <command> [options]
 *
 * Commands:
 *   add       Add a new skill
 *   list      List all skills
 *   search    Search skills by trigger
 *   suggest   Get skill suggestions
 *   extract   Extract skills from content
 *   get       Get skill by ID
 *   update    Update a skill
 *   delete    Delete a skill
 *   execute   Execute a skill
 *   review    Review a skill
 *   install   Install skills from GitHub
 *   help      Show this help
 */

const { SkillsClient } = require('../src/index');

const API_BASE = process.env.HYSTERESIS_API_BASE || process.env.HYSTERESIS_URL || 'http://localhost:8080';
const API_KEY = process.env.HYSTERESIS_API_KEY || '';

const client = new SkillsClient(API_BASE);
if (API_KEY) {
  client.client.defaults.headers['X-API-Key'] = API_KEY;
}

async function main() {
  const args = process.argv.slice(2);
  const cmd = args[0] || 'help';

  switch (cmd) {
    case 'add': {
      const [trigger, action, name, domain, confidence] = args.slice(1);
      if (!trigger || !action) {
        console.error('Usage: skills add <trigger> <action> [name] [domain] [confidence]');
        console.error('  trigger    — what activates this skill (e.g. "fix bug")');
        console.error('  action     — what the skill does (e.g. "debug and fix")');
        process.exit(1);
      }
      const result = await client.addSkill(trigger, action, name || null, domain || null, parseFloat(confidence) || 0.5);
      console.log(JSON.stringify(result, null, 2));
      break;
    }

    case 'list': {
      const domain = args[1] || null;
      const limit = parseInt(args[2]) || 100;
      const skills = await client.listSkills(domain, limit);
      if (!skills || skills.length === 0) {
        console.log('No skills found.');
        return;
      }
      console.log(`\n  ${skills.length} skill(s)${domain ? ` (domain: ${domain})` : ''}:\n`);
      for (const s of skills) {
        const name = s.name || s.trigger || 'unnamed';
        console.log(`  ${s.id ? s.id.substring(0, 8) + '…' : '???'}  ${name}`);
        if (s.trigger) console.log(`       trigger: ${s.trigger}`);
        if (s.domain) console.log(`       domain:  ${s.domain}`);
        if (s.confidence) console.log(`       conf:    ${(s.confidence * 100).toFixed(0)}%`);
        console.log('');
      }
      break;
    }

    case 'search': {
      const query = args.slice(1).join(' ');
      if (!query) {
        console.error('Usage: skills search <query>');
        process.exit(1);
      }
      const results = await client.searchSkills(query);
      console.log(JSON.stringify(results, null, 2));
      break;
    }

    case 'suggest': {
      const trigger = args.slice(1).join(' ');
      if (!trigger) {
        console.error('Usage: skills suggest <trigger> [context]');
        process.exit(1);
      }
      const result = await client.suggestSkills(trigger);
      console.log(JSON.stringify(result, null, 2));
      break;
    }

    case 'extract': {
      const content = args.slice(1).join(' ');
      if (!content) {
        console.error('Usage: skills extract <content>');
        console.error('  Reads content and extracts skill suggestions from it');
        process.exit(1);
      }
      const result = await client.extractSkills(content);
      console.log(JSON.stringify(result, null, 2));
      break;
    }

    case 'get': {
      const id = args[1];
      if (!id) {
        console.error('Usage: skills get <skill-id>');
        process.exit(1);
      }
      const skill = await client.getSkill(id);
      console.log(JSON.stringify(skill, null, 2));
      break;
    }

    case 'update': {
      const id = args[1];
      if (!id || args.length < 4) {
        console.error('Usage: skills update <skill-id> <field> <value>');
        console.error('  Example: skills update sk-123 name "new name"');
        process.exit(1);
      }
      const updates = {};
      for (let i = 2; i < args.length; i += 2) {
        updates[args[i]] = args[i + 1];
      }
      const result = await client.updateSkill(id, updates);
      console.log(JSON.stringify(result, null, 2));
      break;
    }

    case 'delete': {
      const id = args[1];
      if (!id) {
        console.error('Usage: skills delete <skill-id>');
        process.exit(1);
      }
      await client.deleteSkill(id);
      console.log('Skill deleted.');
      break;
    }

    case 'execute': {
      const id = args[1];
      const context = args.slice(2).join(' ');
      if (!id) {
        console.error('Usage: skills execute <skill-id> [context]');
        process.exit(1);
      }
      const result = await client.executeSkill(id, { context });
      console.log(JSON.stringify(result, null, 2));
      break;
    }

    case 'review': {
      const id = args[1];
      const approved = args[2] !== 'false' && args[2] !== 'reject';
      const notes = args.slice(3).join(' ') || null;
      if (!id) {
        console.error('Usage: skills review <skill-id> [approve|reject] [notes]');
        process.exit(1);
      }
      const result = await client.reviewSkill(id, approved, notes);
      console.log(JSON.stringify(result, null, 2));
      break;
    }

    case 'install': {
      const repo = args[1];
      if (!repo) {
        console.error('Usage: skills install <github-repo>');
        console.error('  Example: skills install Himan-D/hystersis-skills');
        process.exit(1);
      }
      console.log(`Installing skills from ${repo}...`);
      const url = `https://api.github.com/repos/${repo}/contents/skills`;
      try {
        const axios = require('axios');
        const res = await axios.get(url);
        const files = res.data.filter(f => f.name.endsWith('.md'));
        for (const file of files) {
          const content = (await axios.get(file.download_url)).data;
          const extracted = await client.extractSkills(content);
          console.log(`  ✓ ${file.name.replace('.md', '')}`);
        }
        console.log(`\nInstalled ${files.length} skill(s) from ${repo}`);
      } catch {
        console.error(`Failed to install from ${repo}. Make sure the repo exists and has a skills/ folder.`);
        process.exit(1);
      }
      break;
    }

    case 'help':
    default: {
      console.log(`
  Hystersis Skills CLI — Manage AI agent skills

  Usage:
    skills <command> [options]

  Commands:
    add <trigger> <action> [name] [domain] [confidence]
        Create a new skill

    list [domain] [limit]
        List all skills (optionally filtered by domain)

    search <query>
        Search skills by trigger keyword

    suggest <trigger> [context]
        Get LLM-powered skill suggestions

    extract <content>
        Extract skill suggestions from content

    get <skill-id>
        Get skill details by ID

    update <skill-id> <field> <value> [...]
        Update skill fields

    delete <skill-id>
        Delete a skill

    execute <skill-id> [context]
        Execute a skill via LLM

    review <skill-id> [approve|reject] [notes]
        Approve or reject a skill

    install <github-repo>
        Install skills from a GitHub repo

    help
        Show this help

  Environment:
    HYSTERESIS_API_BASE   API server URL (default: http://localhost:8080)
    HYSTERESIS_API_KEY    API key for authentication
    HYSTERESIS_URL        Alias for HYSTERESIS_API_BASE

  Examples:
    skills add "fix-bug" "Debug and fix the issue" debugging dev 0.8
    skills list
    skills search "database"
    skills suggest "I need to optimize a query"
    skills install Himan-D/hystersis-skills
`);
      break;
    }
  }
}

main().catch(err => {
  console.error('Error:', err.response?.data || err.message);
  process.exit(1);
});
