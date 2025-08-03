# Cowpilot Documentation

## Critical Files (Check First)

- `STATE.yaml` - Machine-optimized project state (CHECK FIRST EVERY SESSION)
- `AUTH_FLOWS.yaml` - Authentication patterns and debug findings
- `TODO.md` - Active development tasks
- `LONGRUNNING_TASKS.md` - Long-running task implementation

## Documentation Structure

### `/adr/` - Architecture Decision Records
Immutable records of significant architectural decisions.

### `/guides/` - How-To Guides
User-facing documentation with step-by-step instructions:
- User guide
- Testing guide  
- LLM integration guide
- Claude deployment & troubleshooting
- OAuth testing

### `/reference/` - Technical Reference
Technical specifications and protocols:
- Protocol standards and schemas
- OAuth requirements and compliance
- RTM authentication flow
- MCP protocol schema (JSON/TypeScript)

### `/mcp/` - MCP Protocol Documentation
- Conformance plans
- Implementation status

### `/assets/` - Images & Media

## Guides vs Reference

- **Guides**: How to USE the system (tutorials, deployment, troubleshooting)
- **Reference**: Technical SPECS (protocols, schemas, API definitions)

## Key Principles

1. **No Quick Fixes** - Technical debt compounds
2. **RTFM** - Always verify against actual code/docs
3. **Machine First** - STATE.yaml and AUTH_FLOWS.yaml optimized for Claude
4. **Quality Focus** - Complete solutions over patches

## Current Focus

RTM OAuth flow - Ensuring users complete authentication before callback.
