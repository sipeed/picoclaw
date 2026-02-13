#!/bin/bash

curl http://localhost:11434/api/chat -d '{
  "model": "glm-5:cloud",
  "messages": [{ "role": "user", "content": "Hello!" }]
}'

curl -X POST http://localhost:11434/v1/chat/completions \
-H "Content-Type: application/json" \
-d '{
  "model": "glm-5:cloud",
  "messages": [{ "role": "user", "content": "Say this is a test" }]
}'

curl http://localhost:11434/v1/completions -d '{
  "model": "glm-5:cloud",
  "messages": [{ "role": "user", "content": "Hello!" }]
}'
