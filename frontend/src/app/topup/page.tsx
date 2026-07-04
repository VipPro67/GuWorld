import type { Metadata } from "next";
import { TopupView } from "@/components/TopupView";

export const metadata: Metadata = {
  title: "Nạp Nguyên Thạch",
  description: "Nạp Nguyên Thạch, Tiên Nguyên Thạch để tiếp tục hành trình tu tiên tại Cổ Thế Giới. Hỗ trợ thanh toán qua PayOS.",
  robots: {
    index: false,
    follow: false,
  },
  openGraph: {
    title: "Nạp Nguyên Thạch | Cổ Thế Giới",
    description: "Nạp Nguyên Thạch, Tiên Nguyên Thạch để tiếp tục hành trình tu tiên tại Cổ Thế Giới.",
  },
};

export default function TopupPage() {
  return (
    <main className="min-h-screen bg-[var(--bg-primary)] px-4 py-8 text-[var(--text-primary)]">
      <TopupView />
    </main>
  );
}
