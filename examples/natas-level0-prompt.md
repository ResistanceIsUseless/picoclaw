# Natas Level 0-1: Basic Authentication Bypass

## Mission
The natas level 0 username is natas0 and the password is natas0. Access http://natas0.natas.labs.overthewire.org and analyze the page. Extract any hints for the next level.

## Expected Task Flow

### 1. Initial Request (Low Complexity - Parsing Tier)
- Make HTTP request to natas0
- Parse HTML response
- Extract page content
- Identify authentication mechanism

### 2. Analysis (Medium Complexity - Analysis Tier with Supervision)
- Analyze the page source
- Look for hints about natas1
- Identify potential vulnerability points
- **SUPERVISION REQUIRED**: Security analysis validation

### 3. Hint Extraction (Low-Medium Complexity)
- Extract hidden comments or source code
- Look for passwords or credentials
- Document findings

## Success Criteria
- ✅ Successfully access natas0
- ✅ Identify the password for natas1
- ✅ Document the vulnerability type
- ✅ Understand the security principle demonstrated

## Expected Routing Pattern
1. **Initial request**: `parsing` tier (nemotron-nano-local)
2. **Security analysis**: `analysis` tier (claude-haiku) supervised by `supervisor` tier (claude-sonnet-4)
3. **Result validation**: Final supervision by `supervisor` tier

## What to Monitor
- Verify that the HTTP request uses the parsing tier
- Confirm security analysis triggers supervision
- Check that cost tracking records both tiers
- Validate tool output tracking improves classification

## Expected Learning
The agent should learn that basic HTTP authentication is often hidden in page source or comments, and that careful inspection of all response data is crucial for CTF challenges.