# Deployment Scripts

This directory contains scripts to help deploy and debug Cowpilot on fly.io for Claude.ai integration.

## Setup

First, make the scripts executable:
```bash
chmod +x *.sh
# or
bash setup.sh
```

## Scripts

### 1. `deploy-debug-to-fly.sh`
Main deployment script that:
- Kills local processes
- Checks fly.io status
- Sets debug environment variables
- Builds the application (with optional tests)
- Deploys to fly.io
- Verifies the deployment
- Provides registration instructions

**Usage:**
```bash
./deploy-debug-to-fly.sh
```

### 2. `check-status.sh`
Quick status checker that verifies:
- Health endpoint
- OAuth discovery endpoints
- Authentication requirement
- CORS configuration
- Debug mode status

**Usage:**
```bash
./check-status.sh
```

### 3. `monitor-registration.sh`
Real-time log monitor for debugging Claude.ai registration attempts.
Filters and highlights relevant log entries.

**Usage:**
```bash
./monitor-registration.sh
```

## Typical Workflow

1. Deploy with debug enabled:
   ```bash
   ./deploy-debug-to-fly.sh
   ```

2. Verify everything is working:
   ```bash
   ./check-status.sh
   ```

3. Start monitoring logs:
   ```bash
   ./monitor-registration.sh
   ```

4. Register on Claude.ai with:
   - **Name**: `Cowpilot Tools`
   - **Description**: `MCP server providing various utility tools including echo, time, base64 encoding and more`
   - **URL**: `https://cowpilot.fly.dev`

5. Watch the monitor output to see registration attempts and debug any issues.

## Disabling Debug Mode

After debugging, disable debug logs:
```bash
fly secrets unset -a cowpilot MCP_DEBUG MCP_DEBUG_LEVEL MCP_DEBUG_STORAGE
fly deploy -a cowpilot
```
