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
      NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
      ADMIN_API_KEY: process.env.ADMIN_API_KEY || '',
      NEXT_PUBLIC_AMPLITUDE_API_KEY: process.env.NEXT_PUBLIC_AMPLITUDE_API_KEY || '',
      NEXTAUTH_URL: process.env.NEXTAUTH_URL || '',
      NEXTAUTH_SECRET: process.env.NEXTAUTH_SECRET || '',
    }
  }]
};
