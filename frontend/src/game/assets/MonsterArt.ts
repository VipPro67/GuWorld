import { MONSTER_ATLAS_MANIFEST } from "./monsterAtlasManifest.ts";

export interface MonsterArtCell {
  src: string;
  column: number;
  row: number;
  columns: number;
  rows: number;
}

const cells = new Map<string, MonsterArtCell>();
for (const atlas of MONSTER_ATLAS_MANIFEST) {
  const rows = Math.ceil(atlas.ids.length / atlas.columns);
  atlas.ids.forEach((id, index) => {
    cells.set(id, { src: atlas.src, column: index % atlas.columns, row: Math.floor(index / atlas.columns), columns: atlas.columns, rows });
  });
}

export function getMonsterArt(templateId: string): MonsterArtCell | null {
  return cells.get(templateId) ?? null;
}

const imageLoads = new Map<string, Promise<HTMLImageElement>>();

export function loadMonsterArtCell(cell: MonsterArtCell): Promise<HTMLCanvasElement> {
  let imageLoad = imageLoads.get(cell.src);
  if (!imageLoad) {
    imageLoad = new Promise((resolve, reject) => {
      const image = new Image();
      image.onload = () => resolve(image);
      image.onerror = () => reject(new Error(`Could not load monster art: ${cell.src}`));
      image.src = cell.src;
    });
    imageLoads.set(cell.src, imageLoad);
    void imageLoad.catch(() => imageLoads.delete(cell.src));
  }
  return imageLoad.then((image) => {
    const canvas = document.createElement("canvas");
    canvas.width = 320;
    canvas.height = 320;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("Canvas context unavailable");
    const left = Math.round(cell.column * image.width / cell.columns);
    const top = Math.round(cell.row * image.height / cell.rows);
    const right = Math.round((cell.column + 1) * image.width / cell.columns);
    const bottom = Math.round((cell.row + 1) * image.height / cell.rows);
    context.drawImage(image, left, top, right - left, bottom - top, 0, 0, 320, 320);
    return canvas;
  });
}
