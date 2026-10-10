import type { NextConfig } from "next";

const config: NextConfig = {
  output: "standalone",
  transpilePackages: ["@condo/shared"],
  outputFileTracingRoot: new URL("..", import.meta.url).pathname,
};

export default config;
