import test from "node:test";
import assert from "node:assert/strict";
import { GU_VFX_CATALOG, resolveSkillVfxKey } from "./GuVFXRegistry.ts";
import { getGuRankFrame, getGuRankColor } from "../../lib/gu-frames.ts";

test("GU_VFX_CATALOG covers core combat and skill VFX", () => {
  assert.ok(Object.keys(GU_VFX_CATALOG).length >= 20, "Should catalog at least 20 core VFX");

  for (const [key, def] of Object.entries(GU_VFX_CATALOG)) {
    assert.equal(def.key, key);
    assert.ok(def.textureUrl.startsWith("/assets/vfx/"), `Texture URL should start with /assets/vfx/ for ${key}`);
    assert.ok(def.atlasUrl.startsWith("/assets/vfx/"), `Atlas URL should start with /assets/vfx/ for ${key}`);
  }
});

test("resolveSkillVfxKey resolves known skill keywords to corresponding visual FX", () => {
  assert.equal(resolveSkillVfxKey("FIRE_WING", "Hỏa Dực"), "fx_gu_hoa_du");
  assert.equal(resolveSkillVfxKey("THIEM_QUANG", "Thiểm Quang Cổ"), "fx_gu_thiem_quang");
  assert.equal(resolveSkillVfxKey("DIEN_NHAN", "Điện Nhãn"), "fx_gu_dien_nhan_gpt_dau");
  assert.equal(resolveSkillVfxKey("THINH_PHONG", "Thính Phong Cổ"), "fx_gu_thinh_phong_gpt_kich_hoat");
  assert.equal(resolveSkillVfxKey("XA_TIN", "Xà Tín Cổ"), "fx_gu_xa_tin_dau_gpt_giu");
  assert.equal(resolveSkillVfxKey("KHIEN_DA", "Giáp Đá Cổ"), "fx_thu_giap_da_gpt");
  assert.equal(resolveSkillVfxKey("KHIEN_VANG", "Kim Cương Hộ Giáp"), "fx_khien_anh_kim");
  assert.equal(resolveSkillVfxKey("HOI_PHUC", "Trị Liệu"), "fx_hoi_sinh");
  assert.equal(resolveSkillVfxKey("CHUA_BIET", "Kỹ năng thường"), "fx_hit_chem");
});

test("getGuRankFrame resolves frames for ranks 1 through 9", () => {
  assert.match(getGuRankFrame(1), /ui_khung_dh_chien\.png$/);
  assert.match(getGuRankFrame(5), /ui_khung_dh_phuong_kim_hoang\.png$/);
  assert.match(getGuRankFrame(6), /ui_khung_dh_de_hoang\.png$/);
  assert.match(getGuRankFrame(9), /ui_khung_dh_thien_loi\.png$/);
  assert.match(getGuRankFrame(null), /ui_khung_dh_chien\.png$/);
});

test("getGuRankColor provides distinct visual style tiers", () => {
  assert.equal(getGuRankColor(1).border, "border-zinc-500");
  assert.equal(getGuRankColor(5).border, "border-amber-400");
  assert.equal(getGuRankColor(6).border, "border-fuchsia-400");
});
