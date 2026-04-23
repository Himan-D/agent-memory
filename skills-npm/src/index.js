/**
 * Hystersis Skills Management SDK
 * 
 * Core SDK for managing skills - add, list, suggest, and use skills for AI agents.
 */

const axios = require('axios');

const API_BASE = process.env.HYSTERESIS_API_BASE || 'http://localhost:8080';

class SkillsClient {
  constructor(baseURL = API_BASE) {
    this.baseURL = baseURL;
    this.client = axios.create({
      baseURL,
      timeout: 30000,
    });
  }

  async addSkill(source, name = null, domain = null, confidence = 0.5) {
    const response = await this.client.post('/skills', {
      source,
      name,
      domain,
      confidence,
    });
    return response.data;
  }

  async getSkill(id) {
    const response = await this.client.get(`/skills/${id}`);
    return response.data;
  }

  async listSkills(domain = null, limit = 100) {
    const params = { limit };
    if (domain) {
      params.domain = domain;
    }
    const response = await this.client.get('/skills', { params });
    return response.data;
  }

  async searchSkills(query, limit = 10) {
    const response = await this.client.get('/skills/search', {
      params: { query, limit },
    });
    return response.data;
  }

  async suggestSkills(trigger, context = null, limit = 5) {
    const response = await this.client.post('/skills/suggest', {
      trigger,
      context,
      limit,
    });
    return response.data;
  }

  async extractSkills(content, userID = null) {
    const response = await this.client.post('/skills/extract', {
      content,
      user_id: userID,
    });
    return response.data;
  }

  async updateSkill(id, updates) {
    const response = await this.client.put(`/skills/${id}`, updates);
    return response.data;
  }

  async deleteSkill(id) {
    const response = await this.client.delete(`/skills/${id}`);
    return response.data;
  }

  async getSkillByTrigger(trigger) {
    const response = await this.client.get('/skills/search', {
      params: { trigger },
    });
    return response.data;
  }

  async getSkillsByDomain(domain) {
    const response = await this.client.get('/skills', {
      params: { domain },
    });
    return response.data;
  }

  async executeSkill(id, context = {}) {
    const response = await this.client.post(`/skills/${id}/execute`, context);
    return response.data;
  }

  async reviewSkill(id, approved = true, feedback = null) {
    const response = await this.client.post('/skills/review', {
      id,
      approved,
      feedback,
    });
    return response.data;
  }
}

async function addSkill(source, name = null, domain = null, confidence = 0.5) {
  const client = new SkillsClient();
  return client.addSkill(source, name, domain, confidence);
}

async function listSkills(domain = null, limit = 100) {
  const client = new SkillsClient();
  return client.listSkills(domain, limit);
}

async function searchSkills(query, limit = 10) {
  const client = new SkillsClient();
  return client.searchSkills(query, limit);
}

async function suggestSkills(trigger, context = null, limit = 5) {
  const client = new SkillsClient();
  return client.suggestSkills(trigger, context, limit);
}

async function extractSkills(content, userID = null) {
  const client = new SkillsClient();
  return client.extractSkills(content, userID);
}

async function executeSkill(id, context = {}) {
  const client = new SkillsClient();
  return client.executeSkill(id, context);
}

module.exports = {
  SkillsClient,
  addSkill,
  listSkills,
  searchSkills,
  suggestSkills,
  extractSkills,
  executeSkill,
};