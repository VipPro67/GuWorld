import type { Metadata } from "next";
import { PremiumStoreView } from "@/components/PremiumStoreView";

export const metadata: Metadata = {
  title: "Nạp Nguyên Thạch Thành Công",
  description: "Giao dịch nạp Nguyên Thạch thành công. Cảm ơn đạo hữu đã ủng hộ Cổ Giới!",
  robots: {
    index: false,
    follow: false,
  },
};

export default function TopupSuccessPage() {
  return (
    <main className="min-h-screen bg-[var(--bg-primary)] px-4 py-8 text-[var(--text-primary)]">
      <PremiumStoreView defaultTab="topup" successMode />
    </main>
  );
}
