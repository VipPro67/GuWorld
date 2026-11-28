import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { formatNumber } from "./utils.ts";

describe("formatNumber", () => {
  it("handles zero, negative zero, NaN, and Infinity", () => {
    assert.equal(formatNumber(0), "0");
    assert.equal(formatNumber(-0), "0");
    assert.equal(formatNumber(Number.NaN), "0");
    assert.equal(formatNumber(Number.POSITIVE_INFINITY), "0");
    assert.equal(formatNumber(Number.NEGATIVE_INFINITY), "0");
  });

  it("formats numbers below 10,000 respecting decimals", () => {
    assert.equal(formatNumber(123.456789), "123,46");
    assert.equal(formatNumber(123.456789, 1), "123,5");
    assert.equal(formatNumber(123.456789, 0), "123");
    assert.equal(formatNumber(500), "500");
    assert.equal(formatNumber(9999.99, 0), "10.000");
  });

  it("formats large numbers with Vietnamese cultivation units", () => {
    assert.equal(formatNumber(10_000), "1 Vạn");
    assert.equal(formatNumber(12_345), "1,23 Vạn");
    assert.equal(formatNumber(1_234_567), "123 Vạn");
    assert.equal(formatNumber(99_999_999, 0), "1 Ức");
    assert.equal(formatNumber(100_000_000), "1 Ức");
    assert.equal(formatNumber(1_500_000_000_000), "1,5 Triệu");
  });

  it("handles negative large numbers", () => {
    assert.equal(formatNumber(-12_345), "-1,23 Vạn");
  });
});
