---
title: First Steps with Kanuka
description: Getting started with Kanuka.
---

After [installing Kanuka](/getting-started/installation/), you can check that
it is available by running the `kanuka` command:

```bash
$ kanuka
Welcome to Kanuka! Run 'kanuka --help' to see available commands.
```

Run `kanuka --help` to see a list of all the available commands.

:::caution[Note]
Kanuka is under very active development, and so features will be constantly
added and changed over time. It is NOT production ready.
:::

## Quick Start

Kanuka provides secrets management for securely sharing environment variables
and API keys across your team.

### 1. Initialize Your First Project

```bash
# Navigate to your project
cd my-project

# Initialize Kanuka
kanuka secrets init
```

If this is your first time using Kanuka, you'll be prompted to set up your
identity (email, name, and device name). This only happens once.

### 2. Encrypt Your Secrets

```bash
# Encrypt all .env files
kanuka secrets encrypt
```

This creates encrypted `.kanuka` files that are safe to commit to version control.

### 3. Share with Your Team

Commit the `.kanuka` directory and encrypted files:

```bash
git add .kanuka/ *.kanuka
git commit -m "Add encrypted secrets"
git push
```

### 4. Team Members Decrypt

When a team member clones the repo and has been registered, they can decrypt:

```bash
kanuka secrets decrypt
```

## The Kanuka Workflow

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  kanuka secrets │     │  kanuka secrets │     │     git push    │
│       init      │ ──▶ │     encrypt     │ ──▶ │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
  Sets up identity      .env → .env.kanuka      Share with team
  Creates project       Safe to commit          

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    git clone    │     │  kanuka secrets │     │  kanuka secrets │
│     git pull    │ ──▶ │     create      │ ──▶ │     decrypt     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
  Get latest secrets    Create your keys       .env.kanuka → .env
                        (if new to project)    Access your secrets
```

## Next Steps

- **[User Setup](/setup/user-setup/)** - Configure your identity
- **[Project Initialization](/guides/project-init/)** - Initialize a new project
- **[Encryption](/guides/encryption/)** - Encrypt your secrets
- **[Registration](/guides/register/)** - Add team members

## Getting Help

- Run `kanuka --help` to see all available commands
- Use `kanuka <command> --help` for specific command help
- Check the [CLI reference](/reference/references/) for comprehensive documentation
- See the [FAQ](/reference/faq/) for common questions
