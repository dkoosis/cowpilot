# MCP Protocol Standards

## Protocol Overview
Model Context Protocol (MCP) is a JSON-RPC 2.0 based protocol for AI model interaction with external tools.

## Core Concepts

### Message Structure
All MCP messages follow JSON-RPC 2.0:
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {...},
  "id": 1
}
```

### Error Handling

#### JSON-RPC Error Codes
- `-32700`: Parse error
- `-32600`: Invalid request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error

#### Tool Errors
Tool execution errors return in `CallToolResult`:
```json
{
  "content": [{
    "type": "text",
    "text": "Error: division by zero"
  }],
  "isError": true
}
```

### Protocol Flow
1. **Initialize**: Client sends capabilities
2. **Tools List**: Server advertises available tools
3. **Tool Call**: Client invokes tool with arguments
4. **Result**: Server returns tool result or error

### Transport Requirements
- Support streaming responses
- Handle connection lifecycle
- Implement proper error propagation

## Implementation Guidelines

### Validation
- Validate all incoming JSON-RPC messages
- Check method names against supported methods
- Validate parameters before processing

### Security
- Never expose internal errors to clients
- Sanitize all tool outputs
- Implement rate limiting

### State Management
- Tools can be stateless or stateful
- State must be isolated per connection
- Clean up resources on disconnect
