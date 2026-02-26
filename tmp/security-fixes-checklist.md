# PicoClaw Security Fixes - Implementation Checklist

**Date**: 2026-02-25
**Target Completion**: Week 1 (Priority 1-3)

---

## Quick Wins (60 minutes total)

### ✅ Fix #1: GitHub Actions Shell Injection (30 minutes)

**Priority**: 🔴 P1 (Highest)
**Severity**: Medium
**Effort**: 30 minutes
**Impact**: Prevents CI/CD compromise

#### Files to Modify:
1. `.github/workflows/release.yml`
2. `.github/workflows/docker-build.yml`

#### Implementation Steps:

**Step 1: Update release.yml (Lines 21-42)**

```yaml
jobs:
  # Add validation job
  validate:
    name: Validate Inputs
    runs-on: ubuntu-latest
    steps:
      - name: Validate tag format
        run: |
          TAG="${{ inputs.tag }}"
          if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
            echo "Error: Invalid tag format. Expected: v1.2.3 or v1.2.3-beta"
            exit 1
          fi
          echo "✅ Tag validation passed: $TAG"

  create-tag:
    name: Create Git Tag
    needs: validate  # Add dependency
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Checkout
        uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - name: Create and push tag
        shell: bash
        env:
          RELEASE_TAG: ${{ inputs.tag }}  # Use env var
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git tag -a "$RELEASE_TAG" -m "Release $RELEASE_TAG"
          git push origin "$RELEASE_TAG"
```

**Step 2: Update release.yml (Lines 95-102)**

```yaml
      - name: Apply release flags
        shell: bash
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          RELEASE_TAG: ${{ inputs.tag }}  # Use env var
          DRAFT_FLAG: ${{ inputs.draft }}
          PRERELEASE_FLAG: ${{ inputs.prerelease }}
        run: |
          gh release edit "$RELEASE_TAG" \
            --draft="$DRAFT_FLAG" \
            --prerelease="$PRERELEASE_FLAG"
```

**Step 3: Update docker-build.yml (Lines 53-62)**

```yaml
      - name: 🏷️ Prepare image tags
        id: tags
        shell: bash
        env:
          INPUT_TAG: ${{ inputs.tag }}  # Use env var
        run: |
          # Validate tag format
          if ! [[ "$INPUT_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
            echo "Error: Invalid tag format"
            exit 1
          fi

          tag="$INPUT_TAG"
          echo "ghcr_tag=${{ env.GHCR_REGISTRY }}/${{ env.GHCR_IMAGE_NAME }}:${tag}" >> "$GITHUB_OUTPUT"
          echo "ghcr_latest=${{ env.GHCR_REGISTRY }}/${{ env.GHCR_IMAGE_NAME }}:latest" >> "$GITHUB_OUTPUT"
          echo "dockerhub_tag=${{ env.DOCKERHUB_REGISTRY }}/${{ env.DOCKERHUB_IMAGE_NAME }}:${tag}" >> "$GITHUB_OUTPUT"
          echo "dockerhub_latest=${{ env.DOCKERHUB_REGISTRY }}/${{ env.DOCKERHUB_IMAGE_NAME }}:latest" >> "$GITHUB_OUTPUT"
```

#### Testing:
```bash
# Test with valid tag
gh workflow run release.yml -f tag=v1.0.0 -f prerelease=false

# Test with invalid tag (should fail validation)
gh workflow run release.yml -f tag='v1.0.0"; echo malicious'
```

#### Verification:
- [ ] Valid tags (v1.2.3) pass validation
- [ ] Invalid tags fail with error message
- [ ] Shell metacharacters (`;`, `&`, `|`) are rejected
- [ ] Workflow runs successfully end-to-end

---

### ✅ Fix #2: Security Headers on HTTP Responses (15 minutes)

**Priority**: 🟡 P2
**Severity**: Medium
**Effort**: 15 minutes
**Impact**: Prevents content-sniffing XSS

#### Files to Modify:
1. `pkg/channels/wecom.go` (Line 244)
2. `pkg/channels/wecom_app.go` (Line 320)

#### Implementation Steps:

**Step 1: Update wecom.go - handleVerification function**

```go
// pkg/channels/wecom.go - Lines 240-245
func (c *WeComBotChannel) handleVerification(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    msgSignature := query.Get("msg_signature")
    timestamp := query.Get("timestamp")
    nonce := query.Get("nonce")
    echostr := query.Get("echostr")

    if msgSignature == "" || timestamp == "" || nonce == "" || echostr == "" {
        http.Error(w, "Missing parameters", http.StatusBadRequest)
        return
    }

    // Verify signature
    if !WeComVerifySignature(c.config.Token, msgSignature, timestamp, nonce, echostr) {
        logger.WarnC("wecom", "Signature verification failed")
        http.Error(w, "Invalid signature", http.StatusForbidden)
        return
    }

    // Decrypt echostr
    decryptedEchoStr, err := WeComDecryptMessageWithVerify(echostr, c.config.EncodingAESKey, "")
    if err != nil {
        logger.ErrorCF("wecom", "Failed to decrypt echostr", map[string]any{
            "error": err.Error(),
        })
        http.Error(w, "Decryption failed", http.StatusInternalServerError)
        return
    }

    // Remove BOM and whitespace
    decryptedEchoStr = strings.TrimSpace(decryptedEchoStr)
    decryptedEchoStr = strings.TrimPrefix(decryptedEchoStr, "\xef\xbb\xbf")

    // ADD THESE LINES: Set security headers
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Write([]byte(decryptedEchoStr))
}
```

**Step 2: Update wecom_app.go - handleVerification function (same pattern)**

```go
// pkg/channels/wecom_app.go - Lines 316-321
func (c *WeComAppChannel) handleVerification(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    // ... existing verification logic ...

    decryptedEchoStr = strings.TrimSpace(decryptedEchoStr)
    decryptedEchoStr = strings.TrimPrefix(decryptedEchoStr, "\xef\xbb\xbf")

    // ADD THESE LINES: Set security headers
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Write([]byte(decryptedEchoStr))
}
```

**Optional: Add security headers to all HTTP handlers**

```go
// Add to handleWebhook function in both files
func (c *WeComBotChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
    // Add security headers at the top
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")

    ctx := r.Context()

    if r.Method == http.MethodGet {
        c.handleVerification(ctx, w, r)
        return
    }

    if r.Method == http.MethodPost {
        c.handleMessageCallback(ctx, w, r)
        return
    }

    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
```

#### Testing:
```bash
# Test with curl
curl -I http://localhost:8080/webhook/wecom

# Verify headers are present:
# Content-Type: text/plain; charset=utf-8
# X-Content-Type-Options: nosniff
```

#### Verification:
- [ ] Content-Type header is set to text/plain
- [ ] X-Content-Type-Options: nosniff is present
- [ ] Response still functions correctly with WeCom API
- [ ] Webhook verification passes

---

### ✅ Fix #3: Replace math/rand with crypto/rand (15 minutes)

**Priority**: 🟡 P3
**Severity**: Low
**Effort**: 15 minutes
**Impact**: Eliminates weak RNG usage

#### Files to Modify:
1. `pkg/providers/antigravity_provider.go` (Lines 10, 758-765)

#### Implementation Steps:

**Step 1: Update imports**

```go
// pkg/providers/antigravity_provider.go - Line 10
import (
    "bufio"
    "bytes"
    "context"
    "crypto/rand"  // CHANGE: Use crypto/rand instead of math/rand
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/sipeed/picoclaw/pkg/auth"
    "github.com/sipeed/picoclaw/pkg/logger"
)
```

**Step 2: Replace randomString function**

```go
// pkg/providers/antigravity_provider.go - Lines 758-770
// randomString generates a cryptographically secure random string.
// Returns a URL-safe base64-encoded string of length n.
// Falls back to timestamp-based ID if crypto/rand fails (extremely rare).
func randomString(n int) string {
    // Generate enough bytes for the desired output length
    // Base64 encoding increases size by ~33%, so we need fewer bytes
    numBytes := (n * 3) / 4
    if numBytes < n {
        numBytes = n // Ensure we have enough bytes
    }

    b := make([]byte, numBytes)
    if _, err := rand.Read(b); err != nil {
        // Fallback for systems without good entropy source (extremely rare)
        logger.WarnCF("provider.antigravity", "crypto/rand failed, using timestamp fallback", map[string]any{
            "error": err.Error(),
        })
        return fmt.Sprintf("%d", time.Now().UnixNano())
    }

    // Use URL-safe base64 encoding and truncate to desired length
    encoded := base64.URLEncoding.EncodeToString(b)
    if len(encoded) > n {
        return encoded[:n]
    }
    return encoded
}
```

**Alternative (Simpler Implementation)**:

```go
// pkg/providers/antigravity_provider.go - Lines 758-765
func randomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, n)

    // Use crypto/rand to fill bytes
    if _, err := rand.Read(b); err != nil {
        return fmt.Sprintf("%d", time.Now().UnixNano())
    }

    // Map bytes to letters
    for i := range b {
        b[i] = letters[int(b[i])%len(letters)]
    }

    return string(b)
}
```

#### Testing:

```go
// Add test to verify uniqueness
func TestRandomStringUniqueness(t *testing.T) {
    seen := make(map[string]bool)
    iterations := 10000

    for i := 0; i < iterations; i++ {
        s := randomString(9)
        if seen[s] {
            t.Fatalf("Duplicate random string detected after %d iterations: %s", i, s)
        }
        seen[s] = true

        // Verify length
        if len(s) != 9 {
            t.Errorf("Expected length 9, got %d: %s", len(s), s)
        }
    }

    t.Logf("✅ Generated %d unique random strings", iterations)
}
```

```bash
# Run test
go test -v ./pkg/providers -run TestRandomStringUniqueness
```

#### Verification:
- [ ] All uses of math/rand removed from antigravity_provider.go
- [ ] crypto/rand is imported and used
- [ ] randomString() generates unique values
- [ ] Antigravity API requests still work
- [ ] No collisions in request IDs

---

## Post-Implementation Checklist

### Code Changes
- [ ] All 3 fixes implemented
- [ ] Code compiles without errors: `go build ./...`
- [ ] All tests pass: `go test ./...`
- [ ] No new linter warnings: `golangci-lint run`

### Testing
- [ ] GitHub Actions workflow tested with valid and invalid inputs
- [ ] HTTP security headers verified with curl
- [ ] Random string uniqueness tested
- [ ] Integration tests pass for WeCom webhooks
- [ ] Antigravity provider still authenticates correctly

### Documentation
- [ ] Update CHANGELOG.md with security fixes
- [ ] Update README.md with security best practices
- [ ] Document accepted risks (SHA-1 usage)
- [ ] Add security section to documentation

### Git Workflow
```bash
# Create feature branch
git checkout -b security/quick-wins

# Stage changes
git add .github/workflows/release.yml
git add .github/workflows/docker-build.yml
git add pkg/channels/wecom.go
git add pkg/channels/wecom_app.go
git add pkg/providers/antigravity_provider.go

# Commit with descriptive message
git commit -m "security: fix shell injection, XSS, and weak RNG

- Add input validation to GitHub Actions workflows (CWE-78)
- Set Content-Type headers on HTTP responses (CWE-79)
- Replace math/rand with crypto/rand for request IDs (CWE-338)

Addresses 3 out of 9 security findings from 2026-02-25 review.
See: tmp/security-review-report.md"

# Push and create PR
git push origin security/quick-wins
gh pr create --title "Security: Quick wins (shell injection, XSS, weak RNG)" \
  --body "Implements 3 high-priority security fixes from security review.

## Changes
- GitHub Actions: Input validation for workflow_dispatch tags
- WeCom channels: Security headers on HTTP responses
- Antigravity provider: Cryptographically secure RNG

## Testing
- [x] All tests pass
- [x] GitHub Actions validation tested
- [x] HTTP headers verified
- [x] Random string uniqueness tested

Closes #XXX"
```

---

## Verification Commands

```bash
# Verify all changes compile
go build ./...

# Run tests
go test ./...

# Check for new linter issues
golangci-lint run

# Verify GitHub Actions syntax
gh workflow view release.yml
gh workflow view docker-build.yml

# Test WeCom webhook (if running locally)
curl -I http://localhost:8080/webhook/wecom

# Check for any remaining math/rand usage
grep -r "math/rand" pkg/ --exclude-dir=vendor
```

---

## Success Criteria

✅ All 3 fixes implemented and tested
✅ No compilation errors or test failures
✅ GitHub Actions workflows validate inputs
✅ HTTP responses include security headers
✅ crypto/rand used for all random number generation
✅ Code reviewed and merged to main branch

**Estimated Total Time**: 60-90 minutes (including testing)

---

## Next Steps (Week 2+)

After completing the quick wins, proceed with:

1. **Short-term improvements** (Month 1):
   - Add timestamp validation to WeCom signature verification
   - Implement config file permission checking
   - Enhance command deny patterns
   - Document SHA-1 usage as accepted risk

2. **Medium-term enhancements** (Months 2-3):
   - User approval prompts for sensitive commands
   - Container-based execution option
   - Integrate govulncheck into CI/CD
   - Enhanced audit logging

3. **Long-term initiatives** (Months 3-6):
   - Capability-based security model
   - Config file encryption at rest
   - OS keychain integration
   - Comprehensive security documentation

---

**Prepared By**: Claude Code
**Based On**: security-review-report.md (2026-02-25)
**Next Review**: After fixes implemented (Week 2)
