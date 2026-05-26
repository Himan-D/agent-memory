#!/bin/bash
# Publish hystersis Python SDK to PyPI
# Usage: PYPI_API_TOKEN=your_token ./sdk/python/publish.sh

set -e

cd "$(dirname "$0")"

if [ -z "$PYPI_API_TOKEN" ]; then
  echo "Error: PYPI_API_TOKEN environment variable is required"
  echo "Usage: PYPI_API_TOKEN=your_token $0"
  exit 1
fi

echo "Building package..."
pip install build twine --quiet
python -m build

echo "Publishing to PyPI..."
twine upload --username __token__ --password "$PYPI_API_TOKEN" dist/*

echo "Done! hystersis published to PyPI."
