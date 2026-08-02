import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { ThemeProvider } from "@/components/ThemeProvider";
import { Analytics } from "@vercel/analytics/next"
const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const siteUrl = process.env.NEXT_PUBLIC_BACKEND_URL
  ? process.env.NEXT_PUBLIC_BACKEND_URL.replace("://", "://www.").replace(/:\d+$/, "")
  : "https://www.cogioi.io.vn";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: "Cổ Giới - Game Cổ Chân Nhân Online",
  description: "Khai mở Không Khiếu, thâm nhập Thế Giới, tích lũy Nguyên Thạch, luyện hóa Cổ Trùng. Trò chơi nhập vai tu tiên mô phỏng theo tiểu thuyết Cổ Chân Nhân.",
  keywords: ["Cổ Giới", "Cổ Chân Nhân", "web game", "tu tiên", "nhập vai", "Cổ Trùng", "Nguyên Thạch", "Thanh Mao Sơn"],
  authors: [{ name: "Cổ Giới Development" }],
  applicationName: "Cổ Giới",
  generator: "Next.js",
  referrer: "origin-when-cross-origin",
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
  openGraph: {
    type: "website",
    locale: "vi_VN",
    siteName: "Cổ Giới",
    title: "Cổ Giới",
    description: "Khai mở Không Khiếu, thâm nhập Thanh Mao Sơn, tích lũy Nguyên Thạch, luyện hóa Cổ Trùng. Trò chơi nhập vai tu tiên mô phỏng theo tiểu thuyết Cổ Chân Nhân.",
    url: "/",
  },
  twitter: {
    card: "summary",
    title: "Cổ Giới",
    description: "Khai mở Không Khiếu, thâm nhập Thanh Mao Sơn, tích lũy Nguyên Thạch, luyện hóa Cổ Trùng.",
  },
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "256x256", type: "image/x-icon" },
    ],
    apple: [
      { url: "/favicon.ico", sizes: "256x256" },
    ],
  },
  manifest: "/manifest.json",
  verification: {
    google: "",
  },
  alternates: {
    canonical: "/",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="vi"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      suppressHydrationWarning
    >
      <body className="min-h-full flex flex-col">
        <Analytics />
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false}>
          {children}
        </ThemeProvider>
      </body>
    </html>
  );
}
