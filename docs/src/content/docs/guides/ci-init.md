---
title: CI Setup
description: Set up GitHub Actions CI integration with a single command.
---

The `ci-init` command automates the setup of GitHub Actions CI integration,
making it easy to decrypt secrets in your CI/CD workflows.

## Quick start

Run this command in your project directory:

```bash
kanuka secrets ci-init
```

This command:
1. Generates a dedicated CI keypair (the private key is never saved to disk)
2. Registers the CI user with your project
3. Creates a GitHub Actions workflow template
4. Securely displays the private key for you to add to GitHub Secrets

## Prerequisites

Before running `ci-init`, ensure:

- Your project is initialized with `kanuka secrets init`
- You have access to the project (ran `kanuka secrets create`)
- You're running in an interactive terminal (the private key is displayed securely)

## Step-by-step setup

### 1. Run ci-init

```bash
kanuka secrets ci-init
```

The command will display your CI private key directly to the terminal. This key
is shown only once and is never saved to disk.

:::caution[Important]
Copy the private key immediately when displayed. You cannot retrieve it later.
:::

### 2. Add the secret to GitHub

1. Go to your repository on GitHub
2. Navigate to **Settings** > **Secrets and variables** > **Actions**
3. Click **New repository secret**
4. Name: `KANUKA_PRIVATE_KEY`
5. Value: Paste the private key you copied
6. Click **Add secret**

### 3. Commit the changes

The command creates files that need to be committed:

```bash
git add .github/workflows/kanuka-decrypt.yml .kanuka/
git commit -m "Add Kanuka CI integration"
git push
```

## Generated workflow

The `ci-init` command creates a workflow at `.github/workflows/kanuka-decrypt.yml`:

```yaml
name: Decrypt Secrets

on:
  pull_request:
  push:
    branches: [main, master]

jobs:
  decrypt:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Kanuka
        uses: PolarWolf314/kanuka-actions@v1
        with:
          private-key: ${{ secrets.KANUKA_PRIVATE_KEY }}

      - name: Decrypt secrets
        run: kanuka secrets decrypt
```

You can customize this workflow to fit your needs, such as adding deployment
steps or restricting which branches trigger decryption.

## The CI user

The `ci-init` command registers a special CI user with the email:
```
41898282+github-actions[bot]@users.noreply.github.com
```

This is GitHub's official bot user email, making it clear in your project
configuration that this access is for CI automation.

## Security considerations

- **Private key display**: The key is written directly to `/dev/tty` and cleared
  from the screen after you press Enter, minimizing exposure
- **No passphrase**: The CI key has no passphrase since GitHub Secrets provides
  the protection layer
- **Dedicated keypair**: The CI user has its own keypair, separate from human users,
  making it easy to revoke CI access without affecting team members

## Reconfiguring CI access

If you need to regenerate the CI keypair (e.g., if the secret was compromised):

1. Revoke the existing CI user:
   ```bash
   kanuka secrets revoke --user 41898282+github-actions[bot]@users.noreply.github.com
   ```

2. Run `ci-init` again:
   ```bash
   kanuka secrets ci-init
   ```

3. Update the `KANUKA_PRIVATE_KEY` secret in GitHub with the new key

## Troubleshooting

### "CI integration is already configured"

The CI user is already registered. To reconfigure, first revoke the existing
CI user:

```bash
kanuka secrets revoke --user 41898282+github-actions[bot]@users.noreply.github.com
kanuka secrets ci-init
```

### "This command requires an interactive terminal"

The `ci-init` command must be run in an interactive terminal because it securely
displays the private key. Don't run it in scripts or piped commands.

### "Kanuka has not been initialized"

Initialize your project first:

```bash
kanuka secrets init
```

### "You don't have access to this project"

Create your keys first:

```bash
kanuka secrets create
```

## Next steps

- Learn about the [GitHub Action](/guides/github-action/) for more advanced workflows
- Explore [decryption options](/guides/decryption/) for CI environments
- Set up [monorepo workflows](/guides/monorepo/) for multi-service projects
