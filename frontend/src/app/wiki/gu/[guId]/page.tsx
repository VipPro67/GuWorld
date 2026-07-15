import type { Metadata } from "next";
import Link from "next/link";
import { fetchAllGus, fetchGuById, type GuDictionaryItem } from "@/lib/gu-dict";
import { notFound } from "next/navigation";

type Props = {
  params: Promise<{ guId: string }>;
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
};

const pathLabels: Record<string, string> = {
  "Moon Path": "Nguyệt Đạo",
  "Blood Path": "Huyết Đạo",
  "Strength Path": "Lực Đạo",
  "Wood Path": "Mộc Đạo",
  "Water Path": "Thủy Đạo",
  "Earth Path": "Thổ Đạo",
  "Fire Path": "Hỏa Đạo",
  "Wind Path": "Phong Đạo",
  "Lightning Path": "Lôi Đạo",
  "Time Path": "Thời Gian Đạo",
  "Space Path": "Không Gian Đạo",
  "Wisdom Path": "Trí Tuệ Đạo",
};

const rarityLabels: Record<string, string> = {
  COMMON: "Phàm",
  UNCOMMON: "Hiếm",
  RARE: "Quý",
  EPIC: "Sử thi",
  LEGENDARY: "Huyền thoại",
  MYTHIC: "Thần thoại",
};

const rarityColors: Record<string, string> = {
  COMMON: "text-[var(--text-secondary)] border-zinc-700",
  UNCOMMON: "text-emerald-400 border-emerald-500/30",
  RARE: "text-sky-400 border-sky-500/30",
  EPIC: "text-purple-400 border-purple-500/30",
  LEGENDARY: "text-amber-400 border-amber-500/30",
  MYTHIC: "text-rose-400 border-rose-500/30",
};

const tagLabels: Record<string, string> = {
  ATTACK_CORE: "Công kích chính",
  DEFENSE_CORE: "Phòng ngự chính",
  HEAL_CORE: "Trị liệu chính",
  MOVEMENT_CORE: "Di chuyển chính",
  STORAGE: "Lưu trữ",
  UTILITY_SUPPORT: "Hỗ trợ tiện ích",
  CONSUMABLE: "Tiêu hao",
  PASSIVE_BODY: "Thụ động thân thể",
  DAMAGE: "Sát thương",
  DEFENSE: "Phòng ngự",
  DAMAGE_OVER_TIME: "Sát thương theo thời gian",
  CULTIVATION: "Tu luyện",
  MOVEMENT: "Di chuyển",
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

function renderSingleValue(key: string, value: number): string {
  const labels: Record<string, string> = {
    movement: "Tốc độ di chuyển",
    stun: "Choáng",
    storage: "Dung tích",
    generate_gu: "Sinh Cổ",
    base_damage_bonus: "Sát thương cộng thêm",
    base_defense_bonus: "Phòng ngự cộng thêm",
    add_base_attack: "Tấn công nền",
    add_base_defense: "Phòng ngự nền",
    success_rate_bonus: "Tỉ lệ thành công",
    path_conflict_reduction: "Giảm xung đột hệ",
    cleansing_power: "Tẩy rửa",
    bleed: "Chảy máu",
    blood_drain: "Hút máu",
    stealth: "Tàng hình",
    force_rank_up: "Thăng cấp cưỡng chế",
    mana_pollution: "Ô nhiễm chân nguyên",
    movement_bonus: "Hỗ trợ di chuyển",
    damage_multiplier: "Hệ số sát thương",
  };
  const label = labels[key] || key;
  return `${label}: ${value}`;
}

function formatEffectsSection(section: Record<string, number> | undefined): string[] {
  if (!section || Object.keys(section).length === 0) return [];
  return Object.entries(section).map(([k, v]) => renderSingleValue(k, v));
}

function GuStatBar({ label, value, max }: { label: string; value: number; max: number }) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  return (
    <div className="flex items-center gap-2 text-xs">
      <span className="w-24 text-[var(--text-secondary)] font-mono shrink-0">{label}</span>
      <div className="flex-1 h-2 rounded-full bg-[var(--bg-card)] overflow-hidden">
        <div
          className="h-full rounded-full bg-emerald-600/60 transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="w-8 text-right text-[var(--text-muted)] font-mono">{value}</span>
    </div>
  );
}

function GuPageContent({ gu }: { gu: GuDictionaryItem }) {
  const activeEffects = formatEffectsSection(gu.effects?.active);
  const supportEffects = formatEffectsSection(gu.effects?.support);
  const passiveEffects = formatEffectsSection(gu.effects?.passive);
  const houseEffects = formatEffectsSection(gu.effects?.house);

  const maxRank = 9;
  const maxCost = 100;

  return (
    <div className="min-h-screen bg-[var(--bg-primary)] text-[var(--text-primary)]">
      <div className="max-w-4xl mx-auto px-4 py-8">
        <nav className="text-xs text-[var(--text-muted)] font-mono mb-6">
          <Link href="/wiki" className="hover:text-[var(--brand)] transition-colors">
            ← Thư Viện Cổ Trùng
          </Link>
        </nav>

        <div className="flex items-start gap-4 mb-8">
          <div className="w-16 h-16 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] flex items-center justify-center text-3xl shrink-0">
            {getGuTypeIcon(gu.gu_type)}
          </div>
          <div className="min-w-0">
            <h1 className="text-3xl font-serif font-bold tracking-wide">{gu.name}</h1>
            <div className="flex flex-wrap items-center gap-2 mt-2">
              <span className={`px-2 py-0.5 rounded-md text-[11px] font-mono font-bold border ${rarityColors[gu.rarity] || "text-[var(--text-muted)] border-zinc-700"}`}>
                {rarityLabels[gu.rarity] || gu.rarity}
              </span>
              <span className="px-2 py-0.5 rounded-md bg-[var(--bg-card)] text-[11px] text-[var(--text-secondary)] font-mono border border-[var(--border)]">
                Chuyển {gu.rank}
              </span>
              {gu.path && gu.path.map(p => (
                <span key={p} className="px-2 py-0.5 rounded-md bg-[var(--bg-card)] text-[11px] text-[var(--text-secondary)] font-mono border border-[var(--border)]">
                  {pathLabels[p] || p}
                </span>
              ))}
              <span className="px-2 py-0.5 rounded-md bg-[var(--bg-card)] text-[11px] text-[var(--text-secondary)] font-mono border border-[var(--border)]">
                {gu.gu_type}
              </span>
              {gu.is_immortal && (
                <span className="px-2 py-0.5 rounded-md bg-amber-950/40 text-[11px] text-amber-400 font-mono border border-amber-500/30">
                  Tiên Cổ
                </span>
              )}
            </div>
          </div>
        </div>

        <div className="space-y-6">
          <section className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)]">
            <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono mb-3">Mô tả</h2>
            <p className="text-sm text-[var(--text-primary)] leading-relaxed">{gu.description}</p>
          </section>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <section className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)]">
              <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono mb-3">Chỉ số cơ bản</h2>
              <div className="space-y-3">
                <GuStatBar label="Cấp độ" value={gu.rank} max={maxRank} />
                <GuStatBar label="Tiêu hao" value={Math.round(gu.base_mana_cost)} max={maxCost} />
                <div className="flex items-center gap-2 text-xs pt-1 border-t border-[var(--border)]">
                  <span className="w-24 text-[var(--text-secondary)] font-mono shrink-0">Đói mỗi</span>
                  <span className="text-[var(--text-primary)] font-mono">{gu.hunger_rate} giây</span>
                </div>
                <div className="flex items-center gap-2 text-xs">
                  <span className="w-24 text-[var(--text-secondary)] font-mono shrink-0">No tối đa</span>
                  <span className="text-[var(--text-primary)] font-mono">{gu.max_hunger}</span>
                </div>
              </div>
            </section>

            <section className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)]">
              <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono mb-3">Vai trò</h2>
              <div className="space-y-2">
                <p className="text-xs">
                  <span className="text-[var(--text-muted)] font-mono">Vai trò chính: </span>
                  <span className="text-[var(--text-primary)] font-bold">{tagLabels[gu.primary_role] || gu.primary_role}</span>
                </p>
                <div className="flex flex-wrap gap-1.5 mt-2">
                  {gu.combat_roles.map((role) => (
                    <span key={role} className="px-2 py-0.5 rounded-md bg-indigo-950/30 text-[10px] text-indigo-400 font-mono border border-indigo-500/20">
                      {tagLabels[role] || role}
                    </span>
                  ))}
                </div>
              </div>
              <h3 className="text-xs font-bold uppercase tracking-wider text-[var(--text-muted)] font-mono mt-4 mb-2">Tags</h3>
              <div className="flex flex-wrap gap-1.5">
                {gu.combat_tags.map((tag) => (
                  <span key={tag} className="px-1.5 py-0.5 rounded bg-[var(--surface)] text-[9px] text-[var(--text-muted)] font-mono">
                    {tag}
                  </span>
                ))}
              </div>
            </section>
          </div>

          <section className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)]">
            <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono mb-4">Hiệu ứng</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {activeEffects.length > 0 && (
                <div>
                  <h3 className="text-xs font-bold text-emerald-400 font-mono mb-2">Chủ động</h3>
                  <ul className="space-y-1">
                    {activeEffects.map((e, i) => (
                      <li key={i} className="text-xs text-[var(--text-primary)] font-mono">• {e}</li>
                    ))}
                  </ul>
                </div>
              )}
              {supportEffects.length > 0 && (
                <div>
                  <h3 className="text-xs font-bold text-sky-400 font-mono mb-2">Hỗ trợ</h3>
                  <ul className="space-y-1">
                    {supportEffects.map((e, i) => (
                      <li key={i} className="text-xs text-[var(--text-primary)] font-mono">• {e}</li>
                    ))}
                  </ul>
                </div>
              )}
              {passiveEffects.length > 0 && (
                <div>
                  <h3 className="text-xs font-bold text-purple-400 font-mono mb-2">Thụ động</h3>
                  <ul className="space-y-1">
                    {passiveEffects.map((e, i) => (
                      <li key={i} className="text-xs text-[var(--text-primary)] font-mono">• {e}</li>
                    ))}
                  </ul>
                </div>
              )}
              {houseEffects.length > 0 && (
                <div>
                  <h3 className="text-xs font-bold text-amber-400 font-mono mb-2">Tiên Khiếu</h3>
                  <ul className="space-y-1">
                    {houseEffects.map((e, i) => (
                      <li key={i} className="text-xs text-[var(--text-primary)] font-mono">• {e}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          </section>

          <section className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)]">
            <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono mb-3">Thức ăn</h2>
            <div className="flex flex-wrap gap-2">
              {gu.food_types.length > 0 ? (
                gu.food_types.map((food) => (
                  <span key={food} className="px-3 py-1 rounded-xl bg-rose-950/20 text-xs text-rose-400 font-mono border border-rose-500/10">
                    🍖 {food}
                  </span>
                ))
              ) : (
                <span className="text-xs text-[var(--text-muted)] font-mono">Không cần ăn</span>
              )}
            </div>
          </section>
        </div>

        <div className="mt-8 text-center">
          <Link
            href="/wiki"
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-[var(--bg-card)] hover:bg-[var(--bg-elevated)] text-xs text-[var(--text-secondary)] hover:text-[var(--brand)] font-mono transition-colors border border-[var(--border)]"
          >
            ← Quay lại Thư Viện Cổ Trùng
          </Link>
        </div>

        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify({
              "@context": "https://schema.org",
              "@type": "CreativeWork",
              name: gu.name,
              description: gu.description,
              genre: "GameItem",
              keywords: gu.combat_tags.join(", "),
              version: gu.rank,
              creativeWorkStatus: gu.is_immortal ? "Immortal" : "Mortal",
            }),
          }}
        />
      </div>
    </div>
  );
}

export async function generateStaticParams() {
  const gus = await fetchAllGus();
  return gus.map((gu) => ({ guId: gu.id }));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { guId } = await params;
  const gu = await fetchGuById(guId);
  if (!gu) return { title: "Cổ Trùng Không Tìm Thấy" };

  const pathName = gu.path && gu.path.length > 0 ? gu.path.map(p => pathLabels[p] || p).join(" / ") : "Không có";

  return {
    title: gu.name,
    description: `${gu.name} - Cổ Trùng Chuyển ${gu.rank} hệ ${pathName} (${gu.gu_type}). ${gu.description}`,
    openGraph: {
      title: `${gu.name} | Cổ Giới`,
      description: gu.description,
      type: "article",
      locale: "vi_VN",
      siteName: "Cổ Giới",
    },
    twitter: {
      card: "summary",
      title: `${gu.name} - Cổ Trùng Chuyển ${gu.rank}`,
      description: gu.description,
    },
    keywords: [
      gu.name,
      `Cổ Trùng Chuyển ${gu.rank}`,
      pathName,
      gu.gu_type,
      ...gu.combat_tags,
    ],
    robots: {
      index: true,
      follow: true,
    },
    alternates: {
      canonical: `/wiki/gu/${gu.id}`,
    },
  };
}

export default async function GuWikiPage({ params }: Props) {
  const { guId } = await params;
  const gu = await fetchGuById(guId);
  if (!gu) notFound();
  return <GuPageContent gu={gu} />;
}
