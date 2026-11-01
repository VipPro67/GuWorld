import test from "node:test";
import assert from "node:assert/strict";
import { getGuIconUrl, normalizeGuSlug, hasGuIcon } from "./gu-icons.ts";

test("normalizeGuSlug cleans up Vietnamese ranks and titles", () => {
  assert.equal(normalizeGuSlug("Nhất Chuyển Bạch Ngọc Cổ"), "bach_ngoc");
  assert.equal(normalizeGuSlug("Hỏa Dực Tiên Cổ"), "hoa_duc");
  assert.equal(normalizeGuSlug("Điện Nhãn Cổ"), "dien_nhan");
});

test("getGuIconUrl resolves official icon URLs from atlas", () => {
  assert.equal(getGuIconUrl(null, "Bạch Ngọc Cổ"), "/assets/gu_icons/gu_bach_ngoc.png");
  assert.equal(getGuIconUrl(null, "Hỏa Dực Cổ"), "/assets/gu_icons/gu_hoa_du.png");
  assert.equal(getGuIconUrl(null, "Điện Nhãn Cổ"), "/assets/gu_icons/gu_dien_nhan.png");
  assert.equal(getGuIconUrl("gu_cuong_thu"), "/assets/gu_icons/gu_cuong_thu.png");
});

test("hasGuIcon returns true for available icons and false for unknowns", () => {
  assert.ok(hasGuIcon(null, "Bạch Ngọc Cổ"));
  assert.ok(hasGuIcon("gu_bang_bao"));
  assert.ok(!hasGuIcon("gu_completely_nonexistent_xyz"));
});
