export const revalidate = 3600;
export const dynamicParams = true;
import type { Metadata } from "next";
import Link from "next/link";
import { fetchAllGus, fetchGuById, fetchAllItems, type GuDictionaryItem, type GuSkillProfile, type ItemSpec } from "@/lib/gu-dict";
import { notFound } from "next/navigation";
import { EffectEnvelopeView } from "@/components/EffectBlockView";
import { translatePath } from "@/lib/translate";

type Props = {
  params: Promise<{ guId: string }>;
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
};

const rarityLabels: Record<number, string> = {
  1: "Phàm",
  2: "Tinh Phẩm",
  3: "Hiếm",
  4: "Quý",
  5: "Sử Thi",
  6: "Huyền Thoại",
  7: "Thần Thoại",
  8: "Chí Bảo",
  9: "Tuyệt Thế",
};

const rarityColors: Record<number, string> = {
  1: "text-[var(--text-secondary)] border-zinc-700",
  2: "text-[var(--rarity-uncommon)] border-emerald-500/30",
  3: "text-[var(--rarity-rare)] border-sky-500/30",
  4: "text-[var(--rarity-epic)] border-purple-500/30",
  5: "text-[var(--rarity-legendary)] border-amber-500/30",
  6: "text-[var(--rarity-mythic)] border-rose-500/30",
  7: "text-red-400 border-red-500/30",
  8: "text-orange-400 border-orange-500/30",
  9: "text-fuchsia-400 border-fuchsia-500/30",
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
    movement: "Phạm vi di chuyển (ô)",
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
    movement_bonus: "Cộng phạm vi di chuyển",
    damage_multiplier: "Hệ số sát thương",
  };
  const label = labels[key] || key;
  return `${label}: ${value}`;
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

function GuPageContent({ gu, itemMap }: { gu: GuDictionaryItem; itemMap: Record<string, ItemSpec> }) {

  const maxRank = 9;
  const maxCost = 100;

  return (
    <div className="min-h-screen bg-[var(--bg-primary)] text-[var(--text-primary)]">
      <div className="max-w-4xl mx-auto px-2 sm:px-4 py-8">
        <nav className="text-xs text-[var(--text-muted)] font-mono mb-6 flex items-center gap-3">
          <Link href="/" className="hover:text-[var(--brand)] transition-colors font-bold text-emerald-400">
            ← Quay về Game
          </Link>
          <span className="text-zinc-700">|</span>
          <Link href="/wiki" className="hover:text-[var(--brand)] transition-colors">
            Thư Viện Cổ Trùng
          </Link>
        </nav>

        <div className="flex items-start gap-4 mb-8">
          <div className="w-16 h-16 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] flex items-center justify-center text-3xl shrink-0">
            {getGuTypeIcon(gu.gu_type)}
          </div>
          <div className="min-w-0">
            <h1 className="text-3xl font-serif font-bold tracking-wide">{gu.name}</h1>
            <div className="flex flex-wrap items-center gap-2 mt-2">
              {gu.rarity > 0 && (
                <span className={`px-2 py-0.5 rounded-md text-[11px] font-mono font-bold border ${rarityColors[gu.rarity] || "text-[var(--text-muted)] border-zinc-700"}`}>
                  {rarityLabels[gu.rarity] || `Độ hiếm ${gu.rarity}`}
                </span>
              )}
              <span className="px-2 py-0.5 rounded-md bg-[var(--bg-card)] text-[11px] text-[var(--text-secondary)] font-mono border border-[var(--border)]">
                Chuyển {gu.rank}
              </span>
              {gu.path && gu.path.map(p => (
                <span key={p} className="px-2 py-0.5 rounded-md bg-[var(--bg-card)] text-[11px] text-[var(--text-secondary)] font-mono border border-[var(--border)]">
                  {translatePath(p) || p}
                </span>
              ))}
              <span className="px-2 py-0.5 rounded-md bg-[var(--bg-card)] text-[11px] text-[var(--text-secondary)] font-mono border border-[var(--border)]">
                {gu.gu_type}
              </span>
              {gu.rank >= 6 && (
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

          {gu.skill_profile && <GuSkillCard skill={gu.skill_profile} />}

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
              <h3 className="text-xs font-bold uppercase tracking-wider text-[var(--text-muted)] font-mono mt-4 mb-2">Thẻ</h3>
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
            <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
              <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono">Hiệu ứng</h2>
              <Link href="/wiki/effects" className="text-[10px] font-mono text-emerald-400 hover:text-emerald-300 transition-colors">
                Cách đọc hiệu ứng →
              </Link>
            </div>
            <EffectEnvelopeView effects={gu.effects} />
          </section>

          <section className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)]">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono">Thức ăn</h2>
              {gu.food_types.length > 0 && (
                <span className="text-[11px] text-[var(--text-muted)] font-mono">
                  {gu.food_types.length} loại vật phẩm
                </span>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              {gu.food_types.length > 0 ? (
                gu.food_types.map((food) => {
                  const item = itemMap[food];
                  const foodName = item ? item.name : food;
                  return (
                    <Link
                      key={food}
                      href={`/wiki?tab=items&search=${encodeURIComponent(foodName)}`}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-rose-950/25 hover:bg-rose-900/40 text-xs text-rose-300 hover:text-rose-200 font-mono border border-rose-500/20 hover:border-rose-400/40 transition-all cursor-pointer shadow-sm group"
                      title={item ? `${item.name} (${item.item_type || "Vật phẩm"})` : food}
                    >
                      <span className="group-hover:scale-110 transition-transform">🍖</span>
                      <span className="font-semibold">{foodName}</span>
                      {item && item.rank && item.rank > 0 && (
                        <span className="text-[10px] text-rose-400/70 font-normal ml-0.5">
                          · Cấp {item.rank}
                        </span>
                      )}
                    </Link>
                  );
                })
              ) : (
                <span className="text-xs text-[var(--text-muted)] font-mono">Không cần ăn</span>
              )}
            </div>
          </section>

          <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
            <Link
              href={`/wiki?tab=recipes&search=${encodeURIComponent(gu.name)}`}
              className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl bg-amber-950/20 hover:bg-amber-900/30 text-xs text-amber-300 hover:text-amber-200 font-mono border border-amber-500/20 hover:border-amber-400/40 transition-all"
            >
              <span>📜</span>
              <span>Tìm Cổ phương luyện {gu.name}</span>
            </Link>

            <Link
              href="/wiki"
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-[var(--bg-card)] hover:bg-[var(--bg-elevated)] text-xs text-[var(--text-secondary)] hover:text-[var(--brand)] font-mono transition-colors border border-[var(--border)]"
            >
              ← Quay lại Thư Viện Cổ Trùng
            </Link>
          </div>
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
              creativeWorkStatus: gu.rank >= 6 ? "Immortal" : "Mortal",
            }),
          }}
        />
      </div>
    </div>
  );
}

function GuSkillCard({ skill }: { skill: GuSkillProfile }) {
  const range = Math.max(skill.cast_range, 1);
  const scale = 68 / range;
  const targetX = skill.cast_mode === "SELF" ? 90 : 90 + Math.min(skill.cast_range * scale * 0.65, 58);
  const targetY = 90;
  const radius = Math.min(skill.radius * scale, 48);
  const shape = skill.shape === "LINE"
    ? <rect x={targetX} y={targetY - Math.max(3, skill.width * scale / 2)} width={Math.min(70, skill.length * scale)} height={Math.max(6, skill.width * scale)} rx="3" />
    : skill.shape === "CONE"
      ? <path d={`M ${targetX} ${targetY} L ${Math.min(176, targetX + skill.length * scale)} ${Math.max(0, targetY - skill.length * scale * Math.tan((skill.angle_deg || 60) * Math.PI / 360))} A ${Math.min(70, skill.length * scale)} ${Math.min(70, skill.length * scale)} 0 0 1 ${Math.min(176, targetX + skill.length * scale)} ${Math.min(180, targetY + skill.length * scale * Math.tan((skill.angle_deg || 60) * Math.PI / 360))} Z`} />
      : skill.shape === "CIRCLE"
        ? <circle cx={targetX} cy={targetY} r={Math.max(4, radius)} />
        : <circle cx={targetX} cy={targetY} r="7" />;
  const modeLabels = { TARGET: "Chọn mục tiêu", DIRECTION: "Chọn hướng", GROUND: "Chọn điểm", SELF: "Tự thân" };
  return <section className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)]">
    <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono mb-3">Thông tin chiêu</h2>
    <div className="grid md:grid-cols-[180px_1fr] gap-5 items-center">
      <svg viewBox="0 0 180 180" role="img" aria-label={`Minh họa vùng chiêu ${skill.shape}, tầm ${skill.cast_range}`} className="w-44 h-44 rounded-xl bg-slate-950/70 border border-[var(--border)]">
        <circle cx="90" cy="90" r="68" fill="none" stroke="#475569" strokeDasharray="3 4" />
        <line x1="90" y1="90" x2={targetX} y2={targetY} stroke="#64748b" strokeDasharray="3 3" />
        <g fill="rgba(16,185,129,.28)" stroke="#34d399" strokeWidth="2">{shape}</g>
        <circle cx="90" cy="90" r="5" fill="#60a5fa" stroke="#bfdbfe" />
        <text x="90" y="172" textAnchor="middle" fill="#94a3b8" fontSize="9">Tầm tối đa</text>
      </svg>
      <div className="grid grid-cols-2 gap-2 text-xs">
        <SkillStat label="Cách ngắm" value={modeLabels[skill.cast_mode]} />
        <SkillStat label="Vùng tác dụng" value={skill.shape} />
        <SkillStat label="Tầm dùng" value={`${skill.cast_range} ô`} />
        <SkillStat label="Bán kính" value={skill.radius ? `${skill.radius} ô` : "—"} />
        <SkillStat label="Niệm chiêu" value={`${(skill.cast_time_ms / 1000).toFixed(1)} giây`} />
        <SkillStat label="Hồi chiêu" value={`${(skill.cooldown_ms / 1000).toFixed(1)} giây`} />
        <SkillStat label="Hậu động tác" value={`${(skill.recovery_ms / 1000).toFixed(1)} giây`} />
        <SkillStat label="Mục tiêu tối đa" value={String(skill.max_targets)} />
      </div>
    </div>
    <p className="mt-3 text-[10px] text-[var(--text-muted)]">Sơ đồ minh họa theo dữ liệu vùng đánh; tâm xanh là người dùng, vòng nét đứt là tầm thi triển.</p>
  </section>;
}

function SkillStat({ label, value }: { label: string; value: string }) {
  return <div className="rounded-lg bg-[var(--surface)] px-3 py-2"><div className="text-[var(--text-muted)]">{label}</div><div className="mt-1 font-mono text-[var(--text-primary)]">{value}</div></div>;
}

export async function generateStaticParams() {
  const gus = await fetchAllGus();
  const uniqueIds = Array.from(new Set(gus.map((gu) => gu.id)));
  return uniqueIds.map((guId) => ({ guId }));
}

export async function generateMetadata({ params, searchParams }: Props): Promise<Metadata> {
  const { guId } = await params;
  const sp = await searchParams;
  const rank = Number(sp?.rank) || undefined;
  const gu = await fetchGuById(guId, rank);
  if (!gu) return { title: "Cổ Trùng Không Tìm Thấy" };

  const pathName = gu.path && gu.path.length > 0 ? gu.path.map(p => translatePath(p) || p).join(" / ") : "Không có";

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
      canonical: `/wiki/gu/${gu.id}${gu.rank ? `?rank=${gu.rank}` : ""}`,
    },
  };
}

export default async function GuWikiPage({ params, searchParams }: Props) {
  const { guId } = await params;
  const sp = await searchParams;
  const rank = Number(sp?.rank) || undefined;
  const [gu, items] = await Promise.all([
    fetchGuById(guId, rank),
    fetchAllItems(),
  ]);
  if (!gu) notFound();

  const itemMap: Record<string, ItemSpec> = {};
  for (const item of items) {
    itemMap[item.id] = item;
  }

  return <GuPageContent gu={gu} itemMap={itemMap} />;
}
