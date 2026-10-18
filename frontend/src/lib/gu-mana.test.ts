import test from "node:test";
import assert from "node:assert/strict";
import { guManaCost, portalManaCost, formatManaCost } from "./gu-mana.ts";

test("guManaCost: mortal same rank halves per sub-rank", () => {
  assert.equal(guManaCost(10, 1, 1, 1), 10);
  assert.equal(guManaCost(10, 1, 1, 2), 5);
  assert.equal(guManaCost(10, 1, 1, 3), 2.5);
  assert.equal(guManaCost(10, 1, 1, 4), 1.25);
});

test("guManaCost: mortal higher player rank divides by 10 per rank, ignores sub-rank", () => {
  assert.equal(guManaCost(10, 1, 2, 1), 1);
  assert.equal(guManaCost(10, 1, 2, 4), 1);
  assert.equal(guManaCost(10, 1, 3, 1), 0.1);
});

test("guManaCost: mortal lower player rank multiplies (table {1,2,5,20,100})", () => {
  assert.equal(guManaCost(10, 3, 1, 4), 50);
  assert.equal(guManaCost(10, 5, 1, 1), 1000);
});

test("guManaCost: immortal branch", () => {
  assert.equal(guManaCost(50, 7, 7, 1), 50);
  assert.equal(guManaCost(50, 5, 7, 1), 0); // lower-rank mortal Gu is free
  assert.equal(guManaCost(50, 8, 7, 1), 5000); // gap 1 -> x100
  assert.equal(guManaCost(50, 7, 9, 1), 0.005); // 50 / 100^2
});

test("guManaCost: mortal cannot use immortal Gu", () => {
  assert.equal(guManaCost(10, 7, 3, 1), Number.POSITIVE_INFINITY);
});

test("guManaCost: refine tier halves the cost", () => {
  assert.equal(guManaCost(10, 1, 1, 1, 1), 5);
});

test("portalManaCost mirrors GetPortalManaCost", () => {
  assert.equal(portalManaCost(10, 5, 5), 10);
  assert.equal(portalManaCost(10, 8, 6), 40); // 10 * 2^2
  assert.equal(portalManaCost(0, 3, 3), 10); // fallback for missing base
});

test("formatManaCost", () => {
  assert.equal(formatManaCost(10), "10");
  assert.equal(formatManaCost(2.5), "2.5");
  assert.equal(formatManaCost(1.25), "1.25");
  assert.equal(formatManaCost(Number.POSITIVE_INFINITY), "∞");
});
