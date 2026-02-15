#!/bin/bash
# Test script for PicoClaw Doctor in Docker

set -e

echo "🏥 Testing PicoClaw Doctor in Docker"
echo "======================================"
echo ""

# Create .env file template when missing
if [ ! -f .env ]; then
    echo "Creating .env template..."
    cat > .env << EOF
# Set your Gemini API key before running
GEMINI_API_KEY=
EOF
    echo "Please set GEMINI_API_KEY in .env and rerun."
    exit 1
fi

if ! grep -q '^GEMINI_API_KEY=.' .env; then
    echo "GEMINI_API_KEY is empty in .env. Please set it and rerun."
    exit 1
fi

echo "🐳 Building Docker image..."
docker compose build picoclaw-doctor

echo ""
echo "🔍 Running PicoClaw Doctor..."
docker compose run --rm picoclaw-doctor

echo ""
echo "✅ Test complete!"
