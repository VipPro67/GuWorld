import type { Metadata } from "next";
import { PremiumStoreView } from "@/components/PremiumStoreView";

export const metadata: Metadata = {
  title: "Nạp Nguyên Thạch",
  description: "Nạp Nguyên Thạch, Tiên Nguyên Thạch để tiếp tục hành trình tu tiên tại Cổ Giới. Hỗ trợ thanh toán qua PayOS.",
  robots: {
    index: false,
    follow: false,
  },
  openGraph: {
    title: "Nạp Nguyên Thạch | Cổ Giới",
    description: "Nạp Nguyên Thạch, Tiên Nguyên Thạch để tiếp tục hành trình tu tiên tại Cổ Giới.",
  },
};

export default function TopupPage() {
  return (
    <main className="min-h-screen bg-[var(--bg-primary)] px-2 sm:px-4 py-8 text-[var(--text-primary)]">
      <PremiumStoreView defaultTab="topup" />
    </main>
  );
}
