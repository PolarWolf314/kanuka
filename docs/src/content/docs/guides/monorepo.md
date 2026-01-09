---
title: Working with Monorepos
description: Strategies for managing secrets in monorepo projects.
---

Kanuka supports two approaches for managing secrets in monorepos. Choose the
one that best fits your team's access control needs.

## Option 1: Single secrets store at root

Initialize Kanuka once at the monorepo root and use selective encryption to
manage secrets across all services.

### Setup

```bash
cd my-monorepo
kanuka secrets init
```

### Usage

Use file arguments to encrypt or decrypt specific services:

```bash
# Encrypt only the API service
kanuka secrets encrypt services/api/.env

# Encrypt multiple services
kanuka secrets encrypt services/api/.env services/web/.env

# Use glob patterns for all services
kanuka secrets encrypt "services/*/.env"

# Encrypt a specific directory
kanuka secrets encrypt services/api/

# Decrypt just what you need
kanuka secrets decrypt services/api/.env.kanuka
```

### Pros

- **Single source of truth** - One `.kanuka` directory for all access control
- **Simpler key management** - One set of keys to manage and rotate
- **Easier onboarding** - New team members only need to be registered once
- **Unified audit log** - All operations logged in one place

### Cons

- **No per-service access control** - All registered users can decrypt all secrets
- **Larger key rotation scope** - Revoking a user requires re-encrypting all secrets

### Best for

- Small to medium teams where everyone needs access to all services
- Projects where access control is handled at the repository level
- Teams prioritizing simplicity over granular permissions

## Option 2: Separate secrets stores per service

Initialize Kanuka independently in each service that needs secrets management.

### Setup

```bash
cd my-monorepo/services/api
kanuka secrets init

cd ../web
kanuka secrets init

cd ../worker
kanuka secrets init
```

### Usage

Run commands from within each service directory:

```bash
# In services/api
cd services/api
kanuka secrets encrypt
kanuka secrets decrypt

# Register a user for just this service
kanuka secrets register --user alice@example.com
```

### Pros

- **Per-service access control** - Different teams can access different services
- **Isolated key rotation** - Revoking access only affects one service
- **Independent audit logs** - Each service has its own operation history

### Cons

- **More management overhead** - Multiple `.kanuka` directories to maintain
- **Repeated onboarding** - Users may need to be registered in multiple services
- **Must remember context** - Commands must be run from the correct directory

### Best for

- Large organizations with distinct teams per service
- Projects with different security classifications per service
- Situations requiring strict access separation

## Recommendation

**Start with Option 1** (single store at root) unless you have a specific need
for per-service access control. It's simpler to manage and you can always
migrate to Option 2 later if needed.

## Migration between options

### From single store to per-service

1. Decrypt all secrets at the root level
2. Initialize Kanuka in each service directory
3. Register users as needed per service
4. Encrypt secrets in each service
5. Remove the root `.kanuka` directory

### From per-service to single store

1. Decrypt all secrets in each service
2. Remove `.kanuka` directories from each service
3. Initialize Kanuka at the root
4. Register all users who need access
5. Encrypt all secrets from the root

## CI/CD considerations

### Single store approach

Your CI/CD pipeline can decrypt specific services:

```bash
# Decrypt only what this job needs
kanuka secrets decrypt services/api/.env.kanuka
```

### Per-service approach

Your pipeline needs to run from the correct directory:

```bash
cd services/api && kanuka secrets decrypt
```

Or use separate credentials per service for additional isolation.

## Next steps

- Learn about [encrypting specific files](/guides/encryption/)
- Learn about [decrypting specific files](/guides/decryption/)
- See the [command reference](/reference/references/) for all options
