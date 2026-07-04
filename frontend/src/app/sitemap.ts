import type { MetadataRoute } from "next";

const siteUrl = process.env.NEXT_PUBLIC_BACKEND_URL
  ? process.env.NEXT_PUBLIC_BACKEND_URL.replace("://", "://www.").replace(/:\d+$/, "")
  : "https://www.cogioi.io.vn";
const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

interface GuItem {
  id: string;
  name: string;
  rank: number;
}

const wikiGuidePages = [
  { slug: "roadmap", priority: 0.8 },
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
    const res = await fetch(`${backendUrl}/api/dictionary`, {
      cache: "force-cache",
      next: { revalidate: 3600 },
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
