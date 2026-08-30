import type { MetadataRoute } from "next";

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

const wikiGuidePages = [
  { slug: "roadmap", priority: 0.8 },
  { slug: "guide", priority: 0.8 },
  { slug: "gameplay-baothach", priority: 0.7 },
  { slug: "gameplay-tongmon", priority: 0.7 },
  { slug: "gameplay-character", priority: 0.7 },
  { slug: "gameplay-character-chiso", priority: 0.6 },
  { slug: "gameplay-character-thuoctinh", priority: 0.6 },
  { slug: "gameplay-character-linhcan", priority: 0.6 },
  { slug: "gameplay-character-thongthao", priority: 0.6 },
  { slug: "gameplay-character-kynang", priority: 0.6 },
  { slug: "equipment", priority: 0.7 },
  { slug: "combat", priority: 0.7 },
  { slug: "jobs", priority: 0.7 },
  { slug: "activities", priority: 0.7 },
  { slug: "beasts", priority: 0.7 },
  { slug: "companions", priority: 0.7 },
];

async function fetchGuList(): Promise<GuItem[]> {
  try {
    // Thêm timeout 3 giây để tránh bị treo 60s nếu backend chưa sẵn sàng
    const res = await fetch(`${backendUrl}/api/dictionary`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) return [];
    const data = await res.json();
    const gu: Record<string, GuItem> = data.gu || {};
    return Object.values(gu);
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

  const guidePages: MetadataRoute.Sitemap = wikiGuidePages.map((page) => ({
    url: `${siteUrl}/wiki/${page.slug}`,
    lastModified: new Date(),
    changeFrequency: "weekly",
    priority: page.priority as 0.6 | 0.7 | 0.8,
  }));

  return [...staticPages, ...guPages, ...guidePages];
}