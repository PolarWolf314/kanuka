---
title: Project Initialisation
description: A guide to initialising your first Kānuka project.
---

To use Kānuka on your project, it needs to be initialised. Provided Kānuka
hasn't already been initialised, it will automatically create the necessary
configuration files for your repository. You don't need any `.env` files
(secrets) to get started, as Kānuka can work, even on an empty folder.

## Getting started

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

## Next steps

To learn more about `kanuka secrets init`, see the [project structure concepts](/concepts/structure) and the [command reference](/reference/references).

Or, continue reading to learn how to encrypt secrets using Kānuka.
