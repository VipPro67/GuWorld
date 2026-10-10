"use client";

import React, { useMemo, useState } from "react";
import { BookOpen, ChevronRight, Search } from "lucide-react";
import { wikiPages } from "@/lib/wiki-pages";

const categoryLabels: Record<string, string> = {
  roadmap: "Thông tin cập nhật",
  guide: "Hướng dẫn",
  gameplay: "Gameplay",
  character: "Nhân vật",
  combat: "Chiến đấu",
  equipment: "Trang bị & nghề nghiệp",
};

export const WikiView: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedPageId, setSelectedPageId] = useState("guide");
  const query = searchQuery.trim().toLocaleLowerCase("vi");

  const visiblePages = useMemo(() => wikiPages.filter((page) => {
    if (!query) return true;
    return page.title.toLocaleLowerCase("vi").includes(query)
      || page.keywords.some((keyword) => keyword.toLocaleLowerCase("vi").includes(query));
  }), [query]);

  const currentPage = wikiPages.find((page) => page.id === selectedPageId) ?? wikiPages[0];
  const categories = Array.from(new Set(visiblePages.map((page) => page.category)));

  const selectPage = (id: string) => setSelectedPageId(id);

  return (
    <div className="w-full mx-auto bg-zinc-950/70 border border-zinc-900 rounded-3xl overflow-hidden shadow-2xl backdrop-blur-xl flex flex-col md:flex-row h-[680px]">
      <aside className="w-full md:w-72 border-r border-zinc-900 flex flex-col shrink-0 bg-zinc-950/80">
        <div className="p-4 border-b border-zinc-900/60">
          <div className="relative flex items-center">
            <Search className="w-4 h-4 text-zinc-600 absolute left-3 pointer-events-none" />
            <input
              type="search"
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="Tìm kiếm tài liệu..."
              className="w-full bg-zinc-900 border border-zinc-900 focus:border-zinc-700 rounded-xl pl-9 pr-3 py-2 text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none transition-all font-mono"
            />
          </div>
        </div>

        <nav aria-label="Các trang bách khoa" className="flex-grow overflow-y-auto p-3 space-y-4">
          {categories.map((category) => (
            <section key={category}>
              <h2 className="px-3 py-1 text-[10px] font-bold uppercase tracking-wider text-zinc-600 font-mono">
                {categoryLabels[category] ?? category.replaceAll("-", " ")}
              </h2>
              <div className="space-y-0.5">
                {visiblePages.filter((page) => page.category === category).map((page) => (
                  <button
                    key={page.id}
                    type="button"
                    onClick={() => selectPage(page.id)}
                    aria-current={selectedPageId === page.id ? "page" : undefined}
                    className={`w-full text-left px-3 py-2 rounded-lg text-xs font-semibold transition-all ${selectedPageId === page.id
                      ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                      : "text-zinc-400 hover:bg-zinc-900/60 hover:text-zinc-200"}`}
                  >
                    {page.title}
                  </button>
                ))}
              </div>
            </section>
          ))}
          {visiblePages.length === 0 && (
            <p className="px-3 py-4 text-xs text-zinc-500">Không tìm thấy trang phù hợp.</p>
          )}
        </nav>
      </aside>

      <main className="flex-grow flex flex-col min-w-0 bg-zinc-950/20 overflow-y-auto p-6 md:p-8">
        <div className="flex items-center gap-1.5 text-[10px] font-mono text-zinc-600 uppercase tracking-widest mb-2">
          <span>Bách Khoa GuWorld</span>
          <ChevronRight className="w-3 h-3" />
          <span>{categoryLabels[currentPage.category] ?? currentPage.category}</span>
          {currentPage.subCategory && (
            <>
              <ChevronRight className="w-3 h-3" />
              <span>{currentPage.subCategory.replaceAll("-", " ")}</span>
            </>
          )}
        </div>

        <h1 className="text-xl md:text-2xl font-bold font-serif text-zinc-100 border-b border-zinc-900/80 pb-4 mb-6">
          {currentPage.title}
        </h1>

        <div className="flex-grow space-y-6 text-zinc-300 font-serif leading-relaxed">
          {currentPage.content}
          {currentPage.cards && currentPage.cards.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-6">
              {currentPage.cards.map((card) => (
                <button
                  key={card.targetId}
                  type="button"
                  onClick={() => selectPage(card.targetId)}
                  className="p-4 text-left bg-zinc-950/80 border border-zinc-900 hover:border-violet-500/30 hover:bg-zinc-900/30 rounded-2xl group transition-all flex items-center justify-between"
                >
                  <span className="space-y-1">
                    <span className="block text-xs font-bold font-serif text-zinc-200 group-hover:text-indigo-400 transition-colors">
                      {card.title}
                    </span>
                    <span className="block text-[10px] text-zinc-500">{card.description}</span>
                  </span>
                  <ChevronRight className="w-4 h-4 text-zinc-700 group-hover:text-indigo-400 shrink-0" />
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="border-t border-zinc-900/60 pt-4 mt-8 flex items-center justify-between text-[10px] text-zinc-600 font-mono">
          <span className="flex items-center gap-1">
            <BookOpen className="w-3.5 h-3.5 text-zinc-700" />
            Nội dung hướng dẫn được dùng chung với trang wiki công khai.
          </span>
          <a href="/wiki" target="_blank" rel="noreferrer" className="text-indigo-400 hover:text-indigo-300">
            Mở bách khoa đầy đủ
          </a>
        </div>
      </main>
    </div>
  );
};
