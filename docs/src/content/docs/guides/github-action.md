---
title: GitHub Action
description: Use the official Kānuka GitHub Action to decrypt secrets in CI/CD workflows.
---

The official [Kānuka GitHub Action](https://github.com/PolarWolf314/kanuka-actions)
simplifies using Kānuka in your GitHub Actions workflows. It handles installing
Kānuka and configuring your private key securely.

## Why use the GitHub Action?

While you can manually install Kānuka and configure keys in your workflows, the
GitHub Action provides several benefits:

- **Simplified setup** - One step to install and configure Kānuka
- **Secure key handling** - Automatically masks secrets and sets restrictive permissions
- **Version management** - Easy to pin or update Kānuka versions
- **Cross-platform** - Works on Linux and macOS runners

## Installation

Add your private key to GitHub Secrets:

1. Go to your repository's **Settings** > **Secrets and variables** > **Actions**
2. Click **New repository secret**
3. Name it `KANUKA_PRIVATE_KEY`
4. Paste your private key content (including the `-----BEGIN...` and `-----END...` lines)

If your key is passphrase-protected, add another secret named `KANUKA_PASSPHRASE`.

## Basic usage

```yaml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Kānuka
        uses: PolarWolf314/kanuka-actions@v1
        with:
          private-key: ${{ secrets.KANUKA_PRIVATE_KEY }}

      - name: Decrypt secrets
        run: kanuka secrets decrypt

      - name: Deploy
        run: ./deploy.sh
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `private-key` | Private key content for decryption | Yes | - |
| `passphrase` | Passphrase for the private key, if encrypted | No | `''` |
| `version` | Kānuka version to install (e.g., `1.2.0` or `latest`) | No | `latest` |

## Outputs

| Output | Description |
|--------|-------------|
| `private-key-path` | Path to the private key file |

## Examples

### With passphrase-protected key

```yaml
- name: Setup Kānuka
  uses: PolarWolf314/kanuka-actions@v1
  with:
    private-key: ${{ secrets.KANUKA_PRIVATE_KEY }}
    passphrase: ${{ secrets.KANUKA_PASSPHRASE }}
```

### Pinning a specific version

```yaml
- name: Setup Kānuka
  uses: PolarWolf314/kanuka-actions@v1
  with:
    private-key: ${{ secrets.KANUKA_PRIVATE_KEY }}
    version: '1.0.0'
```

### Decrypt specific files

```yaml
- name: Setup Kānuka
  uses: PolarWolf314/kanuka-actions@v1
  with:
    private-key: ${{ secrets.KANUKA_PRIVATE_KEY }}

- name: Decrypt production secrets only
  run: kanuka secrets decrypt .env.production.kanuka
```

### Monorepo with matrix strategy

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [api, web, worker]
    steps:
      - uses: actions/checkout@v4

      - name: Setup Kānuka
        uses: PolarWolf314/kanuka-actions@v1
        with:
          private-key: ${{ secrets.KANUKA_PRIVATE_KEY }}

      - name: Decrypt service secrets
        run: kanuka secrets decrypt "services/${{ matrix.service }}/.env.kanuka"

      - name: Deploy ${{ matrix.service }}
        run: ./deploy.sh ${{ matrix.service }}
```

## Security considerations

The GitHub Action takes several steps to protect your private key:

1. **Masking** - The private key and passphrase are masked in logs using `::add-mask::`
2. **Temporary storage** - The key is written to `$RUNNER_TEMP` which is cleaned up after the job
3. **Restrictive permissions** - The key file is created with `chmod 600`

:::caution
Never commit your private key to the repository. Always use GitHub Secrets to
store sensitive credentials.
:::

## Alternative: Manual setup

If you prefer not to use the GitHub Action, you can set up Kānuka manually.
See the [CI/CD section in the decryption guide](/guides/decryption/#using-in-cicd-pipelines)
for examples using `--private-key-stdin`.

## Next steps

- Learn about [decrypting secrets](/guides/decryption/)
- Explore [monorepo workflows](/guides/monorepo/)
- View the [action source code](https://github.com/PolarWolf314/kanuka-actions)
