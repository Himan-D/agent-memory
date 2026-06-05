#!/usr/bin/env node
/**
 * Node SDK smoke test — mirrors scripts/smoke-test.sh.
 *
 * Usage:
 *   HYSTERSIS_BASE_URL=http://localhost:8080 HYSTERSIS_API_KEY=key node scripts/smoke-sdk.mjs
 */

import { HystersisClient } from '../sdk/nodejs/dist/index.js';

const BASE = process.env.HYSTERSIS_BASE_URL || process.env.AGENT_MEMORY_URL || 'http://localhost:8080';
const API_KEY = process.env.HYSTERSIS_API_KEY || process.env.AGENT_MEMORY_API_KEY || 'test-key';
const USER_ID = `node-smoke-${Date.now()}`;

let pass = 0;
let fail = 0;
let skip = 0;

const ok = (name) => { console.log(`  ✓ ${name}`); pass++; };
const bad = (name, detail = '') => { console.log(`  ✗ ${name}${detail ? ` — ${detail}` : ''}`); fail++; };
const skipped = (name, detail = '') => { console.log(`  ~ ${name} (skipped)${detail ? ` — ${detail}` : ''}`); skip++; };

async function run() {
  console.log('Hystersis Node SDK smoke test');
  console.log(`Base URL: ${BASE}\n`);

  const client = new HystersisClient({ baseUrl: BASE, apiKey: API_KEY });
  let memId = null;

  try {
    console.log('== Public ==');
    try {
      await client.health();
      ok('health()');
    } catch (e) {
      bad('health()', e.message);
      console.log('\nStart server or set HYSTERSIS_BASE_URL.');
      process.exit(1);
    }

    try {
      await client.ready();
      ok('ready()');
    } catch {
      skipped('ready()');
    }

    try {
      const plans = await client.getBillingPlans();
      if (plans.plans) ok('getBillingPlans()');
      else bad('getBillingPlans()', 'missing plans key');
    } catch (e) {
      skipped('getBillingPlans()', e.message);
    }

    console.log('\n== Memory CRUD ==');
    try {
      const mem = await client.createMemory({ content: 'node sdk smoke', user_id: USER_ID });
      memId = mem.id;
      ok(`createMemory() → ${memId}`);
    } catch (e) {
      bad('createMemory()', e.message);
    }

    if (memId) {
      try {
        await client.getMemory(memId);
        ok('getMemory()');
      } catch (e) {
        bad('getMemory()', e.message);
      }
    }

    console.log('\n== Search ==');
    try {
      await client.search('node sdk', { limit: 5 });
      ok('search()');
    } catch (e) {
      bad('search()', e.message);
    }

    try {
      await client.searchHybrid('node sdk', { semantic_limit: 5, keyword_limit: 5 });
      ok('searchHybrid()');
    } catch (e) {
      skipped('searchHybrid()', e.message);
    }

    console.log('\n== v3 compat ==');
    try {
      await client.v3.add({
        messages: [{ role: 'user', content: 'I like Go' }],
        user_id: USER_ID,
      });
      ok('v3.add()');
    } catch (e) {
      skipped('v3.add()', e.message);
    }

    try {
      await client.v3.search('Go', { user_id: USER_ID, limit: 5 });
      ok('v3.search()');
    } catch (e) {
      skipped('v3.search()', e.message);
    }

    console.log('\n== Profiles ==');
    try {
      await client.profiles.get(USER_ID);
      ok('profiles.get()');
    } catch (e) {
      skipped('profiles.get()', e.message);
    }

    if (memId) {
      console.log('\n== Cleanup ==');
      try {
        await client.deleteMemory(memId);
        ok('deleteMemory()');
      } catch (e) {
        skipped('deleteMemory()', e.message);
      }
    }
  } finally {
    // fetch client has no explicit close
  }

  console.log(`\nResults: ${pass} passed, ${fail} failed, ${skip} skipped`);
  process.exit(fail > 0 ? 1 : 0);
}

run();
