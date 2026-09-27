import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";
import * as jsxRuntime from "react/jsx-runtime";

const source = fs.readFileSync(new URL("./EffectBlockView.tsx", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX, esModuleInterop: true } }).outputText;
const module = { exports: {} };
const formatTarget = (target) => target?.bypass_frontline ? `${target.selector} (Xuyên hàng)` : target?.selector || null;
const localRequire = (id) => {
  if (id === "react") return { ...React, default: React };
  if (id === "react/jsx-runtime") return jsxRuntime;
  if (id === "@/lib/translate") return { translatePath: (path) => path };
  if (id === "@/lib/effect-presentation") return { formatEffectTarget: formatTarget, renderableEffectBlocks: (effects) => !effects ? { blocks: [] } : effects.schema_version === 2 ? { blocks: effects.blocks } : { blocks: [], unsupportedSchema: effects.schema_version } };
  return {};
};
new Function("require", "module", "exports", compiled)(localRequire, module, module.exports);
const { EffectEnvelopeView, formatOperation } = module.exports;

test("operation formatter includes path and native params", () => {
  assert.match(formatOperation({ op: "DAMAGE", sub_type: "FLAT", value: 12, path: "fire" }).text, /12.*fire/);
  assert.match(formatOperation({ op: "UNIQUE", code: "CAPTURE_BONUS", params: { value: 0.2 } }).text, /0.2/);
});

test("EffectEnvelopeView renders trigger, target and operation content", () => {
  const html = renderToStaticMarkup(React.createElement(EffectEnvelopeView, { effects: { schema_version: 2, blocks: [{ id: "A", trigger: "ON_ACTION", target: { selector: "TARGET", bypass_frontline: true }, conditions: [{ field: "SELF_HP_PERCENT", operator: "LTE", value: 0.5 }], operations: [{ op: "DAMAGE", value: 12 }] }] } }));
  assert.match(html, /Khi Ra Chiêu/);
  assert.match(html, /Xuyên hàng/);
  assert.match(html, /SELF_HP_PERCENT/);
  assert.match(html, /Sát thương 12/);
});
