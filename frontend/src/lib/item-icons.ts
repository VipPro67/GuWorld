export interface ItemIconSource {
  id: string;
  name: string;
  item_type?: string;
  image_url?: string;
}

const iconUrl = (key: string) => "/assets/items/icons/" + key + ".webp";

export function resolveItemIconKey(item: ItemIconSource): string {
  const id = item.id.toLowerCase();
  const words = new Set((item.name.toLocaleLowerCase("vi").match(/\p{L}+/gu) ?? []));
  const has = (...terms: string[]) => terms.some((term) => words.has(term));

  if (id === "arena_coin" || has("xu", "tiền")) return "coin";
  if (id.startsWith("dao_material_")) return "dao-seal";
  if (has("rương")) return "chest";
  if (has("trứng", "noãn")) return "egg";
  if (has("đan", "hoàn")) return "pill";
  if (id === "item_medicinal_paste") return "salve";
  if (id === "item_immortal_spring_water") return "potion";
  if (id === "item_blood_ginseng" || has("sâm")) return "root";

  // Prefer the named physical object over its element or cultivation path.
  if (has("bì", "da", "mao")) return "pelt";
  if (has("nanh", "răng")) return "fang";
  if (has("trảo", "vuốt")) return "claw";
  if (has("lân", "vảy")) return "scale";
  if (has("cốt", "xương")) return "bone";
  if (has("giác", "sừng")) return "horn";
  if (has("giáp", "xác", "mai", "vỏ")) return "shell";
  if (has("nhãn", "mắt", "đồng")) return "eye";
  if (has("tâm", "tim", "thai")) return "heart";
  if (has("vũ") && !has("trụ")) return "feather";
  if (has("điệp", "trùng", "ong", "bướm") || (has("hạt") && has("tử"))) return "insect";
  if (has("hoa", "liên", "đóa")) return "flower";
  if (has("lá", "diệp")) return "leaf";
  if (has("rễ", "căn")) return "root";
  if (has("nấm") || (has("chi") && has("phàm", "linh", "tiên"))) return "mushroom";
  if (has("thảo")) return "herb";
  if (has("hạt", "chủng")) return "seed";
  if (has("quả", "trái")) return "fruit";
  if (has("ngư", "cá")) return "fish";
  if (has("châu", "trân")) return "pearl";
  if (has("sa", "cát")) return "sand";
  if (has("thiết", "khoáng", "quặng")) return "ore";
  if (has("thỏi")) return "ingot";
  if (has("mộc", "gỗ", "trúc", "thụ")) return "wood";

  // For essence and stone names, the element is the strongest visual cue.
  if (has("huyết")) return "blood";
  if (has("băng", "tuyết", "hàn")) return "ice";
  if (has("lôi", "điện")) return "lightning";
  if (has("hỏa", "viêm", "diễm")) return "ember";
  if (has("ám", "u", "huyễn", "mộng")) return "shadow";
  if (has("quang", "minh")) return "light";
  if (has("thủy", "nước")) return "water";
  if (has("phong", "gió")) return "wind";
  if (has("vân", "mây", "vụ")) return "cloud";
  if (has("thời", "tuế", "trụ")) return "hourglass";
  if (has("nguyệt")) return "moon";
  if (has("nhật")) return "sun";
  if (has("bút")) return "brush";
  if (has("xiềng")) return "chain";
  if (has("chuông")) return "bell";
  if (has("khôi") && has("lỗi")) return "gear";
  if (has("kiếm", "đao", "nhận")) return "blade";
  if (has("ngọc", "bội")) return "jade";
  if (has("tủy", "hồn", "phách")) return "vial";
  if (has("tinh", "thạch", "bảo")) return "crystal";
  return item.item_type?.toLowerCase() === "consumable" ? "potion" : "stone";
}

export function getItemIconUrl(item: ItemIconSource): string {
  const customUrl = item.image_url?.trim();
  if (customUrl && (/^https?:\/\//i.test(customUrl) || (customUrl.startsWith("/") && !customUrl.startsWith("//")))) {
    return customUrl;
  }
  return iconUrl(resolveItemIconKey(item));
}
