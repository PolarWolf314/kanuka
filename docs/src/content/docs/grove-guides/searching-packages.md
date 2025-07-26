---
title: Searching for Packages
description: A guide to discovering available packages for your Grove environment using Kānuka.
---

With thousands of packages available in the Nix ecosystem, finding the right one can be challenging. Grove provides search functionality to help you discover what's available.

:::tip
Grove searches the entire nixpkgs repository, which contains over 100,000 packages including programming languages, development tools, databases, and more!
:::

## Searching for packages

To search for packages by name:

```bash
kanuka grove search nodejs
kanuka grove search python
kanuka grove search postgres
```

This will show you all packages that match your search term, along with their descriptions.

## Search examples

Here are some useful search patterns:

```bash
# Find all Node.js related packages
kanuka grove search node

# Find Python packages
kanuka grove search python

# Find database packages
kanuka grove search database

# Find development tools
kanuka grove search dev

# Find specific tools
kanuka grove search docker
kanuka grove search git
kanuka grove search vim
```

## Understanding search results

Search results typically show:
- **Package name**: The exact name to use with `kanuka grove add`.
- **Description**: What the package does.
- **Version**: The current version available.

## Finding the right package

When searching, you might find multiple related packages:

```bash
kanuka grove search node
# Results might include:
# - nodejs (latest Node.js)
# - nodejs_18 (Node.js version 18)
# - nodejs_16 (Node.js version 16)
# - node2nix (tool for Node.js packages)
```

Choose the one that best fits your needs. For most cases, the main package name (like `nodejs`) gives you the latest stable version.

## What if you can't find a package?

If you can't find what you're looking for:

1. **Try different search terms**: Search for variations or related terms.
2. **Check the package name**: Some packages have unexpected names in nixpkgs.
3. **Look for alternatives**: There might be a similar tool with a different name.
4. **Use channels**: Try searching in different channels if you need specific versions.

## Next steps

To learn more about `kanuka grove search`, see the [package management concepts](/concepts/grove-packages) and the [command reference](/reference/references).

Or, continue reading to learn how to see what's currently in your environment.