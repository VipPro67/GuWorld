import type { WorldEvent } from "../network/Protocol.ts";

export function captureNoticeForEvent(event: WorldEvent): string | null {
  const reason = typeof event.payload?.reason === "string" ? event.payload.reason : "";

  if (event.type === "CAST_REJECTED" && event.payload?.skill_id === "SKILL_ENSLAVE_CHANNELING") {
    if (reason === "capture_gu_required") return "Cần trang bị Cổ Nô Đạo còn hoạt động để thu phục.";
    if (reason === "monster_not_subdued") return "Thú cần bị áp chế trước khi thu phục.";
    if (reason === "out_of_range") return "Mục tiêu đã ra khỏi tầm thu phục.";
    if (reason === "insufficient_mana") return "Không đủ Chân Nguyên để thu phục.";
    return "Không thể bắt đầu thu phục mục tiêu.";
  }

  switch (event.type) {
    case "CAPTURE_STARTED":
      return "Đang kết ấn Nô Đạo...";
    case "CAPTURE_SUCCESS":
      return "Đã thu phục thành công!";
    case "CAPTURE_FAILED":
      if (reason === "resisted") return "Thu phục thất bại: dã thú đã chống cự.";
      if (reason === "out_of_range") return "Thu phục thất bại: mục tiêu đã ra khỏi tầm.";
      if (reason === "target_lost") return "Thu phục thất bại: không còn mục tiêu.";
      return "Thu phục thất bại.";
    case "CAPTURE_INTERRUPTED":
      if (reason === "moved") return "Kết ấn bị gián đoạn do di chuyển.";
      if (reason === "damaged") return "Kết ấn bị gián đoạn do chịu sát thương.";
      if (reason === "crowd_control") return "Kết ấn bị gián đoạn do bị khống chế.";
      return "Kết ấn thu phục bị gián đoạn.";
    case "SOUL_BACKFIRE":
      return "Thu phục thất bại: bị phản phệ Thần Hồn!";
    default:
      return null;
  }
}
