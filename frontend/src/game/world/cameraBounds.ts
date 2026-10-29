/** Camera bounds that keep worlds smaller than the viewport centered. */
export function centeredCameraBounds(
  x: number,
  y: number,
  width: number,
  height: number,
  viewportWidth: number,
  viewportHeight: number,
) {
  return {
    x: x - Math.max(0, viewportWidth - width) / 2,
    y: y - Math.max(0, viewportHeight - height) / 2,
    width: Math.max(width, viewportWidth),
    height: Math.max(height, viewportHeight),
  };
}
