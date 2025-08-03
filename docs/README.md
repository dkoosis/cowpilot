# Cowpilot Documentation

## Session Protocol (CRITICAL)

**Start Every Session:**
1. **FIRST: Check `/docs/STATE.yaml`** for machine-optimized context
2. **SECOND: Check `/docs/AUTH_FLOWS.yaml`** for authentication patterns
3. Always maintain STATE.yaml optimized for Claude performance

## Directory Structure

### Critical Files (Root Level)
- `STATE.yaml` - Machine-optimized project state (CHECK FIRST)
- `AUTH_FLOWS.yaml` - Authentication flow patterns and debug findings
- `LONGRUNNING_TASKS.md` - Long-running task implementation documentation

### Core Documentation

#### `/adr/` - Architecture Decision Records
Immutable records of significant architectural decisions.

#### `/oauth/` - OAuth & Authentication
All OAuth-related documentation consolidated:
- Implementation plans and logs
- Protocol requirements
- RTM authentication flow
- Testing guides
- Deployment documentation

#### `/mcp/` - MCP Protocol
MCP-specific documentation:
- Conformance plans
- Implementation status
- Protocol debugging

#### `/reference/` - Technical Reference
- `schema.json` - MCP protocol schema
- `schema.ts` - TypeScript definitions
- `protocol-standards.md` - Protocol standards

#### `/guides/` - User & Developer Guides
- User guide
- Testing guide
- LLM integration guide
- Claude deployment & troubleshooting

#### `/backlog/` - Project Planning
- `history.md` - Project history
- `todo.md` - Current TODO items
- `rtm-enhancements-backlog.yaml` - RTM feature backlog

#### `/contributing/` - Contribution Guidelines
- Git workflow
- Commit message format
- Contributing guide

#### `/archive/` - Historical Documents
Dated reviews, old documentation, and completed work.

#### `/assets/` - Images & Media
Logos, diagrams, and visual assets.

## Key Principles

1. **No Quick Fixes** - Quick fixes == technical debt
2. **RTFM** - Always check actual code or documentation
3. **Machine First** - STATE.yaml and AUTH_FLOWS.yaml are optimized for Claude, not humans
4. **Transparency** - Voice concerns about risks or better alternatives
5. **Quality Focus** - Provide unfiltered technical counsel

## Current Focus

RTM OAuth flow - Ensuring users complete authentication before callback to prevent race conditions.
