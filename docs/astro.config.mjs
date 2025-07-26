// @ts-check
import { defineConfig, passthroughImageService } from "astro/config";
import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
  image: {
    service: passthroughImageService(),
  },
  integrations: [
    starlight({
      title: "Kānuka",
      favicon: "/favicon.ico",
      customCss: ["./src/styles/custom.css"],
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/PolarWolf314/kanuka",
        },
      ],
      sidebar: [
        {
          label: "Introduction",
          autogenerate: { directory: "introduction" },
        },
        {
          label: "Getting started",
          items: [
            "getting-started/installation",
            "getting-started/first-steps",
          ],
        },
        {
          label: "Grove (Development Environments)",
          items: [
            "grove-guides/environment-init",
            "grove-guides/adding-packages",
            "grove-guides/removing-packages",
            "grove-guides/searching-packages",
            "grove-guides/listing-packages",
            "grove-guides/entering-environment",
            "grove-guides/environment-status",
            "grove-guides/managing-channels",
            "grove-guides/building-containers",
            "grove-guides/aws-integration",
          ],
        },
        {
          label: "Secrets Management",
          items: [
            "guides/project-init",
            "guides/encryption",
            "guides/decryption",
            "guides/create",
            "guides/register",
            "guides/remove",
            "guides/purge",
          ],
        },
        {
          label: "Concepts",
          items: [
            {
              label: "Secrets Management",
              items: [
                "concepts/structure",
                "concepts/encryption",
                "concepts/registration",
                "concepts/purge",
              ],
            },
            {
              label: "Development Environments",
              items: [
                "concepts/grove-environments",
                "concepts/grove-packages",
                "concepts/grove-channels",
                "concepts/grove-containers",
              ],
            },
          ],
        },
        {
          label: "Configuration",
          autogenerate: { directory: "configuration" },
        },
        {
          label: "Reference",
          autogenerate: { directory: "reference" },
        },
      ],
    }),
  ],
});
