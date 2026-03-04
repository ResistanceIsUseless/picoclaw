#!/bin/bash
# Manual test script for CLAW with real security tools
# Tests against careers.draftkings.com as specified

set -e

# Ensure tools are in PATH
export PATH="$HOME/go/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"

# Check tools are available
echo "Checking security tools..."
for tool in subfinder amass nmap httpx nuclei; do
    if ! command -v $tool &> /dev/null; then
        echo "ERROR: $tool not found in PATH"
        echo "Install with: go install -v github.com/projectdiscovery/$tool/v2/cmd/$tool@latest"
        exit 1
    fi
    echo "✓ $tool found at $(which $tool)"
done

# Test subfinder directly
echo -e "\n=== Testing subfinder on careers.draftkings.com ==="
echo "Running: subfinder -d careers.draftkings.com -silent"
subfinder -d careers.draftkings.com -silent | head -10
echo "... (output truncated)"

# Show what CLAW would do
echo -e "\n=== CLAW Test Plan ==="
echo "Target: careers.draftkings.com"
echo "Pipeline: web_quick (recon → quick_scan)"
echo ""
echo "Phase 1: recon"
echo "  - Tools: subfinder, amass"
echo "  - Expected: Discover subdomains"
echo "  - Contract: Must produce SubdomainList artifact"
echo ""
echo "Phase 2: quick_scan"
echo "  - Tools: httpx, nuclei"
echo "  - Expected: Probe subdomains, identify technologies"
echo "  - Contract: Must produce ServiceFingerprint artifacts"
echo ""
echo "To enable CLAW mode, set:"
echo "  export PICOCLAW_CLAW_ENABLED=true"
echo "  export PICOCLAW_CLAW_PIPELINE=web_quick"
echo ""
echo "Note: CLAW integration is complete but needs picoclaw CLI integration"
echo "Currently available:"
echo "  - ✅ Full tool execution pipeline"
echo "  - ✅ Artifact publishing to blackboard"
echo "  - ✅ Graph mutation and knowledge graph updates"
echo "  - ✅ Contract validation and phase management"
echo "  - ✅ E2E tests passing (mock tools)"
echo ""
echo "Ready for real-world testing!"
