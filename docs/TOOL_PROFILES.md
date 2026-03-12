# Tool Profiles

Tool choice stays dynamic.

The agent may run any tool that is best for the job. Tool profiles only control
how output is reduced into context when a tool matches a known profile.

## Core Rule

- tool selection is dynamic
- context optimization is profile-based
- unknown tools still work through the normal path

## Current Profiles

- `subdomain-enum`
- `port-scan`
- `crawl`
- `web-probe`
- `vuln-scan`
- `fuzz`
- `code-analysis`

Implementation:
- `pkg/tools/profiles/profiles.go`

## What profiles do

- map multiple tools into one handling family
- register shared output filters
- identify when structured artifact parsing is available
- let the agent keep raw tool freedom while improving context reduction for common cases

## Structured pipelines

- CLAW pipelines and phase contracts can now require profiles, not just exact tool names
- predefined contracts still keep preferred concrete tools for backward compatibility
- completion can be satisfied by profile execution, while explicit preferred tools remain visible defaults
- structured phase dependencies can now be expressed at the profile level, not only tool-to-tool

## Runtime readiness

- startup can now detect which profiles are actually runnable on the current machine
- the agent path reports missing capability profiles as compact warnings
- the TUI status bar now shows capability readiness counts and can display a startup warning message
- workflow loading can now infer likely required profiles from workflow content and show more targeted warnings
- CLAW pipeline preflight still performs stricter phase-specific blocking checks

## Generic fallback

- tools without a dedicated profile still use a generic large-output filter
- full output is preserved on disk
- prompt context gets a compact summary instead of the full blob when output is large
- this is a fallback, not a replacement for profile-specific handling

## What profiles do not do

- they do not force the agent to use a specific binary
- they do not hardcode execution order
- they do not block unsupported tools
