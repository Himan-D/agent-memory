#!/usr/bin/env python3
"""Test the compression API directly"""

import requests
import json
import time

API_URL = "http://localhost:8080"

# Test 1: Check if server is running
print("=== Testing API ===")
resp = requests.get(f"{API_URL}/compression/stats")
print(f"1. Stats: {resp.status_code}")
print(f"   {resp.json()}")

# Test 2: Test compression
print("\n2. Testing compression...")
data = {
    "text": "machine learning is a subset of artificial intelligence that enables computers to learn from data",
    "modes": ["extraction"]
}
resp = requests.post(f"{API_URL}/playground/compress", json=data)
print(f"   Status: {resp.status_code}")
print(f"   Response: {json.dumps(resp.json(), indent=2)}")

# Test 3: Check what's running on the port
print("\n3. Server info...")
resp = requests.get(f"{API_URL}/")
print(f"   {resp.text[:100]}")