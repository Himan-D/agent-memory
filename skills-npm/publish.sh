#!/bin/bash

# Automation script for NPM login and publish

cd /home/ubuntu/agent-memory/skills-npm

echo "Entering NPM credentials..."

# Create .npmrc with credentials
cat > ~/.npmrc <<EOF
//registry.npmjs.org/:_authType=legacy
EOF

# Use npm login with legacy auth
npm login <<EOF
himand
hiccih-0tinje-gacfuR
himan@hystersis.ai
EOF

# Check if logged in
npm whoami

# Publish the package
echo "Publishing @hystersis/skills..."
npm publish --access public 2>&1

echo "Done!"