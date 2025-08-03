# Git Commit Messages

## Full Commit Message (in COMMIT_MSG.txt)
Use for the main commit with all details:
```bash
git commit -F COMMIT_MSG.txt
```

## Short Version
For a quick commit:
```bash
git commit -m "feat: Add long-running tasks infrastructure with RTM batch operations

- Implement complete MCP-compliant task management (internal/longrunning/)
- Add 5 RTM batch tools with async progress support
- Fix Go formatting issues and ensure nil safety
- Total: 1,213 lines production code + 393 lines tests
- Known limitation: Awaiting mcp-go notification transport"
```

## Files Changed Summary
- **New Package**: internal/longrunning/ (6 files, 1,213 lines)
- **Modified**: cmd/rtm/main.go (task manager integration)
- **Modified**: internal/rtm/batch_handlers.go (5 batch tools, 369 lines)
- **Documentation**: docs/LONGRUNNING_TASKS.md, docs/STATE.yaml
- **Scripts**: 6 new build/test scripts

## Pre-commit Checklist
```bash
# 1. Format code
./run-gofmt.sh

# 2. Run tests
./test-longrunning.sh

# 3. Build check
./build-check.sh

# 4. Stage all changes
git add .

# 5. Commit with full message
git commit -F COMMIT_MSG.txt
```
