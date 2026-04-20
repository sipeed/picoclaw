#!/bin/bash

# Quick curl examples for webhook processing

# Set your gateway URL and token
GATEWAY_URL="${GATEWAY_URL:-http://localhost:18800}"
TOKEN="${TOKEN:-your-token-here}"

echo "Webhook Processing - Quick Examples"
echo "===================================="
echo ""
echo "Gateway URL: $GATEWAY_URL"
echo ""

# Example 1: Basic processing
echo "1. Submit basic processing job:"
echo "--------------------------------"
echo 'curl -X POST '$GATEWAY_URL'/api/webhook/process \'
echo '  -H "Content-Type: application/json" \'
echo '  -d '"'"'{'
echo '    "webhook_url": "https://webhook.site/your-unique-id",'
echo '    "payload": {'
echo '      "data": "Hello, World!",'
echo '      "priority": "high"'
echo '    }'
echo '  }'"'"
echo ""

# Example 2: AI Agent processing
echo "2. AI Agent processing:"
echo "--------------------------------"
echo 'curl -X POST '$GATEWAY_URL'/api/webhook/process \'
echo '  -H "Content-Type: application/json" \'
echo '  -d '"'"'{'
echo '    "webhook_url": "https://your-app.com/callback",'
echo '    "payload": {'
echo '      "prompt": "Analyze this data",'
echo '      "channel": "api",'
echo '      "chat_id": "user-123"'
echo '    }'
echo '  }'"'"
echo ""

# Example 3: Check job status
echo "3. Check job status:"
echo "--------------------------------"
echo 'curl '$GATEWAY_URL'/api/webhook/status?job_id=<JOB_ID>'
echo ""

# Example 4: With jq for pretty output
echo "4. Submit and parse with jq:"
echo "--------------------------------"
echo 'JOB_RESPONSE=$(curl -s -X POST '$GATEWAY_URL'/api/webhook/process \'
echo '  -H "Content-Type: application/json" \'
echo '  -d '"'"'{'
echo '    "webhook_url": "https://webhook.site/test",'
echo '    "payload": {"data": "test"}'
echo '  }'"'"')'
echo ''
echo 'echo $JOB_RESPONSE | jq .'
echo 'JOB_ID=$(echo $JOB_RESPONSE | jq -r .job_id)'
echo 'echo "Job ID: $JOB_ID"'
echo ""

# Example 5: Complex payload
echo "5. Complex payload with nested data:"
echo "--------------------------------"
echo 'curl -X POST '$GATEWAY_URL'/api/webhook/process \'
echo '  -H "Content-Type: application/json" \'
echo '  -d '"'"'{'
echo '    "webhook_url": "https://your-app.com/webhook",'
echo '    "payload": {'
echo '      "task": "process_document",'
echo '      "document": {'
echo '        "url": "https://example.com/doc.pdf",'
echo '        "pages": [1, 2, 3]'
echo '      },'
echo '      "options": {'
echo '        "extract_tables": true,'
echo '        "ocr": true'
echo '      },'
echo '      "metadata": {'
echo '        "user_id": "123",'
echo '        "timestamp": "2026-04-17T10:00:00Z"'
echo '      }'
echo '    }'
echo '  }'"'"
echo ""

echo "===================================="
echo ""
echo "To test with a real webhook receiver:"
echo "1. Visit https://webhook.site"
echo "2. Copy your unique URL"
echo "3. Replace the webhook_url in the examples above"
echo "4. Run the curl command"
echo "5. Watch the callback arrive at webhook.site"
