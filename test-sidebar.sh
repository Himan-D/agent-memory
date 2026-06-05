#!/bin/bash

# Script to test all dashboard sidebar endpoints for performance

BASE_URL="https://app.hystersis.com"
API_BASE="$BASE_URL/api/proxy"

echo "Testing Dashboard Sidebar Endpoints..."
echo "===================================="

# Test 1: Dashboard/Home
echo "1. Testing Dashboard Home..."
time curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/"
echo ""

# Test 2: Memories
echo "2. Testing Memories..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/memories" -H "Content-Type: application/json"
echo ""

# Test 3: Entities  
echo "3. Testing Entities..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/entities" -H "Content-Type: application/json"
echo ""

# Test 4: Sessions
echo "4. Testing Sessions..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/sessions" -H "Content-Type: application/json"
echo ""

# Test 5: Agents
echo "5. Testing Agents..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/agents" -H "Content-Type: application/json"
echo ""

# Test 6: Groups
echo "6. Testing Groups..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/groups" -H "Content-Type: application/json"
echo ""

# Test 7: Projects
echo "7. Testing Projects..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/projects" -H "Content-Type: application/json"
echo ""

# Test 8: Skills
echo "8. Testing Skills..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/skills" -H "Content-Type: application/json"
echo ""

# Test 9: Chains
echo "9. Testing Chains..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/chains" -H "Content-Type: application/json"
echo ""

# Test 10: Webhooks
echo "10. Testing Webhooks..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/webhooks" -H "Content-Type: application/json"
echo ""

# Test 11: API Keys
echo "11. Testing API Keys..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/api-keys" -H "Content-Type: application/json"
echo ""

# Test 12: Alerts
echo "12. Testing Alerts..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/alerts" -H "Content-Type: application/json"
echo ""

# Test 13: Users
echo "13. Testing Users..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/users" -H "Content-Type: application/json"
echo ""

# Test 14: Analytics
echo "14. Testing Analytics..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/analytics" -H "Content-Type: application/json"
echo ""

# Test 15: Notifications
echo "15. Testing Notifications..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/notifications" -H "Content-Type: application/json"
echo ""

# Test 16: Settings
echo "16. Testing Settings..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/settings" -H "Content-Type: application/json"
echo ""

# Test 17: Playground
echo "17. Testing Playground..."
time curl -s -o /dev/null -w "%{http_code}" "$API_BASE?endpoint=/playground" -H "Content-Type: application/json"
echo ""

# Test 18: Demo Page (Public)
echo "18. Testing Public Demo Page..."
time curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/demo"
echo ""

echo "===================================="
echo "Testing Complete!"
echo "Note: 401 errors indicate authentication required"
echo "Non-200 responses may indicate the endpoint needs authentication"
echo "All dashboard features should work after user authentication"