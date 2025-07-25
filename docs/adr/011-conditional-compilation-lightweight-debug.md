---
title: "ADR-011: Conditional Compilation for Lightweight Debug System"
status: "Superseded"
date: "2025-07-25"
tags: ["debug", "conditional-compilation", "sqlite", "build-optimization", "superseded"]
supersedes: ""
superseded-by: "ADR-012"
---

# ADR-011: Conditional Compilation for Lightweight Debug System

## Status
**Superseded by ADR-012** - See history note below

## History Note
**Date**: 2025-07-25  
**Reason for Supersession**: Critical analysis revealed that conditional compilation introduces significant developer friction, CI/CD complexity, and scaling limitations that outweigh the binary size benefits. The ephemeral data approach creates debugging challenges for intermittent bugs, and the single-instance optimization assumption limits future scalability.

**Key Issues Identified**:
- Developer workflow friction (must remember which build to use)
- CI/CD pipeline complexity (dual build/test modes)
- Loss of debug data on crashes/restarts hampers intermittent bug diagnosis
- Architecture doesn't scale beyond single-instance deployment
- Over-optimization for binary size at expense of developer experience

**Superseded by**: [ADR-012: Runtime Debug Configuration](./012-runtime-debug-configuration.md) which adopts a lightweight runtime approach balancing binary concerns with developer experience.

## Context

Following the implementation of [ADR-010: MCP Debug System Architecture](./010-mcp-debug-system-architecture.md), we identified a critical concern: **the debug infrastructure weight is becoming substantial relative to the core MCP server functionality.**

### Current Debug System Issues
1. **Binary Weight**: SQLite3 dependency adds ~2MB + CGo compilation overhead
2. **Code Volume**: Debug infrastructure (~1050 lines) exceeds core server (~600 lines)
3. **Runtime Overhead**: Debug code compiled in even when disabled
4. **Deployment Complexity**: CGo dependency reduces deployment flexibility
5. **Philosophy Conflict**: Heavyweight debug system contradicts "light as possible" core design

### System Requirements
- **Single-instance deployment**: Optimize for current single-node Fly.io deployment
- **Ephemeral debug data**: No persistence requirements - system can be redeployed instantly
- **Zero operational overhead**: No backup, storage, or infrastructure management
- **Developer-centric**: Debug data useful only during active debugging sessions

## Decision

We will implement **conditional compilation using Go build tags** with **ephemeral SQLite storage** for debug builds only.

### Architecture Overview

```
Production Build (default):
├── cmd/cowpilot/main.go (unchanged)
├── internal/debug/noop.go (+build !debug)
└── bin/cowpilot (lightweight, no SQLite, no CGo)

Debug Build (-tags debug):  
├── cmd/cowpilot/main.go (unchanged)
├── internal/debug/storage.go (+build debug)
├── internal/debug/interceptor.go (+build debug)
├── cmd/mcp-debug-proxy/ (always available)
└── bin/cowpilot-debug (ephemeral SQLite, no persistence)
```

### Build Strategy
```bash
# Production build (default) - zero debug overhead
go build -o bin/cowpilot cmd/cowpilot/main.go

# Debug build - ephemeral debug capabilities
go build -tags debug -o bin/cowpilot-debug cmd/cowpilot/main.go

# Debug proxy (always available)
go build -o bin/mcp-debug-proxy cmd/mcp-debug-proxy/main.go
```

### Storage Strategy
**Production**: No debug storage dependencies

**Debug**: Ephemeral SQLite with three simple options:
```go
// Option 1: In-memory (fastest, completely ephemeral)
storage, err := sql.Open("sqlite3", ":memory:")

// Option 2: Temporary file (survives process but not restarts)
storage, err := sql.Open("sqlite3", "/tmp/debug_conversations.db")

// Option 3: Local file (simplest, gets wiped on redeploy)
storage, err := sql.Open("sqlite3", "./debug.db")
```

## Alternatives Considered

1. **Current Approach (Rejected)**
   - Always compile debug code with runtime enable/disable
   - **Problems**: Binary bloat, CGo dependency, runtime overhead

2. **Separate Debug Module (Considered)**
   - Move debug system to separate Go module
   - **Rejected**: Increased complexity, dependency management overhead

3. **Interface-Based Stubs (Considered)**
   - Runtime polymorphism for debug features
   - **Rejected**: Still compiles all code, doesn't solve binary size

4. **LiteFS + Distributed Storage (Rejected)**
   - Global replication with automatic backups
   - **Rejected**: Overkill for single-instance deployment, unnecessary complexity

5. **Managed Database Services (Rejected)**
   - Use Postgres/Redis for debug storage
   - **Rejected**: Massive overkill for ephemeral debug data

6. **Fly Volumes + Backup Management (Rejected)**
   - Persistent storage with backup strategies
   - **Rejected**: Unnecessary persistence - system can be redeployed instantly

## Rationale

### Technical Benefits
- **Zero Production Overhead**: Debug code completely absent from production builds
- **Minimal Debug Infrastructure**: Simple SQLite when debugging, nothing when not
- **Instant Recovery**: No backup needed - redeploy from development machine if issues occur
- **Development-Focused**: Debug data useful only during active debugging sessions

### Operational Benefits
- **Maximum Simplicity**: No infrastructure, backup, or storage management
- **Zero Cost**: No storage, volume, or managed service costs
- **Lightweight Philosophy**: Production builds remain absolutely minimal
- **Instant Deployment**: Fire-and-forget debug builds

### Strategic Alignment
- **Single-Instance Optimized**: Perfect fit for current deployment model
- **Developer-Centric**: Optimized for development and debugging workflows
- **Redeployable Architecture**: Embraces stateless, easily reproducible deployments

## Consequences

### Positive
- **Production Performance**: Zero debug overhead in production builds
- **Maximum Simplicity**: No storage, backup, or infrastructure concerns
- **Zero Cost**: No additional storage or service costs
- **Instant Recovery**: Redeploy from development machine if needed
- **Deployment Flexibility**: Same codebase, different build targets
- **Developer Focus**: Debug data available exactly when and where needed

### Negative
- **Ephemeral Data**: Debug data lost on restart/redeploy
- **Build Complexity**: Requires two build modes (production/debug)
- **Testing Overhead**: Must test both build variants
- **CI/CD Complexity**: Build pipeline needs conditional logic

### Mitigation Strategies
- **Makefile Targets**: Simplify build complexity with clear targets
- **CI/CD Automation**: Automated testing of both build variants
- **Documentation**: Clear guidelines for debug vs production builds
- **Embrace Ephemeral**: Design debug workflows around temporary data

## Implementation Plan

### Phase 1: Conditional Compilation (Week 1)
1. Add build tags to existing debug code:
   ```go
   //go:build debug
   // +build debug
   ```
2. Create no-op implementations for production builds
3. Simplify storage to ephemeral SQLite
4. Update Makefile with build targets
5. Test both build variants

### Phase 2: Documentation & Deployment (Week 2)
1. Update all documentation
2. Create deployment guides
3. CI/CD pipeline updates
4. Team training materials

## Technical Specifications

### Build Tags Implementation
```go
// internal/debug/storage.go
//go:build debug
// +build debug

package debug
// Simple SQLite implementation - ephemeral storage

// internal/debug/noop.go  
//go:build !debug
// +build !debug

package debug
// No-op stubs for production
```

### SQLite Configuration
```go
// Debug builds only - choose based on needs:

// Option 1: In-memory (fastest, completely ephemeral)
storage, err := sql.Open("sqlite3", ":memory:")

// Option 2: Temporary file (survives process restarts)
storage, err := sql.Open("sqlite3", "/tmp/debug_conversations.db")

// Option 3: Local file (simplest, gets wiped on redeploy)
storage, err := sql.Open("sqlite3", "./debug.db")
```

### Makefile Targets
```makefile
# Production build (default)
build:
	go build -o bin/cowpilot cmd/cowpilot/main.go

# Debug build
build-debug:
	go build -tags debug -o bin/cowpilot-debug cmd/cowpilot/main.go

# Both builds
build-all: build build-debug
```

## Cost Analysis

### Production Deployment
- **Compute**: ~$5-10/month (standard Fly.io app)
- **Storage**: $0 (no debug dependencies)
- **Total**: ~$5-10/month

### Debug Deployment  
- **Compute**: ~$5-10/month (same as production)
- **Debug Storage**: $0 (ephemeral, no persistent storage)
- **Total**: ~$5-10/month (zero additional cost)

## Success Metrics

### Performance Targets
- **Production Binary Size**: No increase from debug dependencies
- **Debug Binary Size**: Minimal increase (SQLite only when needed)
- **Debug Latency**: <5ms overhead per request
- **Memory Usage**: Minimal SQLite overhead in debug builds only

### Operational Targets
- **Build Time**: <2x increase for debug builds
- **Deployment Time**: No change from current process
- **Storage Costs**: $0 (no persistent storage)
- **Recovery Time**: <60 seconds to redeploy if needed

## References
- [ADR-010: MCP Debug System Architecture](./010-mcp-debug-system-architecture.md)
- [Go Build Constraints](https://pkg.go.dev/go/build#hdr-Build_Constraints)
- [SQLite Documentation](https://sqlite.org/docs.html)
- [Fly.io Deployment Guide](https://fly.io/docs/)

## Notes
- This ADR builds directly on ADR-010's architecture while addressing binary weight concerns with maximum simplicity
- Ephemeral storage aligns with stateless, redeployable architecture philosophy
- Implementation maintains the non-invasive proxy architecture from Phase 1
- Future phases of the debug system (protocol validation, dashboard) will work with ephemeral data
- Conditional compilation pattern can be applied to other heavyweight features in the future
- "Redeploy if broken" approach eliminates all backup and persistence complexity
