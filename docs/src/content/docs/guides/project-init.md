---
title: Project Initialisation
description: A guide to initialising your first Kānuka project.
---

To use Kānuka on your project, it needs to be initialised. Provided Kānuka
hasn't already been initialised, it will automatically create the necessary
configuration files for your repository. You don't need any `.env` files
(secrets) to get started, as Kānuka can work, even on an empty folder.

## Getting Started

To initialise Kānuka on a new project, run the following commands:

```bash
# Create the directory for your new project
mkdir my_new_project
# Navigate to the project
cd my_new_project
# Initialise Kānuka
kanuka secrets init
```

That's it! If you want to initialise Kānuka on an existing project, just
navigate to the root of that project and run:

```bash
kanuka secrets init
```

## First-Time User Setup

If this is your first time using Kānuka, the `secrets init` command will
automatically prompt you to set up your user identity:

```bash
$ kanuka secrets init
⚠ User configuration not found.

Running initial setup...

Welcome to Kānuka! Let's set up your identity.

Email address: alice@example.com
Display name (optional): Alice Smith
Default device name [MacBook-Pro]: 

✓ User configuration saved to ~/.config/kanuka/config.toml

Your settings:
  Email:   alice@example.com
  Name:    Alice Smith
  Device:  MacBook-Pro
  User ID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8

Initializing project...
```

This setup only happens once. On subsequent projects, Kānuka will use your
existing identity.

You can also set up your identity before initializing any projects by running
`kanuka config init`. See [User Setup](/setup/user-setup/) for more details.

## Project Name

After user setup, you'll be prompted for a project name:

```bash
Project name [my_new_project]: 
```

Press Enter to accept the default (your directory name) or type a custom name.

## Non-Interactive Mode

For CI/CD pipelines or scripts, use the `--yes` flag to skip prompts:

```bash
kanuka secrets init --yes
```

This requires your user configuration to already be set up (via `kanuka config init`).
If the user config is incomplete, the command will fail with a clear error message.

You can also specify the project name:

```bash
kanuka secrets init --name "My Project" --yes
```

## What Gets Created

After initialization, your project will have:

```
my_new_project/
├── .kanuka/
│   ├── config.toml           # Project configuration
│   ├── public_keys/
│   │   └── <your-uuid>.pub   # Your public key
│   └── secrets/
│       └── <your-uuid>.kanuka # Your encrypted symmetric key
└── (your project files)
```

Additionally, on your local machine:
- A new key pair is created in `~/.local/share/kanuka/keys/<project-uuid>/`
- Your user config is updated with this project entry

## Next Steps

To learn more about `kanuka secrets init`, see the [project structure concepts](/concepts/structure), the [project configuration concepts](/concepts/project-configuration/), and the [command reference](/reference/references).

Or, continue reading to learn how to encrypt secrets using Kānuka.
