import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { getPaginationSlice, getPaginationInfo, getPaginationRange } from "./pagination.ts";

describe("getPaginationInfo", () => {
  it("computes info correctly for total 105 with pageSize 30", () => {
    const info = getPaginationInfo(105, 1, 30);
    assert.equal(info.totalItems, 105);
    assert.equal(info.totalPages, 4);
    assert.equal(info.currentPage, 1);
    assert.equal(info.startIndex, 0);
    assert.equal(info.endIndex, 30);
  });

  it("handles pageSize 'all'", () => {
    const info = getPaginationInfo(105, 1, "all");
    assert.equal(info.totalItems, 105);
    assert.equal(info.totalPages, 1);
    assert.equal(info.currentPage, 1);
    assert.equal(info.startIndex, 0);
    assert.equal(info.endIndex, 105);
  });
});

describe("getPaginationSlice", () => {
  const dummyItems = Array.from({ length: 105 }, (_, i) => `item_${i + 1}`);

  it("handles pageSize 'all'", () => {
    const result = getPaginationSlice(dummyItems, 1, "all");
    assert.equal(result.paginatedItems.length, 105);
    assert.equal(result.totalItems, 105);
    assert.equal(result.totalPages, 1);
    assert.equal(result.currentPage, 1);
    assert.equal(result.startIndex, 0);
    assert.equal(result.endIndex, 105);
  });

  it("handles empty items array", () => {
    const result = getPaginationSlice([], 1, 20);
    assert.equal(result.paginatedItems.length, 0);
    assert.equal(result.totalItems, 0);
    assert.equal(result.totalPages, 1);
    assert.equal(result.currentPage, 1);
  });

  it("slices correctly on first page", () => {
    const result = getPaginationSlice(dummyItems, 1, 30);
    assert.equal(result.paginatedItems.length, 30);
    assert.equal(result.totalPages, 4);
    assert.equal(result.currentPage, 1);
    assert.equal(result.startIndex, 0);
    assert.equal(result.endIndex, 30);
    assert.equal(result.paginatedItems[0], "item_1");
    assert.equal(result.paginatedItems[29], "item_30");
  });

  it("slices correctly on middle page", () => {
    const result = getPaginationSlice(dummyItems, 2, 30);
    assert.equal(result.paginatedItems.length, 30);
    assert.equal(result.currentPage, 2);
    assert.equal(result.startIndex, 30);
    assert.equal(result.endIndex, 60);
    assert.equal(result.paginatedItems[0], "item_31");
    assert.equal(result.paginatedItems[29], "item_60");
  });

  it("slices correctly on last partial page", () => {
    const result = getPaginationSlice(dummyItems, 4, 30);
    assert.equal(result.paginatedItems.length, 15);
    assert.equal(result.currentPage, 4);
    assert.equal(result.startIndex, 90);
    assert.equal(result.endIndex, 105);
    assert.equal(result.paginatedItems[0], "item_91");
    assert.equal(result.paginatedItems[14], "item_105");
  });

  it("clamps out of bounds page numbers", () => {
    const resultHigh = getPaginationSlice(dummyItems, 999, 30);
    assert.equal(resultHigh.currentPage, 4);
    assert.equal(resultHigh.paginatedItems.length, 15);

    const resultLow = getPaginationSlice(dummyItems, -5, 30);
    assert.equal(resultLow.currentPage, 1);
    assert.equal(resultLow.paginatedItems.length, 30);
  });
});

describe("getPaginationRange", () => {
  it("returns single page when totalPages is 1", () => {
    assert.deepEqual(getPaginationRange(1, 1), [1]);
  });

  it("returns full range when totalPages <= 7", () => {
    assert.deepEqual(getPaginationRange(3, 5), [1, 2, 3, 4, 5]);
    assert.deepEqual(getPaginationRange(1, 7), [1, 2, 3, 4, 5, 6, 7]);
  });

  it("shows right dots when near beginning of large range", () => {
    const range = getPaginationRange(1, 20);
    assert.deepEqual(range, [1, 2, 3, 4, 5, "...", 20]);
  });

  it("shows left dots when near end of large range", () => {
    const range = getPaginationRange(20, 20);
    assert.deepEqual(range, [1, "...", 16, 17, 18, 19, 20]);
  });

  it("shows both dots when in middle of large range", () => {
    const range = getPaginationRange(10, 20);
    assert.deepEqual(range, [1, "...", 9, 10, 11, "...", 20]);
  });
});
