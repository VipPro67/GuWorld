import type { MetadataRoute } from "next";

const siteUrl = process.env.NEXT_PUBLIC_BACKEND_URL
  ? process.env.NEXT_PUBLIC_BACKEND_URL.replace("://", "://www.").replace(/:\d+$/, "")
  : "https://www.cogioi.io.vn";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/api/", "/topup/"],
    },
    sitemap: `${siteUrl}/sitemap.xml`,
  };
}
