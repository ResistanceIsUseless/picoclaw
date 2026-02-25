#!/bin/bash
# Test script for tier routing functionality

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "======================================"
echo "Tier Routing Test Script"
echo "======================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if build exists
if [ ! -f "$PROJECT_ROOT/build/picoclaw" ]; then
    echo -e "${YELLOW}Build not found. Building...${NC}"
    cd "$PROJECT_ROOT"
    make build
fi

# Check environment variables
echo "Checking environment..."
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo -e "${RED}ERROR: ANTHROPIC_API_KEY not set${NC}"
    echo "Set it with: export ANTHROPIC_API_KEY='sk-ant-...'"
    exit 1
fi
echo -e "${GREEN}✓ ANTHROPIC_API_KEY set${NC}"

if [ -z "$LM_STUDIO_BASE_URL" ]; then
    echo -e "${YELLOW}WARNING: LM_STUDIO_BASE_URL not set${NC}"
    echo "Defaulting to: http://localhost:1234/v1"
    export LM_STUDIO_BASE_URL="http://localhost:1234/v1"
fi
echo -e "${GREEN}✓ LM_STUDIO_BASE_URL: $LM_STUDIO_BASE_URL${NC}"

# Check if LM Studio is running
echo ""
echo "Checking LM Studio..."
if curl -s -f "$LM_STUDIO_BASE_URL/models" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ LM Studio is running${NC}"
    MODEL_COUNT=$(curl -s "$LM_STUDIO_BASE_URL/models" | jq '.data | length' 2>/dev/null || echo "0")
    echo "  Models loaded: $MODEL_COUNT"
else
    echo -e "${YELLOW}WARNING: LM Studio not responding${NC}"
    echo "  Make sure LM Studio is running on port 1234"
    echo "  Or tier routing will fallback to API-only mode"
fi

# Check config
echo ""
echo "Checking config..."
CONFIG_PATH="$HOME/.picoclaw/config.json"
if [ ! -f "$CONFIG_PATH" ]; then
    echo -e "${YELLOW}Config not found. Creating from example...${NC}"
    mkdir -p "$HOME/.picoclaw"
    cp "$PROJECT_ROOT/config/config.tier-routing.example.json" "$CONFIG_PATH"
    echo "  Created: $CONFIG_PATH"
    echo "  Please edit it with your settings and run again"
    exit 1
fi

ROUTING_ENABLED=$(cat "$CONFIG_PATH" | jq -r '.routing.enabled' 2>/dev/null || echo "false")
if [ "$ROUTING_ENABLED" != "true" ]; then
    echo -e "${RED}ERROR: Tier routing not enabled in config${NC}"
    echo "  Edit $CONFIG_PATH and set routing.enabled to true"
    exit 1
fi
echo -e "${GREEN}✓ Tier routing enabled${NC}"

TIER_COUNT=$(cat "$CONFIG_PATH" | jq '.routing.tiers | length' 2>/dev/null || echo "0")
echo "  Tiers configured: $TIER_COUNT"

# Run tests
echo ""
echo "======================================"
echo "Running Tests"
echo "======================================"
echo ""

# Test 1: Simple planning task
echo "Test 1: Planning task (should use heavy tier)"
echo "----------------------------------------"
"$PROJECT_ROOT/build/picoclaw" agent -m "Create a plan to scan 192.168.1.0/24 for web services. Just outline the steps, don't execute yet." 2>&1 | grep -E "(Routing to tier|tier=)" || echo "No tier routing logs (check if enabled)"
echo ""

# Test 2: Parsing task
echo "Test 2: Parsing task (should use light tier)"
echo "----------------------------------------"
LONG_OUTPUT=$(printf 'Line %d\n' {1..100})
echo "$LONG_OUTPUT" | "$PROJECT_ROOT/build/picoclaw" agent -m "Parse this output and extract the unique numbers" 2>&1 | grep -E "(Routing to tier|tier=)" || echo "No tier routing logs"
echo ""

# Test 3: Cost report
echo "Test 3: Checking cost tracking"
echo "----------------------------------------"
# Note: Cost report would be shown at session end
# This is a simplified test
echo "Cost tracking is per-session. Full report shown after interactive session."
echo ""

# Summary
echo "======================================"
echo "Test Summary"
echo "======================================"
echo ""
echo "What to look for in the output:"
echo "  1. 'Routing to tier tier=heavy' for planning tasks"
echo "  2. 'Routing to tier tier=light' for parsing tasks"
echo "  3. Different model names based on tier"
echo ""
echo "If you don't see tier routing logs:"
echo "  - Check that routing.enabled=true in config"
echo "  - Check that logger level is INFO or DEBUG"
echo "  - Review TIER_ROUTING_GUIDE.md for troubleshooting"
echo ""
echo "Next steps:"
echo "  1. Start an interactive session:"
echo "     ./build/picoclaw agent"
echo "  2. Run multiple commands and watch tier switches"
echo "  3. Exit to see cost report"
echo ""
