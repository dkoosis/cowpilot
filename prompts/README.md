# Prompt Library

This library contains structured prompts for use with LLMs. Each prompt includes metadata describing its purpose, inputs, and outputs.

## Prompt Index

| ID | Title | Purpose | Tags |
|----|-------|---------|------|
| 001P | [Semantic Naming & Conceptual Grouping Analysis](001P-review-semantic-naming.md) | Analyze project structure and naming for conceptual clarity and consistency | `review`, `naming`, `architecture` |
| 002P | [Code Smell Analysis](002P-analyze-code-smells.md) | Analyze TypeScript source code for common and worker-specific code smells | `review`, `code-quality`, `refactoring` |

## Usage Schedule

### Weekly Reviews (Monday morning)
- **001P-review-semantic-naming.md** - Check naming drift and new additions
- **002P-analyze-code-smells.md** - Identify accumulating technical debt

### Major Milestone Reviews
- Run both prompts before merging major features
- Run after significant refactoring

### Monthly Deep Dive
- Run with extended thinking on complex areas

## Quick Run Commands

```bash
# Generate current tree structure
tree -L 3 -I 'node_modules|dist|.git' > docs/tree.txt

# Run semantic review
# 1. Copy tree.txt content
# 2. Paste into Claude with 001P-review-semantic-naming.md

# Run code smell analysis  
# 1. Copy relevant source files
# 2. Paste into Claude with 002P-analyze-code-smells.md
```

## Tips for Claude

1. **Use Sonnet (no extended thinking)** for regular reviews
2. **Use Opus + extended thinking** only for complex architectural decisions
3. **Always provide the tree structure first**, then the prompt
4. **Save outputs** in `/reviews/YYYY-MM-DD-type.md` for tracking

## Prompt File Structure

All prompts follow the naming pattern: `###P-verb-noun-kebab-case.md`

Each prompt includes:
- YAML frontmatter with metadata
- Clear role definition
- Specific analysis criteria
- Output format specification