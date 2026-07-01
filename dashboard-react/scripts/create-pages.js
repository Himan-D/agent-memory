import fs from 'fs';
import path from 'path';

const pages = [
  'dashboard/index.tsx',
  'dashboard/agents.tsx',
  'dashboard/alerts.tsx',
  'dashboard/analytics.tsx',
  'dashboard/api-keys.tsx',
  'dashboard/audit.tsx',
  'dashboard/billing.tsx',
  'dashboard/chains.tsx',
  'dashboard/documents.tsx',
  'dashboard/entities.tsx',
  'dashboard/groups.tsx',
  'dashboard/memories.tsx',
  'dashboard/notifications.tsx',
  'dashboard/playground.tsx',
  'dashboard/projects.tsx',
  'dashboard/search.tsx',
  'dashboard/sessions.tsx',
  'dashboard/settings.tsx',
  'dashboard/skills.tsx',
  'dashboard/users.tsx',
  'dashboard/webhooks.tsx',
  'auth/signin.tsx',
  'auth/signup.tsx',
  'auth/error.tsx',
  'demo/index.tsx'
];

const basePath = path.join(process.cwd(), 'src', 'pages');

pages.forEach(page => {
  const pagePath = path.join(basePath, page);
  const dirPath = path.dirname(pagePath);
  
  if (!fs.existsSync(dirPath)) {
    fs.mkdirSync(dirPath, { recursive: true });
  }

  const componentName = path.basename(page, '.tsx') === 'index' 
    ? path.basename(dirPath).charAt(0).toUpperCase() + path.basename(dirPath).slice(1) + 'Page'
    : path.basename(page, '.tsx').split('-').map(p => p.charAt(0).toUpperCase() + p.slice(1)).join('') + 'Page';
    
  const content = `export default function ${componentName}() {
  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold">${componentName} Placeholder</h1>
      <p className="mt-2 text-muted-foreground">This page is pending migration in Phase 3B.</p>
    </div>
  );
}
`;

  fs.writeFileSync(pagePath, content);
  console.log(`Created ${pagePath}`);
});
