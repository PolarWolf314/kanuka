---
title: Credits and Acknowledgments
description: Acknowledgments for the open source projects that make Kānuka possible.
---

Kānuka is built on the shoulders of giants. We're grateful to the maintainers and contributors of these amazing open source projects that make Kānuka possible.

:::note
Kānuka is not affiliated with any of the projects listed below. We simply use and appreciate their excellent work.
:::

## Go Dependencies

### Cobra CLI Framework

**GitHub**: [spf13/cobra](https://github.com/spf13/cobra)

Cobra provides the command-line interface framework that powers Kānuka's CLI commands and help system.

### Viper Configuration

**GitHub**: [spf13/viper](https://github.com/spf13/viper)

Viper handles configuration management throughout Kānuka, providing flexible configuration options via files, environment variables, and command-line flags.

## Cryptographic Libraries

### Go Cryptography

**Documentation**: [Go crypto package](https://pkg.go.dev/crypto)

Go's standard library cryptography packages provide the foundation for Kānuka's secrets management, including RSA key generation and management.

### NaCl (Networking and Cryptography Library)

**GitHub**: [golang/crypto](https://github.com/golang/crypto)  
**Documentation**: [golang.org/x/crypto/nacl](https://pkg.go.dev/golang.org/x/crypto/nacl)

NaCl's secretbox provides the symmetric encryption used to secure your secrets files, offering authenticated encryption with excellent security properties.

## Documentation

### Starlight

**Website**: [starlight.astro.build](https://starlight.astro.build/)  
**Documentation**: [Starlight Docs](https://starlight.astro.build/getting-started/)

Starlight powers this documentation site, providing an excellent documentation framework built on Astro.

### Astro

**Website**: [astro.build](https://astro.build/)  
**Documentation**: [Astro Docs](https://docs.astro.build/)

Astro provides the static site generation framework that builds and serves this documentation.

## Thank You

We're incredibly grateful to all the maintainers, contributors, and communities behind these projects. Open source software makes projects like Kānuka possible, and we're proud to be part of this ecosystem.

If you maintain one of these projects and would like us to update how we've credited your work, please [open an issue](https://github.com/PolarWolf314/kanuka/issues) and let us know.
