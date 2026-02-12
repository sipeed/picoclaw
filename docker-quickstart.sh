#!/bin/bash

# PicoClaw Docker Quick Start Script
# This script helps you get started with PicoClaw in Docker quickly

set -e

echo "🦞 PicoClaw Docker Quick Start"
echo "================================"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed."
    echo "Please install Docker from https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version &> /dev/null && ! command -v docker-compose &> /dev/null; then
    echo "❌ Error: Docker Compose is not installed."
    echo "Please install Docker Compose from https://docs.docker.com/compose/install/"
    exit 1
fi

# Use docker compose or docker-compose based on availability
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo "✅ Docker is installed"
echo ""

# Check if config.json exists
if [ ! -f "config.json" ]; then
    echo "📝 Creating config.json from config.example.json..."
    if [ -f "config.example.json" ]; then
        cp config.example.json config.json
        echo "✅ config.json created"
        echo ""
        echo "⚠️  IMPORTANT: Please edit config.json and add your API keys:"
        echo "   - OpenRouter API key (https://openrouter.ai/keys)"
        echo "   - Zhipu API key (https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys)"
        echo "   - Or other LLM provider API keys"
        echo ""
        read -p "Press Enter when you've added your API keys to config.json..."
    else
        echo "❌ Error: config.example.json not found"
        exit 1
    fi
else
    echo "✅ config.json already exists"
    echo ""
fi

# Build and start services
echo "🔨 Building Docker image..."
$DOCKER_COMPOSE build

echo ""
echo "🚀 Starting PicoClaw..."
$DOCKER_COMPOSE up -d

echo ""
echo "✨ PicoClaw is now running!"
echo ""
echo "Quick commands:"
echo "  Interactive chat:  $DOCKER_COMPOSE exec picoclaw picoclaw agent"
echo "  Single message:    $DOCKER_COMPOSE exec picoclaw picoclaw agent -m \"Your message\""
echo "  View logs:         $DOCKER_COMPOSE logs -f picoclaw"
echo "  Stop service:      $DOCKER_COMPOSE down"
echo "  Restart service:   $DOCKER_COMPOSE restart"
echo ""
echo "💡 Tip: Use 'make docker-logs' to view logs"
echo "📖 Documentation: See README.md and docker/README.md for more details"
echo ""
