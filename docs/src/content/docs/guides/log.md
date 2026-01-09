---
title: Viewing Operation History
description: Using the log command to view and filter the audit log.
---

The `log` command displays the history of secrets operations from the
[audit log](/guides/audit-log/).

## Basic usage

View the full log in chronological order (oldest first):

```bash
kanuka secrets log
```

Output:

```
2024-01-15 10:30:00  alice@example.com    encrypt    .env, .env.local
2024-01-15 10:35:00  bob@example.com      register   charlie@example.com
2024-01-15 11:00:00  alice@example.com    revoke     charlie@example.com
2024-01-15 11:30:00  alice@example.com    sync       3 users, 5 files
```

## Filtering entries

### By user

Show only entries from a specific user:

```bash
kanuka secrets log --user alice@example.com
```

### By operation

Show only specific operation types:

```bash
# Single operation
kanuka secrets log --operation encrypt

# Multiple operations (comma-separated)
kanuka secrets log --operation register,revoke
```

### By date

Filter entries by date range:

```bash
# Entries after a date
kanuka secrets log --since 2024-01-01

# Entries before a date
kanuka secrets log --until 2024-01-31

# Entries within a range
kanuka secrets log --since 2024-01-01 --until 2024-01-31
```

### Combining filters

Filters can be combined:

```bash
kanuka secrets log --user alice@example.com --operation encrypt --since 2024-01-01
```

## Limiting output

### Number of entries

Show only the last N entries:

```bash
kanuka secrets log -n 10
```

### Reverse order

Show most recent entries first (like `git log`):

```bash
kanuka secrets log --reverse
```

Combine with `-n` to get the N most recent entries:

```bash
kanuka secrets log --reverse -n 5
```

## Output formats

### Default format

The default format shows timestamp, user, operation, and details in columns:

```
2024-01-15 10:30:00  alice@example.com    encrypt    .env, .env.local
```

### Compact format

Use `--oneline` for a more compact format:

```bash
kanuka secrets log --oneline
```

Output:

```
2024-01-15 alice@example.com encrypt 2 files
2024-01-15 bob@example.com register charlie@example.com
```

### JSON format

For scripting and automation, output as a JSON array:

```bash
kanuka secrets log --json
```

Output:

```json
[
  {"ts":"2024-01-15T10:30:00.123456Z","user":"alice@example.com","uuid":"a1b2c3d4","op":"encrypt","files":[".env",".env.local"]},
  {"ts":"2024-01-15T10:35:00.456789Z","user":"bob@example.com","uuid":"b2c3d4e5","op":"register","target_user":"charlie@example.com"}
]
```

## Examples

### Recent activity

See what happened in the last week:

```bash
kanuka secrets log --since $(date -v-7d +%Y-%m-%d) --reverse
```

### User audit

Review all actions by a specific user:

```bash
kanuka secrets log --user alice@example.com
```

### Access changes

Track who has been added or removed:

```bash
kanuka secrets log --operation register,revoke
```

### CI/CD integration

Get recent activity as JSON for processing:

```bash
kanuka secrets log --json -n 100 | jq '.[] | select(.op == "encrypt")'
```

## When log is empty

If the audit log doesn't exist or is empty, you'll see an appropriate message:

```bash
# No log file
$ kanuka secrets log
No audit log found. Operations will be logged after running any secrets command.

# Empty log
$ kanuka secrets log
No audit log entries found.
```

## Next steps

- Learn about the [audit log format](/guides/audit-log/)
- See the [command reference](/reference/references/) for all options
