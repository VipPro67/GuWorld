import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowRight, ChevronRight } from "lucide-react";
import { wikiPages, getWikiPage, type WikiPage } from "@/lib/wiki-pages";

type Props = {
  params: Promise<{ slug: string }>;
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
};

const categoryLabels: Record<string, string> = {
  roadmap: "Lộ trình",
  gameplay: "Game Play",
  equipment: "Trang bị",
  combat: "Chiến đấu",
  jobs: "Nghề",
  activities: "Hoạt động",
  beasts: "Linh thú",
  companions: "Đồng hành",
};

export async function generateStaticParams() {
  return wikiPages.map((page) => ({ slug: page.id }));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const page = getWikiPage(slug);
  if (!page) return { title: "Không Tìm Thấy" };

  return {
    title: `${page.title} | Cổ Giới`,
    description: `Hướng dẫn ${page.title.toLowerCase()} trong Cổ Giới - Wiki game Cổ Chân Nhân.`,
    openGraph: {
      title: `${page.title} | Cổ Giới`,
      description: `Hướng dẫn ${page.title.toLowerCase()} trong Cổ Giới.`,
      type: "article",
      locale: "vi_VN",
      siteName: "Cổ Giới",
    },
    twitter: {
      card: "summary",
      title: page.title,
      description: `Hướng dẫn ${page.title.toLowerCase()} trong Cổ Giới.`,
    },
    keywords: [
      page.title,
      ...page.keywords,
      "Cổ Giới",
      "GuWorld",
      "hướng dẫn",
      "wiki",
    ],
    robots: {
      index: true,
      follow: true,
    },
    alternates: {
      canonical: `/wiki/${page.id}`,
    },
  };
}

function WikiPageContent({ page }: { page: WikiPage }) {
  return (
    <div className="min-h-screen bg-[var(--bg-primary)] text-[var(--text-primary)]">
      <div className="max-w-4xl mx-auto px-4 py-8">
        <nav className="flex items-center gap-1.5 text-[10px] font-mono text-[var(--text-muted)] uppercase tracking-widest mb-6">
          <Link
            href="/wiki"
            className="hover:text-[var(--brand)] transition-colors"
          >
            Hướng Dẫn
          </Link>
          <ChevronRight className="w-3 h-3" />
          <span className="text-[var(--text-muted)]">
            {categoryLabels[page.category] || page.category}
          </span>
          {page.subCategory && (
            <>
              <ChevronRight className="w-3 h-3" />
              <span className="text-[var(--text-muted)]">
                {page.subCategory.replace("gameplay-", "")}
              </span>
            </>
          )}
          <ChevronRight className="w-3 h-3" />
          <span className="text-[var(--text-secondary)]">{page.title}</span>
        </nav>

        <h1 className="text-2xl md:text-3xl font-serif font-bold text-[var(--text-primary)] border-b border-[var(--border)]/80 pb-4 mb-6">
          {page.title}
        </h1>

        <div className="space-y-6 text-[var(--text-primary)] font-serif leading-relaxed">
          {page.content}

          {page.cards && page.cards.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-6">
              {page.cards.map((card, idx) => (
                <Link
                  key={idx}
                  href={`/wiki/${card.targetId}`}
                  className="p-4 bg-[var(--surface)]/80 border border-[var(--border)] hover:border-violet-500/20 hover:bg-[var(--bg-card)]/20 rounded-2xl group transition-all duration-300 flex items-center justify-between"
                >
                  <div className="space-y-1">
                    <h4 className="text-xs font-bold font-serif text-[var(--text-primary)] group-hover:text-indigo-400 transition-colors">
                      {card.title}
                    </h4>
                    <p className="text-[10px] text-[var(--text-muted)] font-mono">
                      {card.description}
                    </p>
                  </div>
                  <ArrowRight className="w-4 h-4 text-[var(--text-muted)] group-hover:text-indigo-400 group-hover:translate-x-1 transition-all shrink-0" />
                </Link>
              ))}
            </div>
          )}
        </div>

        <div className="border-t border-[var(--border)]/60 pt-4 mt-8 flex items-center justify-between text-[10px] text-[var(--text-muted)] font-mono">
          <span>Bản dịch & Tổng hợp theo nguyên tác Cổ Chân Nhân.</span>
          <span>Cổ Giới Wiki v1.0</span>
        </div>

        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify({
              "@context": "https://schema.org",
              "@type": "Article",
              name: page.title,
              description: `Hướng dẫn ${page.title.toLowerCase()} trong Cổ Giới`,
              genre: "GameGuide",
              keywords: page.keywords.join(", "),
              inLanguage: "vi-VN",
            }),
          }}
        />
      </div>
    </div>
  );
}

export default async function WikiGuidePage({ params }: Props) {
  const { slug } = await params;
  const page = getWikiPage(slug);
  if (!page) notFound();
  return <WikiPageContent page={page} />;
}
