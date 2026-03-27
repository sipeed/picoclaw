#!/bin/bash
# Test script for Affine integration

set -e

echo "🧪 Testing PicoClaw Affine Integration"
echo "======================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if picoclaw binary exists
if [ ! -f "./picoclaw" ]; then
    echo -e "${RED}❌ picoclaw binary not found${NC}"
    echo "Building PicoClaw..."
    make build
fi

# Check if config exists
if [ ! -f "$HOME/.picoclaw/config.json" ]; then
    echo -e "${YELLOW}⚠️  Config not found${NC}"
    echo "Creating config from example..."
    mkdir -p ~/.picoclaw
    cp config/config.example.json ~/.picoclaw/config.json
    echo -e "${YELLOW}⚠️  Please edit ~/.picoclaw/config.json with your Affine API credentials${NC}"
    exit 1
fi

# Check if Affine is enabled
if ! grep -q '"enabled": true' ~/.picoclaw/config.json | grep -A 5 '"affine"'; then
    echo -e "${YELLOW}⚠️  Affine integration is not enabled in config${NC}"
    echo "Please set tools.affine.enabled to true in ~/.picoclaw/config.json"
    exit 1
fi

echo "Running tests..."
echo ""

# Test 1: Unit tests
echo -e "${YELLOW}Test 1: Running unit tests...${NC}"
if go test ./pkg/tools -v -run TestAffineTool; then
    echo -e "${GREEN}✅ Unit tests passed${NC}"
else
    echo -e "${RED}❌ Unit tests failed${NC}"
fi
echo ""

# Test 2: List workspaces
echo -e "${YELLOW}Test 2: List workspaces${NC}"
if ./picoclaw agent -m "List my Affine workspaces"; then
    echo -e "${GREEN}✅ List workspaces successful${NC}"
else
    echo -e "${RED}❌ List workspaces failed${NC}"
    echo "Check your API credentials in ~/.picoclaw/config.json"
fi
echo ""

# Test 3: List pages
echo -e "${YELLOW}Test 3: List pages${NC}"
if ./picoclaw agent -m "List pages in my Affine workspace"; then
    echo -e "${GREEN}✅ List pages successful${NC}"
else
    echo -e "${RED}❌ List pages failed${NC}"
fi
echo ""

# Test 4: Get structure
echo -e "${YELLOW}Test 4: Get workspace structure${NC}"
if ./picoclaw agent -m "Show me the structure of my Affine workspace"; then
    echo -e "${GREEN}✅ Get structure successful${NC}"
else
    echo -e "${RED}❌ Get structure failed${NC}"
fi
echo ""

# Test 5: Create test note
echo -e "${YELLOW}Test 5: Create test note${NC}"
TEST_TITLE="Codespace Test $(date +%Y-%m-%d-%H-%M-%S)"
if ./picoclaw agent -m "Create a note in Affine titled '$TEST_TITLE' with content 'This is a test from PicoClaw Codespace' and tags 'test' and 'codespace'"; then
    echo -e "${GREEN}✅ Create note successful${NC}"
else
    echo -e "${RED}❌ Create note failed${NC}"
fi
echo ""

# Test 6: Search
echo -e "${YELLOW}Test 6: Search for test notes${NC}"
if ./picoclaw agent -m "Search my Affine workspace for 'Codespace Test'"; then
    echo -e "${GREEN}✅ Search successful${NC}"
else
    echo -e "${RED}❌ Search failed${NC}"
fi
echo ""

echo "======================================"
echo -e "${GREEN}🎉 Testing complete!${NC}"
echo ""
echo "Next steps:"
echo "  - Check the created test note in your Affine workspace"
echo "  - Try more commands: ./picoclaw agent -m 'Your message'"
echo "  - Read docs: docs/AFFINE_INTEGRATION.md"
