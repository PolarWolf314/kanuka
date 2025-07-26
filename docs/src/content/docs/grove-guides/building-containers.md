---
title: Building Containers from Your Environment
description: A guide to building containers from your Grove development environment.
---

You can build containers directly from your Grove development environment. This means your deployment containers will match your development environment exactly.

:::tip
Grove containers are incredibly efficient because they only include what you actually need. No package managers, no extra dependencies, just your packages and their requirements. Plus, they're completely reproducible!
:::

## Setting up container support

If you didn't initialize with containers, you can add support to an existing environment:

```bash
kanuka grove container init
```

Or if you're starting fresh:

```bash
kanuka grove init --containers
```

## Building your container

Once container support is enabled, building is simple:

```bash
kanuka grove container build
```

This builds a container from your current Grove environment with all your packages included.

## Using your container

After building, sync it to Docker:

```bash
kanuka grove container sync
```

Now you can use it with Docker:

```bash
docker run -it your-container-name
```

## Testing your container

You can enter the container interactively for testing:

```bash
kanuka grove container enter
```

This starts a shell inside the container, which is great for debugging.

## Next steps

To learn more about `kanuka grove container`, see the [container concepts](/concepts/grove-containers) and the [command reference](/reference/references).

Or, continue reading to learn about Grove's other features.