module.exports = {
  apps: [{
    name: 'dashboard',
    cwd: '/home/ubuntu/agent-memory/dashboard',
    script: 'npx',
    args: 'next start',
    instances: 1,
    autorestart: true,
    max_memory_restart: '1G',
    env: {
      NODE_ENV: 'production',
      NEXT_PUBLIC_API_URL: 'http://localhost:8080',
      ADMIN_API_KEY: 'am_AYQh3k5V47AVVoyY_1776234755',
      NEXT_PUBLIC_AMPLITUDE_API_KEY: '5a684520b5dcd448c4fd3874a8a9b663',
      NEXTAUTH_URL: 'https://dashboard.hystersis.ai',
      NEXTAUTH_SECRET: '0g0XXNo7EKz2AYmTVIk/Ma0EqYptwkP8mjNterPENZs=',
    }
  }]
};
