import type { MetadataRoute } from "next";
import { wikiPages } from "@/lib/wiki-pages";

// Bỏ qua pre-render tĩnh khi build Docker trên GitHub, tạo động khi có request
export const dynamic = "force-dynamic";
export const revalidate = 3600;

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || "https://www.cogioi.io.vn";
const backendUrl = process.env.INTERNAL_BACKEND_URL || process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

interface GuItem {
  id: string;
  name: string;
  rank: number;
}

async function fetchGuList(): Promise<GuItem[]> {
  try {
    // Thêm timeout 3 giây để tránh bị treo 60s nếu backend chưa sẵn sàng
    const res = await fetch(`${backendUrl}/api/dictionary/gus`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : Object.values(data || {});
  } catch {
    return [];
  }
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const gus = await fetchGuList();

  const staticPages: MetadataRoute.Sitemap = [
    {
      url: siteUrl,
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 1.0,
    },
    {
      url: `${siteUrl}/topup`,
      lastModified: new Date(),
      changeFrequency: "monthly",
      priority: 0.5,
    },
    {
      url: `${siteUrl}/wiki`,
      lastModified: new Date(),
      changeFrequency: "weekly",
      priority: 0.8,
    },
  ];

  const guPages: MetadataRoute.Sitemap = gus.map((gu) => ({
    url: `${siteUrl}/wiki/gu/${gu.id}`,
    lastModified: new Date(),
    changeFrequency: "weekly",
    priority: 0.7,
  }));

  const guidePages: MetadataRoute.Sitemap = wikiPages.map((page) => ({
    url: `${siteUrl}/wiki/${page.id}`,
    lastModified: new Date(),
    changeFrequency: "weekly",
    priority: page.id === "guide" || page.id === "roadmap" ? 0.8 : page.subCategory ? 0.6 : 0.7,
  }));

  return [...staticPages, ...guPages, ...guidePages];
}
