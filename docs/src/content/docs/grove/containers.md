---
title: Building Containers
description: A guide to building containers from your Grove development environment.
---

One of the coolest things about Grove is that you can build containers directly from your development environment. This means your deployment containers will match your development environment exactly - no more "it works on my machine" problems!

## How it works

Grove uses a tool called nix2container to build efficient, reproducible containers from your development environment. Everything you've added to your Grove environment can be packaged up into a container.

## Setting up container support

### Add containers to an existing environment
```bash
kanuka grove container init
```

This adds container support to your existing Grove environment.

### Or start with containers from the beginning
```bash
kanuka grove init --containers
```

### Build your container
```bash
kanuka grove container build
```

This builds a container from your current Grove environment.

### Sync to Docker
```bash
kanuka grove container sync
```

This copies the built container from the Nix store to your Docker daemon, so you can use it with `docker run`.

### Enter the container
```bash
kanuka grove container enter
```

This starts an interactive shell inside the container, which is great for testing and debugging.

## How to build containers

### 1. Enable container support
```bash
# For new projects
kanuka grove init --containers

# For existing projects
kanuka grove container init
```

### 2. Set up your environment
```bash
# Add packages like you normally would
kanuka grove add nodejs
kanuka grove add python3
kanuka grove add git
```

### 3. Build your container
```bash
kanuka grove container build
```

### 4. Use Container
```bash
# Sync to Docker (optional)
kanuka grove container sync

# Test interactively
kanuka grove container enter

# Or use with Docker
docker run -it <container-name>
```

## Container Configuration

When you initialize container support, Grove adds configuration to your `devenv.nix`:

```nix
{ pkgs, ... }: {
  # Your existing packages
  packages = [ pkgs.nodejs pkgs.python3 pkgs.git ];

  # Container configuration added by Grove
  containers.myapp.name = "myapp";
  containers.myapp.copyToRoot = pkgs.buildEnv {
    name = "image-root";
    paths = [ pkgs.nodejs pkgs.python3 pkgs.git ];
    pathsToLink = [ "/bin" ];
  };
}
```

## Container Features

### Reproducible Builds
- Containers built from exact same packages as development environment
- Deterministic builds using Nix
- No "works on my machine" issues

### Efficient Images
- Minimal base images
- Only includes declared dependencies
- Efficient layer caching
- Small image sizes

### Development Parity
- Same tools in container as development environment
- Same package versions
- Same configuration

## Advanced Container Usage

### Custom Container Configuration

You can customize container settings in `devenv.nix`:

```nix
{ pkgs, ... }: {
  containers.myapp = {
    name = "myapp";
    tag = "latest";
    copyToRoot = pkgs.buildEnv {
      name = "image-root";
      paths = with pkgs; [ nodejs python3 git curl ];
      pathsToLink = [ "/bin" "/lib" ];
    };
    config = {
      Cmd = [ "${pkgs.nodejs}/bin/node" "app.js" ];
      WorkingDir = "/app";
      ExposedPorts = {
        "3000/tcp" = {};
      };
    };
  };
}
```

### Multiple Containers

Define multiple containers for different purposes:

```nix
{ pkgs, ... }: {
  containers.app = {
    name = "myapp";
    # App container configuration
  };
  
  containers.worker = {
    name = "myapp-worker";
    # Worker container configuration
  };
}
```

### Container with Services

Include services in your containers:

```nix
{ pkgs, ... }: {
  services.postgres.enable = true;
  services.redis.enable = true;
  
  containers.fullstack = {
    name = "fullstack-app";
    # Container includes services
  };
}
```

## Integration with CI/CD

### GitHub Actions Example
```yaml
name: Build Container
on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: cachix/install-nix-action@v20
      - uses: cachix/cachix-action@v12
        with:
          name: devenv
      
      - name: Install devenv
        run: nix profile install nixpkgs#devenv
      
      - name: Build container
        run: kanuka grove container build
      
      - name: Push to registry
        run: |
          kanuka grove container sync
          docker tag myapp registry.example.com/myapp:${{ github.sha }}
          docker push registry.example.com/myapp:${{ github.sha }}
```

### GitLab CI Example
```yaml
build-container:
  image: nixos/nix
  script:
    - nix profile install nixpkgs#devenv
    - kanuka grove container build
    - kanuka grove container sync
    - docker tag myapp $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
```

## Container Best Practices

### Development Workflow
1. **Develop in Grove environment** for fast iteration
2. **Test in container** to verify deployment compatibility
3. **Build containers** for production deployment

### Image Optimization
1. **Minimize packages** - only include what's needed
2. **Use multi-stage builds** if needed
3. **Leverage Nix caching** for faster builds

### Security
1. **Regular updates** - update channels for security patches
2. **Minimal attack surface** - fewer packages = fewer vulnerabilities
3. **Scan images** - use container security scanning tools

## Troubleshooting

### Container Build Fails
```bash
# Check devenv configuration
kanuka grove status

# Verify packages are available
kanuka grove list

# Check container configuration in devenv.nix
```

### Sync Issues
```bash
# Ensure Docker is running
docker info

# Check if container was built
kanuka grove container build

# Try rebuilding
kanuka grove container build
kanuka grove container sync
```

### Container Won't Start
```bash
# Test interactively
kanuka grove container enter

# Check container configuration
# Verify entry point and command
```

### Size Issues
```bash
# Check what's included
kanuka grove list

# Remove unnecessary packages
kanuka grove remove <unused-package>

# Rebuild container
kanuka grove container build
```

## Container Examples

### Node.js Application
```bash
# Setup environment
kanuka grove init --containers
kanuka grove add nodejs
kanuka grove add npm

# Build and test
kanuka grove container build
kanuka grove container enter
# Test your app inside container
```

### Python Application
```bash
# Setup environment
kanuka grove init --containers
kanuka grove add python3
kanuka grove add python3Packages.pip

# Build and deploy
kanuka grove container build
kanuka grove container sync
docker run -it myapp python app.py
```

### Full-Stack Application
```bash
# Setup environment
kanuka grove init --containers
kanuka grove add nodejs
kanuka grove add python3
kanuka grove add postgresql
kanuka grove add redis

# Build complete environment
kanuka grove container build
```

## Next Steps

- Learn about [AWS integration](/grove/aws-integration/) for cloud deployment
- Explore [package management](/grove/package-management/) for optimizing containers
- Check out [channel management](/grove/channels/) for stable container builds