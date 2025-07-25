---
title: "ADR-012: Runtime Debug Configuration"
status: "Accepted"
date: "2025-07-25"
tags: ["debug", "runtime-configuration", "sqlite", "developer-experience"]
supersedes: "ADR-011"
superseded-by: ""
---

# ADR-012: Runtime Debug Configuration

## Status
Accepted

## Context

[ADR-011: Conditional Compilation](./011-conditional-compilation-lightweight-debug.md) addressed binary weight concerns but introduced significant trade-offs:

- **Developer friction**: Must remember which build mode to use
- **CI/CD complexity**: Dual build/test pipeline requirements  
- **Ephemeral data loss**: Debug data lost on crashes, hampering intermittent bug diagnosis
- **Scaling limitations**: Single-instance assumption breaks with horizontal scaling
- **Over-optimization**: Prioritized binary size over developer experience

### Current Debug System Status
- Phase 1 implemented: conversation logging, SQLite storage, debug proxy
- ~2MB binary increase from SQLite3 dependency + CGo
- ~1050 lines debug code vs ~600 lines core server

### Requirements
- Lightweight production runtime (not build-time optimization)
- Persistent debug data when needed for intermittent bugs
- Developer-friendly workflow with minimal friction
- Horizontal scaling compatibility
- Balance binary concerns with practical debugging needs

## Decision

Implement **runtime debug configuration** with lightweight defaults and configurable storage options.

### Architecture

```
Single Build Target:
├── cmd/cowpilot/main.go (debug code always compiled)
├── internal/debug/storage.go (runtime-configured)
├── internal/debug/interceptor.go (runtime-enabled)
└── bin/cowpilot (includes debug, runtime-disabled by default)
```

### Configuration Strategy
```go
type DebugConfig struct {
    Enabled     bool   // Default: false
    StorageType string // "disabled", "memory", "file"
    MaxMemoryMB int    // Memory storage limit
    MaxFileMB   int    // File storage limit  
    RetentionH  int    // Auto-cleanup hours
}
```

### Storage Options
```bash
# Production (default)
# No environment variables = debug disabled, zero runtime cost

# Memory debug (development)
MCP_DEBUG=true MCP_DEBUG_STORAGE=memory

# Persistent debug (troubleshooting)  
MCP_DEBUG=true MCP_DEBUG_STORAGE=file MCP_DEBUG_PATH=./debug.db

# Bounded debug (production troubleshooting)
MCP_DEBUG=true MCP_DEBUG_STORAGE=memory MCP_DEBUG_MAX_MB=50
```

## Rationale

### Technical Benefits
- **Single build pipeline**: No CI/CD complexity
- **Runtime flexibility**: Configure per deployment needs
- **Bounded resource usage**: Memory/disk limits prevent runaway growth
- **Persistent when needed**: Solves intermittent bug debugging
- **Horizontal scaling**: Works with multiple instances

### Developer Benefits  
- **Zero friction**: Simple environment variable toggle
- **Always available**: No need to rebuild for debugging
- **Familiar workflow**: Standard runtime configuration pattern
- **Debugging continuity**: Data persists across application restarts

### Production Benefits
- **Zero cost by default**: Disabled = no runtime overhead
- **Flexible troubleshooting**: Can enable temporarily without redeploy
- **Bounded impact**: Resource limits prevent production issues
- **Quick recovery**: Disable via environment variable

## Implementation

### Phase 1: Runtime Configuration
```go
// Load debug config from environment
func LoadDebugConfig() *DebugConfig {
    enabled := os.Getenv("MCP_DEBUG") == "true"
    if !enabled {
        return &DebugConfig{Enabled: false} // Zero overhead
    }
    
    return &DebugConfig{
        Enabled:     true,
        StorageType: getEnvDefault("MCP_DEBUG_STORAGE", "memory"),
        MaxMemoryMB: getEnvInt("MCP_DEBUG_MAX_MB", 100),
        RetentionH:  getEnvInt("MCP_DEBUG_RETENTION_H", 24),
    }
}
```

### Phase 2: Bounded Storage
```go
// Memory storage with LRU eviction
type BoundedMemoryStorage struct {
    maxBytes int64
    cache    *lru.Cache
}

// File storage with rotation
type BoundedFileStorage struct {
    maxBytes int64
    path     string
    db       *sql.DB
}
```

### Environment Variables
```bash
MCP_DEBUG=true|false                    # Enable/disable debug system
MCP_DEBUG_STORAGE=disabled|memory|file  # Storage backend
MCP_DEBUG_PATH=./debug.db              # File storage path
MCP_DEBUG_MAX_MB=100                   # Storage size limit
MCP_DEBUG_RETENTION_H=24               # Auto-cleanup hours
MCP_DEBUG_LEVEL=DEBUG|INFO|WARN        # Log verbosity
```

## Binary Size Mitigation

### Acceptable Trade-offs
- **SQLite dependency**: Standard for Go applications, mature ecosystem
- **~2MB increase**: Reasonable for debugging capabilities
- **CGo dependency**: Acceptable for non-edge deployment scenarios

### Future Optimizations (if needed)
- **Optional features via build tags**: Advanced analytics, LiteFS integration
- **Pure Go SQLite**: Consider modernc.org/sqlite for CGo-free option
- **Pluggable storage**: Interface-based approach for custom backends

## Consequences

### Positive
- **Developer experience**: Simple, familiar configuration pattern
- **Debugging effectiveness**: Persistent data for intermittent issues
- **Operational simplicity**: Single build, runtime configuration
- **Production safety**: Disabled by default, bounded when enabled
- **Scaling ready**: Multi-instance compatible

### Negative  
- **Binary size**: ~2MB increase from SQLite (acceptable trade-off)
- **Runtime dependency**: Debug code always compiled (minimal impact when disabled)
- **Memory usage**: Bounded but non-zero when enabled

### Mitigation
- **Resource bounds**: Prevent runaway resource usage
- **Auto-cleanup**: Automatic data retention management
- **Quick disable**: Environment variable toggle for emergency shutdown

## Success Metrics

### Developer Experience
- **Setup time**: <30 seconds to enable debug mode
- **Workflow friction**: Zero additional steps for debugging
- **Bug diagnosis**: Persistent data improves intermittent bug resolution

### Production Impact
- **Disabled overhead**: <1ms per request measurement overhead
- **Memory usage**: <10MB when enabled with default limits
- **Binary size**: Accept ~2MB increase as reasonable

### Operational
- **Deployment time**: No change from current process
- **Troubleshooting**: Enable debug mode without redeploy
- **Resource usage**: Bounded and predictable

## Migration from ADR-011

### Immediate Actions
1. **Supersede ADR-011**: Mark as superseded with clear reasoning
2. **Update existing debug code**: Remove build tags, add runtime configuration  
3. **Simplify Makefile**: Single build target with environment-based testing
4. **Update documentation**: Runtime configuration examples

### Implementation Timeline
- **Week 1**: Runtime configuration framework
- **Week 2**: Bounded storage implementations, testing
- **Week 3**: Documentation, deployment guides

## References
- [ADR-010: MCP Debug System Architecture](./010-mcp-debug-system-architecture.md)
- [ADR-011: Conditional Compilation (Superseded)](./011-conditional-compilation-lightweight-debug.md)
- [Go Environment Variables](https://golang.org/pkg/os/#Getenv)
- [SQLite Go Driver](https://github.com/mattn/go-sqlite3)

## Notes
- This approach prioritizes developer experience while maintaining production safety
- Binary size increase is accepted as reasonable trade-off for debugging capabilities  
- Runtime configuration provides flexibility for different deployment scenarios
- Future optimizations available if binary size becomes critical constraint
