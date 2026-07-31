import type { Metadata } from "next";
import Link from "next/link";
import { fetchAllGus, type GuDictionaryItem } from "@/lib/gu-dict";
import { wikiPages } from "@/lib/wiki-pages";

export const metadata: Metadata = {
  title: "Thư Viện Cổ Trùng",
  description: "Tra cứu toàn bộ Cổ Trùng trong Cổ Giới: thuộc tính, hệ phái, tác dụng, cách nuôi dưỡng và sát chiêu phối hợp.",
  openGraph: {
    title: "Thư Viện Cổ Trùng | Cổ Giới",
    description: "Tra cứu toàn bộ Cổ Trùng trong Cổ Giới.",
  },
};

const rarityColors: Record<string, string> = {
  COMMON: "text-[var(--text-secondary)]",
  UNCOMMON: "text-[var(--rarity-uncommon)]",
  RARE: "text-[var(--rarity-rare)]",
  EPIC: "text-[var(--rarity-epic)]",
  LEGENDARY: "text-[var(--rarity-legendary)]",
  MYTHIC: "text-[var(--rarity-mythic)]",
};

const rarityLabels: Record<string, string> = {
  COMMON: "Phàm",
  UNCOMMON: "Hiếm",
  RARE: "Quý",
  EPIC: "Sử thi",
  LEGENDARY: "Huyền thoại",
  MYTHIC: "Thần thoại",
};

const pathLabels: Record<string, string> = {
  "Blade Path": "Đao Đạo",
  "Blood Path": "Huyết Đạo",
  "Bone Path": "Cốt Đạo",
  "Cloud Path": "Vân Đạo",
  "Concealment Path": "Ẩn Nặc Đạo",
  "Creation Path": "Tạo Hóa Đạo",
  "Dark Path": "Hắc Ám Đạo",
  "Death Path": "Tử Vong Đạo",
  "Destruction Path": "Hủy Diệt Đạo",
  "Dream Path": "Mộng Đạo",
  "Earth Path": "Thổ Đạo",
  "Enchantment Path": "Phù Chú Đạo",
  "Enslavement Path": "Nô Dịch Đạo",
  "Fire Path": "Hỏa Đạo",
  "Food Path": "Thực Đạo",
  "Formation Path": "Trận Pháp Đạo",
  "Heaven Path": "Thiên Đạo",
  "Human Path": "Nhân Đạo",
  "Ice Path": "Hàn Băng Đạo",
  "Illusion Path": "Ảo Giác Đạo",
  "Information Path": "Thông Tin Đạo",
  "Investigation Path": "Trinh Sát Đạo",
  "Life Path": "Sinh Đạo",
  "Light Path": "Quang Minh Đạo",
  "Lightning Path": "Lôi Đạo",
  "Luck Path": "Khí Vận Đạo",
  "Metal Path": "Kim Đạo",
  "Moon Path": "Nguyệt Đạo",
  "Movement Path": "Di Động Đạo",
  "Poison Path": "Độc Đạo",
  "Qi Path": "Khí Đạo",
  "Refinement Path": "Tinh Luyện Đạo",
  "Rule Path": "Quy Tắc Đạo",
  "Shadow Path": "Ảnh Đạo",
  "Smoke Path": "Yên Vụ Đạo",
  "Soul Path": "Hồn Đạo",
  "Sound Path": "Âm Thanh Đạo",
  "Space Path": "Không Gian Đạo",
  "Strength Path": "Lực Đạo",
  "Sun Path": "Nhật Đạo",
  "Sword Path": "Kiếm Đạo",
  "Theft Path": "Trộm Đạo",
  "Theme Path": "Chủ Đề Đạo",
  "Time Path": "Thời Gian Đạo",
  "Transformation Path": "Biến Hóa Đạo",
  "Water Path": "Thủy Đạo",
  "Wind Path": "Phong Đạo",
  "Wisdom Path": "Trí Tuệ Đạo",
  "Wood Path": "Mộc Đạo",
};

function getGuTypeIcon(type: string): string {
  const icons: Record<string, string> = {
    Movement: "🦅",
    Storage: "📦",
    Healing: "❤️",
    Passive: "🔋",
    Support: "🔧",
    Defense: "🛡️",
    Consumable: "💊",
    Active: "⚡",
    Detection: "👁️",
    Illusion: "🌀",
    Summon: "👻",
  };
  return icons[type] || "🔮";
}

export default async function WikiIndexPage() {
  const gus = await fetchAllGus();

  const byRank: Record<number, GuDictionaryItem[]> = {};
  for (const gu of gus) {
    const rank = gu.rank || 0;
    if (!byRank[rank]) byRank[rank] = [];
    byRank[rank].push(gu);
  }

  const sortedRanks = Object.keys(byRank).map(Number).sort((a, b) => a - b);

  const guideSections = wikiPages.filter((p) => !p.subCategory && p.category !== "roadmap");
  const roadmap = wikiPages.find((p) => p.id === "roadmap");
  const subPages = wikiPages.filter((p) => p.subCategory);
  const subByParent: Record<string, typeof subPages> = {};
  for (const sp of subPages) {
    if (!subByParent[sp.subCategory!]) subByParent[sp.subCategory!] = [];
    subByParent[sp.subCategory!].push(sp);
  }

  const sectionIcon: Record<string, string> = {
    gameplay: "📜",
    equipment: "🛡️",
    combat: "⚔️",
    jobs: "🔧",
    activities: "🗺️",
    beasts: "🐾",
    companions: "👥",
  };

  const sectionLabels: Record<string, string> = {
    gameplay: "Hướng Dẫn Gameplay",
    equipment: "Trang Bị",
    combat: "Chiến Đấu",
    jobs: "Nghề Nghiệp",
    activities: "Hoạt Động",
    beasts: "Linh Thú",
    companions: "Đồng Hành",
  };

  return (
    <div className="min-h-screen bg-[var(--bg-primary)] text-[var(--text-primary)]">
      <div className="max-w-6xl mx-auto px-4 py-8">
        <div className="flex items-center gap-3 mb-8">
          <div className="w-10 h-10 rounded-xl bg-indigo-950/60 border border-indigo-500/20 flex items-center justify-center">
            <span className="text-lg">📖</span>
          </div>
          <div>
            <h1 className="text-2xl font-serif font-bold tracking-wider bg-gradient-to-r from-zinc-100 to-zinc-400 bg-clip-text text-transparent">
              Cổ Giới Wiki
            </h1>
            <p className="text-xs text-[var(--text-muted)] font-mono mt-0.5">
              {wikiPages.length} bài hướng dẫn · {gus.length} loại Cổ Trùng
            </p>
          </div>
        </div>

        <section className="mb-12 p-5 rounded-2xl bg-zinc-900/20 border border-[var(--border)]">
          <div className="flex items-center gap-2 mb-4">
            <span className="text-sm">📚</span>
            <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono">Hướng Dẫn Game</h2>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            {roadmap && (
              <Link
                href={`/wiki/${roadmap.id}`}
                className="flex items-center gap-2.5 p-3 rounded-xl bg-[var(--surface)] border border-[var(--border)] hover:border-[var(--brand)]/30 hover:bg-[var(--bg-elevated)] transition-all"
              >
                <span className="text-lg">🗺️</span>
                <span className="text-xs font-serif font-bold text-[var(--text-primary)] hover:text-indigo-400 transition-colors">
                  {roadmap.title}
                </span>
              </Link>
            )}
            {guideSections.map((page) => {
              const subs = subByParent[page.id];
              return (
                <div key={page.id} className="p-3 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-1.5">
                  <Link
                    href={`/wiki/${page.id}`}
                    className="flex items-center gap-2"
                  >
                    <span className="text-lg">{sectionIcon[page.category] || "📄"}</span>
                    <span className="text-xs font-serif font-bold text-[var(--text-primary)] hover:text-emerald-400 transition-colors">
                      {page.title}
                    </span>
                  </Link>
                  {subs && subs.length > 0 && (
                    <div className="pl-8 space-y-0.5">
                      {subs.map((sub) => (
                        <Link
                          key={sub.id}
                          href={`/wiki/${sub.id}`}
                          className="block text-[11px] text-[var(--text-muted)] hover:text-emerald-400 font-mono transition-colors"
                        >
                          └ {sub.title}
                        </Link>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </section>

        {gus.length === 0 ? (
          <div className="text-center py-20">
            <p className="text-[var(--text-muted)] font-mono text-sm">Không tìm thấy dữ liệu Cổ Trùng</p>
          </div>
        ) : (
          <div className="space-y-8">
            {sortedRanks.map((rank) => (
              <section key={rank}>
                <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-muted)] font-mono mb-4 border-b border-[var(--border)] pb-2">
                  Chuyển {rank}
                  <span className="text-[var(--text-muted)] ml-2 font-normal normal-case">
                    ({byRank[rank].length} loại)
                  </span>
                </h2>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  {byRank[rank]
                    .sort((a, b) => a.name.localeCompare(b.name))
                    .map((gu) => (
                      <Link
                        key={gu.id}
                        href={`/wiki/gu/${gu.id}`}
                        className="group block p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] hover:border-[var(--border-hover)] transition-all hover:bg-[var(--bg-elevated)]"
                      >
                        <div className="flex items-start justify-between gap-2">
                          <div className="flex items-center gap-2.5 min-w-0">
                            <span className="text-lg shrink-0">{getGuTypeIcon(gu.gu_type)}</span>
                            <div className="min-w-0">
                              <h3 className="text-sm font-serif font-bold text-[var(--text-primary)] group-hover:text-emerald-400 transition-colors truncate">
                                {gu.name}
                              </h3>
                              <p className="text-[10px] text-[var(--text-muted)] font-mono truncate">
                                {gu.path && gu.path.map(p => pathLabels[p] || p).join(" / ")} · {gu.gu_type}
                              </p>
                            </div>
                          </div>
                          <span className={`text-[10px] font-mono font-bold shrink-0 ${rarityColors[gu.rarity] || "text-[var(--text-muted)]"}`}>
                            {rarityLabels[gu.rarity] || gu.rarity}
                          </span>
                        </div>
                        <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mt-2 line-clamp-2">
                          {gu.description}
                        </p>
                        <div className="flex flex-wrap gap-1 mt-2">
                          {gu.combat_tags.slice(0, 3).map((tag) => (
                            <span
                              key={tag}
                              className="px-1.5 py-0.5 rounded bg-[var(--surface)] text-[9px] text-[var(--text-muted)] font-mono"
                            >
                              {tag}
                            </span>
                          ))}
                        </div>
                      </Link>
                    ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
