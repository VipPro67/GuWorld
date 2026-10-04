export const revalidate = 3600;
import type { Metadata } from "next";
import { fetchDictionaryUnified, fetchMasterRecipes } from "@/lib/gu-dict";
import { wikiPages } from "@/lib/wiki-pages";
import WikiClientView from "@/components/WikiClientView";

export const metadata: Metadata = {
  title: "Thư Viện Cổ Trùng",
  description: "Tra cứu toàn bộ Cổ Trùng, Vật phẩm, Quái vật, Khu vực và Cổ phương trong Cổ Giới: thuộc tính, hệ phái, tác dụng, công thức phối hợp, nguồn gốc vật phẩm.",
  openGraph: {
    title: "Thư Viện Cổ Trùng | Cổ Giới",
    description: "Tra cứu toàn bộ Cổ Trùng, Vật phẩm, Quái vật, Khu vực và Cổ phương trong Cổ Giới.",
  },
};

export default async function WikiIndexPage() {
  const { gus, items, monsters, lootTables, regions, monsterSpawns } = await fetchDictionaryUnified();
  const recipes = await fetchMasterRecipes();

  const guideSections = wikiPages.filter((p) => !p.subCategory && p.category !== "roadmap");
  const roadmap = wikiPages.find((p) => p.id === "roadmap");
  const subPages = wikiPages.filter((p) => p.subCategory);
  const subByParent: Record<string, typeof subPages> = {};
  for (const sp of subPages) {
    if (!subByParent[sp.subCategory!]) subByParent[sp.subCategory!] = [];
    subByParent[sp.subCategory!].push(sp);
  }

  const sectionIcon: Record<string, string> = {
    gameplay: "📜",
    equipment: "🛡️",
    combat: "⚔️",
    jobs: "🔧",
    activities: "🗺️",
    beasts: "🐾",
    companions: "👥",
  };

  return (
    <WikiClientView
      initialGus={gus}
      initialItems={items}
      initialRecipes={recipes}
      initialMonsters={monsters}
      initialLootTables={lootTables}
      initialRegions={regions}
      initialMonsterSpawns={monsterSpawns}
      guideSections={guideSections}
      roadmap={roadmap}
      subByParent={subByParent}
      sectionIcon={sectionIcon}
    />
  );
}
