import { TopupView } from "@/components/TopupView";

export default function TopupSuccessPage() {
  return (
    <main className="min-h-screen bg-[#050505] px-4 py-8 text-zinc-100">
      <TopupView successMode />
    </main>
  );
}
