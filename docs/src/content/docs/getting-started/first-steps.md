---
title: First steps with Kānuka
description: Getting started with Kānuka.
---

After [installing Kānuka](/getting-started/installation/), you can check that
it is available by running the `kanuka` command:

```bash
$ kanuka
Welcome to Kānuka! Run 'kanuka --help' to see available commands.
```

Run `kanuka --help` to see a list of all the available commands.

:::caution[Note]
Kānuka is under very active development, and so features will be constantly
added and changed over time. It is NOT production ready.
:::

## Getting Started with Kānuka

Kānuka provides secrets management for securely sharing environment variables and API keys across your team.

### 🔐 Secrets Management
Ideal for securely sharing environment variables and API keys across your team.

```bash
# Initialize secrets management
kanuka secrets init

# Create your encryption keys
kanuka secrets create

# Encrypt your .env file
kanuka secrets encrypt
```

**Next steps**: Continue to [project initialization](/guides/project-init/) or learn about [encryption concepts](/concepts/encryption/).

## Getting Help

- Run `kanuka --help` to see all available commands
- Use `kanuka <command> --help` for specific command help
- Check the [CLI reference](/reference/references/) for comprehensive documentation
