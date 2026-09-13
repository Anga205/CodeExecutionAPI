// @ts-check
import { defineConfig } from 'astro/config';
import netlify from "@astrojs/netlify";
import tailwindcss from "@tailwindcss/vite";

// https://astro.build/config
export default defineConfig({
    site: "https://compiler.anga.codes",
    output: "server",
    adapter: netlify(),
    vite: {
        resolve: {
            tsconfigPaths: true
        },
        plugins: [tailwindcss()]
    }
});
