# ADR-013: Multi-Server Architecture with Shared Codebase

## Status
Proposed

## Context

We currently have three MCP servers in development:
1. **Everything Server** - All tools (testing/development)
2. **RTM Server** - Task management via Remember The Milk API
3. **Spektrix Server** - Customer management via Spektrix API

**Current Challenges:**
- Code duplication across server implementations
- Testing each server requires manual configuration switching
- Deployment complexity with multiple Dockerfiles/configs
- Shared infrastructure (auth, debug, middleware) needs coordination
- Risk of feature drift between servers

**Business Requirements:**
- Independent deployment of specialized servers
- Shared infrastructure code for consistency
- Isolated testing of server-specific functionality
- Ability to add new service integrations easily

## Decision

Implement a **unified codebase with modular server architecture** supporting independent development, testing, and deployment of specialized MCP servers.

### Architecture Overview

```
cowpilot/
├── cmd/
│   ├── everything-server/     # All capabilities (dev/testing)
│   ├── rtm-server/           # RTM-only production server
│   └── spektrix-server/      # Spektrix-only production server
├── internal/
│   ├── shared/               # Common server infrastructure
│   │   ├── server.go         # Base server setup
│   │   ├── middleware.go     # CORS, auth, debug
│   │   └── health.go         # Health endpoints
│   ├── rtm/                  # RTM service implementation
│   ├── spektrix/             # Spektrix service implementation
│   ├── auth/                 # Shared auth utilities
│   └── debug/                # Debug system
├── deployments/
│   ├── fly-rtm.toml          # RTM server config
│   ├── fly-spektrix.toml     # Spektrix server config
│   └── Dockerfile.template   # Parameterized build
└── scripts/
    ├── build-all.sh          # Build all servers
    ├── test-server.sh        # Test specific server
    └── deploy-server.sh      # Deploy specific server
```

### Key Design Decisions

1. **Single Repository** - All servers in one repo with shared internal packages
2. **Service Isolation** - Each service in separate internal package
3. **Shared Infrastructure** - Common auth, debug, middleware in internal/shared
4. **Independent Executables** - Separate main.go for each server
5. **Parameterized Deployment** - Single Dockerfile with build args

## Implementation Details

### Build System
```makefile
# Individual server builds
build-rtm: test-rtm
    go build -o bin/rtm-server ./cmd/rtm-server

build-spektrix: test-spektrix
    go build -o bin/spektrix-server ./cmd/spektrix-server

# Individual server tests
test-rtm:
    MCP_SERVER_TYPE=rtm go test ./internal/rtm/... ./cmd/rtm-server/...

test-spektrix:
    MCP_SERVER_TYPE=spektrix go test ./internal/spektrix/... ./cmd/spektrix-server/...

# Deploy specific server
deploy-rtm: build-rtm
    fly deploy -c deployments/fly-rtm.toml --build-arg SERVER_TYPE=rtm

deploy-spektrix: build-spektrix
    fly deploy -c deployments/fly-spektrix.toml --build-arg SERVER_TYPE=spektrix
```

### Shared Infrastructure Pattern
```go
// internal/shared/server.go
type ServerConfig struct {
    Name        string
    Version     string
    Port        string
    Services    []Service
}

type Service interface {
    SetupTools(*server.MCPServer)
    SetupResources(*server.MCPServer)
    RequiredEnvVars() []string
}

func NewMCPServer(config ServerConfig) *server.MCPServer {
    // Common server setup
}
```

### Service Implementation Pattern
```go
// internal/rtm/service.go
type RTMService struct {
    client *Client
}

func (s *RTMService) SetupTools(server *server.MCPServer) {
    // RTM-specific tools
}

func (s *RTMService) RequiredEnvVars() []string {
    return []string{"RTM_API_KEY", "RTM_API_SECRET"}
}
```

### Deployment Strategy

**Separate Fly Apps:**
- `cowpilot` - RTM server
- `mcp-adapters-spektrix` - Spektrix server
- `cowpilot-dev` - Everything server (optional)

**Parameterized Dockerfile:**
```dockerfile
ARG SERVER_TYPE=rtm
COPY . .
RUN go build -o server ./cmd/${SERVER_TYPE}-server
CMD ["./server"]
```

**Environment-Specific Configs:**
```toml
# deployments/fly-rtm.toml
app = "mcp-adapters"'
[build.args]
  SERVER_TYPE = 'rtm'
[env]
  PORT = '8081'
```

### Testing Strategy

**Server-Specific Tests:**
```bash
# Test RTM server in isolation
scripts/test-server.sh rtm

# Test Spektrix server in isolation  
scripts/test-server.sh spektrix

# Integration test against deployed servers
scripts/test-integration.sh rtm
scripts/test-integration.sh spektrix
```

**Shared Component Tests:**
```bash
# Test shared infrastructure
go test ./internal/shared/...
go test ./internal/auth/...
go test ./internal/debug/...
```

## Consequences

### Positive
- **Code Reuse**: Shared auth, debug, middleware eliminates duplication
- **Independent Deployment**: Each server deploys separately with focused functionality
- **Isolated Testing**: Test server-specific features without interference
- **Scalable**: Easy to add new service integrations
- **Consistent Infrastructure**: All servers use same patterns
- **Simplified Maintenance**: Single codebase reduces coordination overhead

### Negative
- **Increased Complexity**: More build targets and deployment configs
- **Cross-Service Dependencies**: Changes to shared code affect all servers
- **Testing Overhead**: Must test each server configuration
- **Documentation Burden**: Multiple deployment paths to document

### Risks & Mitigations

**Risk**: Shared code changes break specific servers
**Mitigation**: Comprehensive test matrix, staged deployments

**Risk**: Build complexity increases developer friction
**Mitigation**: Make targets abstract complexity, clear documentation

**Risk**: Deployment config drift between servers
**Mitigation**: Template-based configs, validation scripts

## Alternatives Considered

### 1. Separate Repositories
**Pros**: Complete isolation, simpler builds
**Cons**: Code duplication, coordination overhead, shared component versioning

### 2. Monolithic Server with Feature Flags
**Pros**: Single deployment, shared everything
**Cons**: Larger binaries, complex configuration, harder to optimize per-service

### 3. Microservices with Shared Libraries
**Pros**: Service independence, shared code as libraries
**Cons**: Library versioning complexity, deployment coordination

## Implementation Plan

### Phase 1: Refactor Existing Code (1-2 days)
1. Create `internal/shared/` package
2. Extract common server setup patterns
3. Move service-specific code to dedicated packages
4. Update existing servers to use shared infrastructure

### Phase 2: Build System (1 day)
1. Add Makefile targets for individual servers
2. Create parameterized Dockerfile
3. Add server-specific test scripts
4. Update CI/CD for multi-server builds

### Phase 3: Deployment Infrastructure (1 day)
1. Create separate Fly app configs
2. Set up environment-specific secrets
3. Add deployment scripts
4. Document deployment procedures

### Phase 4: Testing & Validation (1 day)
1. Comprehensive test suite for each server
2. Integration tests against deployed instances
3. Performance validation
4. Documentation updates

## Success Metrics
- Each server builds and deploys independently
- Zero code duplication in shared components
- Test suite covers all server configurations
- New service integration takes <1 day
- Developer can work on one service without affecting others

## References
- [Multi-Server Development Patterns](https://microservices.io/patterns/decomposition/decompose-by-service.html)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Fly.io Multi-App Deployment](https://fly.io/docs/apps/overview/)
