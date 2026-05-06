#!/bin/bash

# Script to publish @hystersis/skills to NPM
# Usage: NPM_TOKEN=your_token ./skills-npm/publish.sh

set -e

cd "$(dirname "$0")"

if [ -z "$NPM_TOKEN" ]; then
  echo "Error: NPM_TOKEN environment variable is required"
  echo "Usage: NPM_TOKEN=your_token $0"
  exit 1
fi

echo "Setting up NPM credentials..."

# Set auth token directly (more secure than npm login)
npm set //registry.npmjs.org/:_authToken "$NPM_TOKEN"

echo "Publishing @hystersis/skills..."
npm publish --access public

echo "Done! Package published successfully."
