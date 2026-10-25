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
});

test("inventory uses bundled art for missing or invalid image URLs and keeps valid overrides", () => {
  assert.equal(getItemIconUrl({ id: "ITEM_RANDOM_RACE", name: "Hoán Huyết Đan", image_url: "200.0" }), "/assets/items/icons/pill.webp");
  assert.equal(getItemIconUrl({ id: "mine_chest", name: "Rương Mỏ Linh Thạch" }), "/assets/items/icons/chest.webp");
  assert.equal(getItemIconUrl({ id: "custom", name: "Đan", image_url: "/uploads/custom.webp" }), "/uploads/custom.webp");
});
