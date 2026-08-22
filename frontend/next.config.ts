import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone", // Thêm dòng này
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