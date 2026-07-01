const fs = require('fs');
const path = require('path');

const dashboardPagesDir = 'C:\\Users\\hp\\Desktop\\Project\\agent-memory\\dashboard\\src\\app\\(dashboard)';

const pages = fs.readdirSync(dashboardPagesDir, { withFileTypes: true })
  .filter(dirent => dirent.isDirectory())
  .map(dirent => dirent.name);

const dependencies = {};

function analyzeDependencies(filePath, pageName) {
  if (!fs.existsSync(filePath)) return;
  const content = fs.readFileSync(filePath, 'utf-8');
  // Match both named and default imports
  const importRegex = /import\s+[^;]+from\s+['"]([^'"]+)['"]/g;
  let match;
  
  if (!dependencies[pageName]) {
    dependencies[pageName] = { components: new Set(), hooks: new Set(), lib: new Set(), others: new Set() };
  }
  
  while ((match = importRegex.exec(content)) !== null) {
    const importPath = match[1];
    if (importPath.startsWith('@/components/')) {
      dependencies[pageName].components.add(importPath);
    } else if (importPath.startsWith('@/hooks/')) {
      dependencies[pageName].hooks.add(importPath);
    } else if (importPath.startsWith('@/lib/')) {
      dependencies[pageName].lib.add(importPath);
    } else if (importPath.startsWith('@/')) {
      dependencies[pageName].others.add(importPath);
    } else if (importPath.startsWith('.')) {
      // Also check local directory dependencies if there are local components
      dependencies[pageName].others.add(importPath);
    }
  }
}

pages.forEach(page => {
  const pagePath = path.join(dashboardPagesDir, page, 'page.tsx');
  analyzeDependencies(pagePath, page);
  
  // also check for any components inside the page folder
  const pageDir = path.join(dashboardPagesDir, page);
  const files = fs.readdirSync(pageDir).filter(f => f.endsWith('.tsx') && f !== 'page.tsx');
  files.forEach(f => {
    analyzeDependencies(path.join(pageDir, f), page);
  });
});

const report = {};
for (const [page, deps] of Object.entries(dependencies)) {
  report[page] = {
    components: Array.from(deps.components),
    hooks: Array.from(deps.hooks),
    lib: Array.from(deps.lib),
    others: Array.from(deps.others)
  };
}

console.log(JSON.stringify(report, null, 2));
