import { WikiPage } from "./types";
import { roadmapPages } from "./roadmap";
import { guidePages } from "./guide";
import { gameplayPages } from "./gameplay";
import { characterPages } from "./character";
import { cultivationDaoPages } from "./cultivation-dao";
import { combatPages } from "./combat";
import { equipmentJobsPages } from "./equipment-jobs";

export * from "./types";
export * from "./roadmap";
export * from "./guide";
export * from "./gameplay";
export * from "./character";
export * from "./cultivation-dao";
export * from "./combat";
export * from "./equipment-jobs";

export const wikiPages: WikiPage[] = [
  ...roadmapPages,
  ...guidePages,
  ...cultivationDaoPages,
  ...gameplayPages,
  ...characterPages,
  ...equipmentJobsPages,
  ...combatPages,
];

export function getWikiPage(slug: string): WikiPage | undefined {
  return wikiPages.find(page => page.id === slug);
}
