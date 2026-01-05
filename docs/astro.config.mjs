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
          label: "Secrets Management",
          items: [
            "guides/project-init",
            "guides/encryption",
            "guides/decryption",
            "guides/create",
            "guides/register",
            "guides/revoke",
            "guides/purge",
          ],
        },
        {
          label: "Concepts",
          items: [
            "concepts/structure",
            "concepts/encryption",
            "concepts/registration",
            "concepts/purge",
          ],
        },
        {
          label: "Configuration",
          autogenerate: { directory: "configuration" },
        },
        {
          label: "Reference",
          items: ["reference/references", "reference/credits"],
        },
      ],
    }),
  ],
});
