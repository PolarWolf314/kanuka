---
title: Searching for Packages
description: A guide to discovering available packages for your Grove environment using Kānuka.
---

With thousands of packages available in the Nix ecosystem, finding the right one can be challenging. Grove provides search functionality to help you discover what's available.

:::tip
Grove searches the entire nixpkgs repository, which contains over 100,000 packages including programming languages, development tools, databases, and more! You can also browse packages online at [NixOS Package Search](https://search.nixos.org/packages).
:::

## Searching for packages

To search for packages by name:

```bash
kanuka grove search nodejs
kanuka grove search python
kanuka grove search postgres
```

This will show you all packages that match your search term, along with their descriptions and versions.

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

Search results show packages in this format:

```
✓ Search results for 'nodejs':

nodejs_24 @ 24.4.1
  Event-driven I/O framework for the V8 JavaScript engine

nodejs_20 @ 20.19.4
  Event-driven I/O framework for the V8 JavaScript engine

nodejs-slim @ 22.17.0
  Event-driven I/O framework for the V8 JavaScript engine

→ Found 3 packages (use --details for more details)
→ Add a package: kanuka grove add <package>
```

Each result shows:
- **Package name**: The exact name to use with `kanuka grove add`.
- **Version**: The current version available (after @).
- **Description**: What the package does.

## Advanced search options

You can use additional flags to refine your search:

```bash
# Limit the number of results
kanuka grove search python --max-results 5

# Search by exact package name
kanuka grove search --name python3

# Search for packages providing a specific program
kanuka grove search --program node

# Get detailed information including programs and homepage
kanuka grove search --name python3 --details
```

The `--details` flag shows additional information like available programs and homepage URLs.

## What if you can't find a package?

If you can't find what you're looking for:

1. **Try different search terms**: Search for variations or related terms.
2. **Check the package name**: Some packages have unexpected names in nixpkgs.
3. **Look for alternatives**: There might be a similar tool with a different name.
4. **Use channels**: Try searching in different channels if you need specific versions.

## Next steps

To learn more about `kanuka grove search`, see the [package management concepts](/concepts/grove-packages) and the [command reference](/reference/references).

Or, continue reading to learn how to see what's currently in your environment.