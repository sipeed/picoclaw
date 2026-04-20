#!/bin/bash

# Webhook Processing Test Script

set -e

HOST="http://localhost:18800"

echo "==================================="
echo "Webhook Processing Test"
echo "==================================="
echo ""

# Test 1: Submit a processing job
echo "Test 1: Submit processing job"
echo "-----------------------------------"

RESPONSE=$(curl -s -X POST "${HOST}/api/webhook/process" \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/unique-id",
    "payload": {
      "data": "Hello, World!",
      "priority": "high",
      "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }
  }')

echo "Response:"
echo "$RESPONSE" | jq .

JOB_ID=$(echo "$RESPONSE" | jq -r '.job_id')
echo ""
echo "Job ID: $JOB_ID"
echo ""

# Test 2: Check job status immediately
echo "Test 2: Check job status (immediately)"
echo "-----------------------------------"

curl -s "${HOST}/api/webhook/status?job_id=${JOB_ID}" | jq .
echo ""

# Test 3: Wait and check again
echo "Test 3: Wait 3 seconds and check again"
echo "-----------------------------------"
sleep 3

curl -s "${HOST}/api/webhook/status?job_id=${JOB_ID}" | jq .
echo ""

# Test 4: Submit job without webhook_url (should fail)
echo "Test 4: Submit invalid job (missing webhook_url)"
echo "-----------------------------------"

curl -s -X POST "${HOST}/api/webhook/process" \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {
      "data": "This should fail"
    }
  }' | jq .
echo ""

# Test 5: Check non-existent job
echo "Test 5: Check non-existent job"
echo "-----------------------------------"

curl -s "${HOST}/webhook/status?job_id=non-existent-id" | jq .
echo ""

# Test 6: Submit without auth token (should fail)
echo "Test 6: Submit without auth token (should fail)"
echo "-----------------------------------"

curl -s -X POST "${HOST}/webhook/process" \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_url": "https://webhook.site/test",
    "payload": {
      "data": "unauthorized"
    }
  }' | jq .
echo ""

echo "==================================="
echo "All tests completed!"
echo "==================================="
echo ""
echo "To test with a real webhook receiver:"
echo "1. Go to https://webhook.site"
echo "2. Copy your unique URL"
echo "3. Replace 'https://webhook.site/unique-id' in the script"
echo "4. Run this script again"
echo "5. Check webhook.site to see the callback"
