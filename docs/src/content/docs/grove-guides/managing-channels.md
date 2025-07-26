---
title: Managing Package Channels
description: A guide to controlling package versions and sources in your Grove environment.
---

Channels are how Grove controls which versions of packages you get. Think of them as different "streams" of packages - some have the latest and greatest, others focus on stability.

:::tip
Using different channels lets you balance between having the latest features and having a stable environment. You can even mix channels in the same project - use stable for critical tools and unstable for development tools!
:::

## Seeing what channels you have

To see all the channels you've got configured:

```bash
kanuka grove channel list
```

This shows you all your channels along with their URLs and current versions.

## Adding a new channel

You can add custom channels for specific package versions:

```bash
# Add a specific nixpkgs branch
kanuka grove channel add nixos-22-11 github:NixOS/nixpkgs/nixos-22.11

# Add a custom channel
kanuka grove channel add my-packages github:myorg/my-nixpkgs
```

## Removing a channel

If you no longer need a channel:

```bash
kanuka grove channel remove <channel-name>
```

## Pinning a channel

To lock a channel to a specific version:

```bash
kanuka grove channel pin nixpkgs-stable abc123def456
```

## Updating channels

To update your channels to the latest versions:

```bash
kanuka grove channel update
```

## Next steps

To learn more about `kanuka grove channel`, see the [channel management concepts](/concepts/grove-channels) and the [command reference](/reference/references).

Or, continue reading to learn how to build containers from your environment.

