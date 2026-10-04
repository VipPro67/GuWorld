/**
 * Pagination helper utilities for Wiki and other large catalog views.
 */

export type PageSize = number | "all";

export interface PaginationInfo {
  totalItems: number;
  totalPages: number;
  currentPage: number;
  startIndex: number;
  endIndex: number;
}

export interface PaginationSliceResult<T> extends PaginationInfo {
  paginatedItems: T[];
}

/**
 * Calculates start/end indices, total pages, and safe page without slicing.
 */
export function getPaginationInfo(
  totalItems: number,
  page: number,
  pageSize: PageSize
): PaginationInfo {
  if (pageSize === "all" || totalItems === 0) {
    return {
      totalItems,
      totalPages: 1,
      currentPage: 1,
      startIndex: 0,
      endIndex: totalItems,
    };
  }

  const validPageSize = Math.max(1, pageSize);
  const totalPages = Math.max(1, Math.ceil(totalItems / validPageSize));
  const safePage = Math.min(Math.max(1, page), totalPages);

  const startIndex = (safePage - 1) * validPageSize;
  const endIndex = Math.min(startIndex + validPageSize, totalItems);

  return {
    totalItems,
    totalPages,
    currentPage: safePage,
    startIndex,
    endIndex,
  };
}

/**
 * Calculates start/end indices, total pages, and slices the array safely.
 */
export function getPaginationSlice<T>(
  items: T[],
  page: number,
  pageSize: PageSize
): PaginationSliceResult<T> {
  const info = getPaginationInfo(items.length, page, pageSize);
  const paginatedItems =
    pageSize === "all"
      ? items
      : items.slice(info.startIndex, info.endIndex);

  return {
    ...info,
    paginatedItems,
  };
}

/**
 * Generates an array of page numbers and ellipsis ("...") for rendering page buttons.
 */
export function getPaginationRange(
  currentPage: number,
  totalPages: number,
  siblingCount = 1
): (number | "...")[] {
  if (totalPages <= 1) return [1];

  // If total pages is small enough, show all page numbers
  const totalNumbers = siblingCount * 2 + 5; // e.g. 1 + 1 + 1 + 1 + 1 + 2 dots = 7
  if (totalPages <= totalNumbers) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const leftSiblingIndex = Math.max(currentPage - siblingCount, 1);
  const rightSiblingIndex = Math.min(currentPage + siblingCount, totalPages);

  const shouldShowLeftDots = leftSiblingIndex > 2;
  const shouldShowRightDots = rightSiblingIndex < totalPages - 2;

  if (!shouldShowLeftDots && shouldShowRightDots) {
    const leftItemCount = 3 + 2 * siblingCount;
    const leftRange = Array.from({ length: leftItemCount }, (_, i) => i + 1);
    return [...leftRange, "...", totalPages];
  }

  if (shouldShowLeftDots && !shouldShowRightDots) {
    const rightItemCount = 3 + 2 * siblingCount;
    const rightRange = Array.from(
      { length: rightItemCount },
      (_, i) => totalPages - rightItemCount + 1 + i
    );
    return [1, "...", ...rightRange];
  }

  if (shouldShowLeftDots && shouldShowRightDots) {
    const middleRange = Array.from(
      { length: rightSiblingIndex - leftSiblingIndex + 1 },
      (_, i) => leftSiblingIndex + i
    );
    return [1, "...", ...middleRange, "...", totalPages];
  }

  return Array.from({ length: totalPages }, (_, i) => i + 1);
}
