# Claude Memory System

You have access to `memo` - a persistent memory system. Use it to store and retrieve knowledge across sessions.

## Quick Start

```bash
# At session start - recover context for this project
memo context

# Search semantically
memo similar "what you're looking for" --here

# Remember something important
memo remember <type> "content" --tags tag1,tag2
```

## Commands

| Command | Description |
|---------|-------------|
| `memo context [limit]` | Show memories for current project |
| `memo similar <query> [--here] [--limit N]` | Semantic search |
| `memo recall <query>` | Full-text search |
| `memo remember <type> <content> [--tags] [--force]` | Store a memory (dedup checked; --force to skip) |
| `memo get <id>` | View a specific memory |
| `memo update <id> <content>` | Edit a memory's content |
| `memo tag <id> <tag>` | Add a tag to a memory |
| `memo related <id>` | Find similar memories |
| `memo forget <id>` | Delete a memory |
| `memo merge <id1> <id2> ["content"]` | Merge two memories (optional content override) |
| `memo list [--type T] [--project P] [--here]` | List with filters |
| `memo prune [--days N] [--delete]` | Find stale memories (dry run by default) |
| `memo stats` | Show counts by type |
| `memo projects` | List all projects with memory counts |

## Types

- `fact` - Objective information
- `learned` - Something discovered/figured out
- `preference` - User preferences
- `context` - Project background

## Scoping

- **Project-scoped**: Memories are auto-tagged with `project:<name>` based on git repo or directory
- **Global**: Memories without a project tag are global (appear everywhere)
- Use `--here` to filter to current project only

## When to Use

- **Session start**: Run `memo context` to recover project knowledge
- **Before answering**: Check `memo similar "topic" --here` if unsure about project details
- **Learning something**: `memo remember learned "..."` - save proactively, don't wait to be asked
- **User preference**: `memo remember preference "..."` (consider keeping global - no project tag)
- **Before context compaction**: Save important discoveries
