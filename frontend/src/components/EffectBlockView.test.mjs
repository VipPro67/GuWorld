import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";
import * as jsxRuntime from "react/jsx-runtime";

const source = fs.readFileSync(new URL("./EffectBlockView.tsx", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, esModuleInterop: true } }).outputText;
const mod = { exports: {} };
const formatTarget = (target) => target?.bypass_frontline ? `${target.selector} (Xuyên hàng)` : target?.selector || null;
const localRequire = (id) => {
  if (id === "react") return { ...React, default: React };
  if (id === "react/jsx-runtime") return jsxRuntime;
  if (id === "@/lib/translate") return { translatePath: (path) => path };
  if (id === "@/lib/wiki/effect-glossary") return { COMBAT_STATUSES: [], NATIVE_HOOKS: [] };
  if (id === "@/lib/effect-presentation") return { formatEffectTarget: formatTarget, renderableEffectBlocks: (effects) => !effects ? { blocks: [] } : effects.schema_version === 2 ? { blocks: effects.blocks } : { blocks: [], unsupportedSchema: effects.schema_version } };
  return {};
};
new Function("require", "module", "exports", compiled)(localRequire, mod, mod.exports);
const { EffectEnvelopeView, formatOperation } = mod.exports;

test("operation formatter includes path and native params", () => {
  assert.match(formatOperation({ op: "DAMAGE", sub_type: "FLAT", value: 12, path: "fire" }).text, /12.*fire/);
  assert.match(formatOperation({ op: "BONUS_STRIKE", chance: 0.45, value: 0.75, min_damage: 50 }).text, /Đòn phụ 45% · ×0.75 \(sàn 50\)/);
  assert.match(formatOperation({ op: "CHAIN_DAMAGE", value: 0.5, min_damage: 30, target_count: 1 }).text, /Sát thương lan ×0.5 · 1 mục tiêu phụ \(sàn 30\)/);
  assert.match(formatOperation({ op: "UNIQUE", code: "CAPTURE_BONUS", params: { value: 0.2 } }).text, /0.2/);
  assert.match(formatOperation({ op: "REVIVE", params: { chance: 0.5, mode: "in_place" } }).text, /Hồi sinh khi chịu đòn chí tử/);
  assert.match(formatOperation({ op: "HP_COST", sub_type: "CURRENT_HP_PERCENT", value: 0.2, min_cost: 10, min_remaining: 1 }).text, /Trả 20% HP hiện tại \(tối thiểu 10\), giữ lại 1 HP/);
  assert.match(formatOperation({ op: "HEAL", sub_type: "LIFESTEAL", value: 50 }).text, /Hút máu 50%/);
  assert.match(formatOperation({ op: "HEAL", sub_type: "RESOURCE", resource: "MANA", unit: "PERCENT", value: 20 }).text, /Hồi 20% Max Chân Nguyên/);
  assert.match(formatOperation({ op: "NOTIFY", message: "battle log" }).text, /battle log/);
  assert.match(formatOperation({ op: "TRANSFER_RESOURCE", sub_type: "PERCENT_OF_MAX_HP", resource: "HP", value: 0.15, min_amount: 30 }).text, /Rút 15% Max HP \(tối thiểu 30\)/);
  assert.match(formatOperation({ op: "TRANSFER_RESOURCE", sub_type: "PERCENT_OF_TARGET_SHIELD", resource: "HP", value: 0.5 }).text, /Hấp thụ 50% Khiên mục tiêu, hồi HP cho bản thân/);
  assert.match(formatOperation({ op: "REVOKE_STATUS", status_id: "regional_wall_suppression" }).text, /Gỡ trạng thái người chơi: regional_wall_suppression/);
  assert.match(formatOperation({ op: "TARGET_DAMAGE_MODIFIER", sub_type: "REDUCE", value: 0.3, min_multiplier: 0.2 }).text, /Giảm 30% sát thương mục tiêu \(sàn ×0.2\)/);
  assert.match(formatOperation({ op: "DEFENSE_PENETRATION", value: 0.5 }).text, /Xuyên 50% phòng thủ/);
  assert.match(formatOperation({ op: "RESOURCE_CONVERSION", sub_type: "DISPEL_TO_RESOURCES" }).text, /Giải Mộng/);
  assert.match(formatOperation({ op: "TARGET_REDIRECTION", sub_type: "CONFUSION" }).text, /Chuyển hướng đòn địch/);
  assert.match(formatOperation({ op: "OUTCOME_ROULETTE", sub_type: "LUCK" }).text, /Cát Hung Chi Biến/);
  assert.match(formatOperation({ op: "OUTCOME_ROULETTE", sub_type: "CHAOS" }).text, /Hỗn Độn Đánh Cược/);
});

test("EffectEnvelopeView renders trigger, target and operation content", () => {
  const html = renderToStaticMarkup(React.createElement(EffectEnvelopeView, { effects: { schema_version: 2, blocks: [{ id: "A", trigger: "ON_ACTION", target: { selector: "TARGET", bypass_frontline: true }, conditions: [{ field: "SELF_HP_PERCENT", operator: "LTE", value: 0.5 }], operations: [{ op: "DAMAGE", value: 12 }] }] } }));
  assert.match(html, /Khi Ra Chiêu/);
  assert.match(html, /Xuyên hàng/);
  assert.match(html, /SELF_HP_PERCENT/);
  assert.match(html, /Sát thương 12/);
});
