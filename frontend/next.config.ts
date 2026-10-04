import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone", // Thêm dòng này
  compress: false, // Caddy reverse proxy handles zstd+gzip compression
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "**",
      },
    ],
  },
  poweredByHeader: false,
  reactStrictMode: true,
};

export default nextConfig;
if (process.env.NODE_ENV === "development") {
  import('@opennextjs/cloudflare').then(m => m.initOpenNextCloudflareForDev());
}
