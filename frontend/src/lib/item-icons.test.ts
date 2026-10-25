import assert from "node:assert/strict";
import test from "node:test";
import { getItemIconUrl, resolveItemIconKey } from "./item-icons.ts";

test("item icons follow the named object before the cultivation path", () => {
  assert.equal(resolveItemIconKey({ id: "fire_crystal_rank1", name: "Ly Hỏa Thử Bì" }), "pelt");
  assert.equal(resolveItemIconKey({ id: "fire_crystal_rank2", name: "Luyện Diễm Điểu Vũ" }), "feather");
  assert.equal(resolveItemIconKey({ id: "fire_crystal_rank3", name: "Viêm Linh Hỏa Tinh" }), "ember");
  assert.equal(resolveItemIconKey({ id: "fire_crystal_rank6", name: "Dung Nham Hoang Mãng Lân" }), "scale");
  assert.equal(resolveItemIconKey({ id: "dao_material_fire", name: "Đạo Tài Hỏa Đạo" }), "dao-seal");
  assert.equal(resolveItemIconKey({ id: "creation_stone_rank1", name: "Thổ Cư Sinh Cơ Thạch" }), "crystal");
  assert.equal(resolveItemIconKey({ id: "dark_crystal_rank1", name: "Âm Hồ Hắc Tinh" }), "crystal");
  assert.equal(resolveItemIconKey({ id: "painting_stone_rank9", name: "Giang Sơn Họa Điểu Hoang Tủy" }), "vial");
  assert.equal(resolveItemIconKey({ id: "enslavement_stone_rank1", name: "Ngự Khuyển Nô Dịch Thạch" }), "crystal");
  assert.equal(resolveItemIconKey({ id: "emotion_silk_rank1", name: "Tình Tơ Tụ Phiến" }), "silk");
  assert.equal(resolveItemIconKey({ id: "gluttonous_stomach_sac_rank2", name: "Trư Hoàng Thực Bao" }), "pouch");
  assert.equal(resolveItemIconKey({ id: "heaven_tribulation_shard_rank1", name: "Phàm Kiếp Toái Phiến" }), "shard");
  assert.equal(resolveItemIconKey({ id: "spirit_guardian_clay_rank1", name: "Sơn Thần Linh Nê" }), "clay");
  assert.equal(resolveItemIconKey({ id: "wolf_king_fang_rank1", name: "Ngự Thú Khuyển Độc Nha" }), "fang");
  assert.equal(resolveItemIconKey({ id: "iron_bear_paw_rank4", name: "Bạch Cương Hùng Chưởng" }), "claw");
  assert.equal(resolveItemIconKey({ id: "killing_iron_rank3", name: "Sát Lang Sắt Mảnh" }), "ore");
});

test("inventory uses bundled art for missing or invalid image URLs and keeps valid overrides", () => {
  assert.equal(getItemIconUrl({ id: "ITEM_RANDOM_RACE", name: "Hoán Huyết Đan", image_url: "200.0" }), "/assets/items/icons/pill.webp");
  assert.equal(getItemIconUrl({ id: "mine_chest", name: "Rương Mỏ Linh Thạch" }), "/assets/items/icons/chest.webp");
  assert.equal(getItemIconUrl({ id: "custom", name: "Đan", image_url: "/uploads/custom.webp" }), "/uploads/custom.webp");
});
