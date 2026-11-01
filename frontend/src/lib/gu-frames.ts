/**
 * Visual frame and theme resolver for Gu according to rank (Chuyển 1-9) and path.
 */

export function getGuRankFrame(rank: number | null | undefined): string {
  const r = rank ?? 1;
  switch (r) {
    case 1:
      return "/assets/ui/frames/ui_khung_dh_chien.png";
    case 2:
      return "/assets/ui/frames/ui_khung_dh_cuong_man.png";
    case 3:
      return "/assets/ui/frames/ui_khung_dh_hoang_nghia.png";
    case 4:
      return "/assets/ui/frames/ui_khung_dh_bang_gia.png";
    case 5:
      return "/assets/ui/frames/ui_khung_dh_phuong_kim_hoang.png";
    case 6:
      return "/assets/ui/frames/ui_khung_dh_de_hoang.png";
    case 7:
      return "/assets/ui/frames/ui_khung_dh_hac_long.png";
    case 8:
      return "/assets/ui/frames/ui_khung_dh_bach_long.png";
    case 9:
      return "/assets/ui/frames/ui_khung_dh_thien_loi.png";
    default:
      return "/assets/ui/frames/ui_khung_dh_chien.png";
  }
}

export function getGuRankColor(rank: number | null | undefined): {
  border: string;
  glow: string;
  text: string;
  badge: string;
} {
  const r = rank ?? 1;
  if (r >= 6) {
    return {
      border: "border-fuchsia-400",
      glow: "shadow-[0_0_12px_rgba(217,70,239,0.5)]",
      text: "text-fuchsia-200",
      badge: "bg-fuchsia-950 text-fuchsia-300 border-fuchsia-500",
    };
  }
  if (r === 5) {
    return {
      border: "border-amber-400",
      glow: "shadow-[0_0_10px_rgba(245,158,11,0.5)]",
      text: "text-amber-200",
      badge: "bg-amber-950 text-amber-300 border-amber-500",
    };
  }
  if (r === 4) {
    return {
      border: "border-purple-400",
      glow: "shadow-[0_0_8px_rgba(168,85,247,0.4)]",
      text: "text-purple-200",
      badge: "bg-purple-950 text-purple-300 border-purple-500",
    };
  }
  if (r === 3) {
    return {
      border: "border-sky-400",
      glow: "shadow-[0_0_8px_rgba(56,189,248,0.4)]",
      text: "text-sky-200",
      badge: "bg-sky-950 text-sky-300 border-sky-500",
    };
  }
  if (r === 2) {
    return {
      border: "border-emerald-400",
      glow: "shadow-[0_0_6px_rgba(52,211,153,0.35)]",
      text: "text-emerald-200",
      badge: "bg-emerald-950 text-emerald-300 border-emerald-500",
    };
  }
  return {
    border: "border-zinc-500",
    glow: "shadow-none",
    text: "text-zinc-200",
    badge: "bg-zinc-900 text-zinc-300 border-zinc-600",
  };
}
