import Link from "next/link";
import { ChevronRight } from "lucide-react";

export type AdminBreadcrumbItem = { label: string; href?: string };

export function AdminBreadcrumbs({ items }: { items: AdminBreadcrumbItem[] }) {
  return (
    <nav aria-label="Breadcrumb" className="flex flex-wrap items-center gap-1 text-xs text-zinc-500">
      {items.map((item, index) => {
        const current = index === items.length - 1;
        return (
          <span key={`${item.label}-${index}`} className="inline-flex items-center gap-1">
            {index > 0 && <ChevronRight aria-hidden="true" className="h-3 w-3 text-zinc-700" />}
            {item.href && !current ? <Link href={item.href} className="transition hover:text-amber-400">{item.label}</Link> : <span aria-current={current ? "page" : undefined} className={current ? "text-zinc-300" : undefined}>{item.label}</span>}
          </span>
        );
      })}
    </nav>
  );
}
