"use client";

import React, { useState, useEffect } from "react";
import { X, Shield, FileText, AlertTriangle, Scale, Coins } from "lucide-react";

export type LegalDocType = "TOS" | "PRIVACY" | "PAYMENT";

interface LegalModalProps {
  isOpen: boolean;
  initialTab?: LegalDocType;
  onClose: () => void;
  onRequestDeleteAccount?: () => void;
}

export const LegalModal: React.FC<LegalModalProps> = ({
  isOpen,
  initialTab = "TOS",
  onClose,
  onRequestDeleteAccount,
}) => {
  const [activeTab, setActiveTab] = useState<LegalDocType>(initialTab);

  useEffect(() => {
    if (isOpen) {
      setActiveTab(initialTab);
    }
  }, [isOpen, initialTab]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-[120] flex items-center justify-center p-2 sm:p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200 overscroll-contain"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Văn Bản Pháp Lý & Quy Chế GuWorld"
        className="relative w-full max-w-3xl max-h-[90dvh] flex flex-col bg-[#111827] border border-amber-900/50 rounded-2xl shadow-2xl shadow-black/80 text-zinc-200 overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="shrink-0 flex items-center justify-between gap-2 px-4 py-3 sm:px-6 sm:py-4 border-b border-zinc-800 bg-zinc-950/60">
          <div className="flex items-center gap-2.5 min-w-0">
            <Scale className="w-5 h-5 text-amber-500 shrink-0" />
            <div className="min-w-0">
              <h2 className="text-sm sm:text-base font-serif font-bold text-amber-400 truncate">
                Văn Bản Pháp Lý & Quy Chế GuWorld
              </h2>
              <p className="text-[11px] text-zinc-400 truncate">
                Căn cứ Nghị định 147/2024/NĐ-CP & Nghị định 13/2023/NĐ-CP
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="shrink-0 p-1.5 text-zinc-400 hover:text-white rounded-lg hover:bg-zinc-800/80 transition-colors cursor-pointer"
            aria-label="Đóng"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tab Selection */}
        <div className="shrink-0 grid grid-cols-3 sm:flex border-b border-zinc-800 bg-zinc-900/40 px-2 sm:px-6 pt-1.5 sm:pt-2 gap-1 sm:gap-2">
          <button
            type="button"
            onClick={() => setActiveTab("TOS")}
            className={`py-2 px-1.5 sm:pb-3 sm:px-3 text-[11px] sm:text-xs font-semibold flex flex-col sm:flex-row items-center justify-center gap-1 sm:gap-1.5 border-b-2 transition-all text-center leading-tight cursor-pointer ${
              activeTab === "TOS"
                ? "border-amber-500 text-amber-400"
                : "border-transparent text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <FileText className="w-3.5 h-3.5 shrink-0" />
            <span>Điều Khoản Dịch Vụ</span>
          </button>

          <button
            type="button"
            onClick={() => setActiveTab("PRIVACY")}
            className={`py-2 px-1.5 sm:pb-3 sm:px-3 text-[11px] sm:text-xs font-semibold flex flex-col sm:flex-row items-center justify-center gap-1 sm:gap-1.5 border-b-2 transition-all text-center leading-tight cursor-pointer ${
              activeTab === "PRIVACY"
                ? "border-amber-500 text-amber-400"
                : "border-transparent text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <Shield className="w-3.5 h-3.5 shrink-0" />
            <span>Bảo Mật (NĐ 13)</span>
          </button>

          <button
            type="button"
            onClick={() => setActiveTab("PAYMENT")}
            className={`py-2 px-1.5 sm:pb-3 sm:px-3 text-[11px] sm:text-xs font-semibold flex flex-col sm:flex-row items-center justify-center gap-1 sm:gap-1.5 border-b-2 transition-all text-center leading-tight cursor-pointer ${
              activeTab === "PAYMENT"
                ? "border-amber-500 text-amber-400"
                : "border-transparent text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <Coins className="w-3.5 h-3.5 shrink-0" />
            <span>Nạp Tiền & Bản Quyền</span>
          </button>
        </div>

        {/* Content Body */}
        <div className="flex-1 min-h-0 overflow-y-auto overscroll-contain p-4 sm:p-6 text-xs sm:text-sm text-zinc-300 space-y-4 leading-relaxed font-sans">
          {activeTab === "TOS" && (
            <div className="space-y-4">
              <div className="p-3 bg-red-950/30 border-l-4 border-red-500 rounded-r-lg text-red-200 text-xs">
                <p className="font-bold flex items-center gap-1.5 text-red-400 mb-1">
                  <AlertTriangle className="w-4 h-4" />
                  CẢNH BÁO PHÁP ĐỊNH (NGHỊ ĐỊNH 147/2024/NĐ-CP):
                </p>
                <p>
                  1. Trò chơi có yếu tố chiến đấu PvP, phân loại dành cho người từ đủ <strong>18 tuổi trở lên (18+)</strong>.
                </p>
                <p>
                  2. <strong>Chơi quá 180 phút một ngày sẽ ảnh hưởng xấu đến sức khỏe.</strong>
                </p>
                <p>
                  3. Dịch vụ vận hành ở trạng thái <strong>Thử Nghiệm Alpha</strong> phi thương mại.
                </p>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">1. Quản Lý Tài Khoản & Tư Cách Tham Gia</h3>
                <p>
                  Người chơi cam kết từ đủ 18 tuổi trở lên và sử dụng địa chỉ Email hợp pháp của mình. Người chơi có nghĩa vụ tự bảo quản bí mật mật khẩu. Ban Quản Trị (BQT) không chịu trách nhiệm đối với tổn thất do người chơi tự ý chia sẻ tài khoản.
                </p>
              </div>

              <div className="p-3 bg-amber-950/20 border border-amber-800/40 rounded-lg">
                <h3 className="text-amber-400 font-bold text-sm mb-1">2. Bản Chất Phi Tiền Tệ & Cấm Giao Dịch Tiền Thật (Cấm RMT)</h3>
                <p className="mb-2">
                  Căn cứ Điều 53–57 Nghị định 147/2024/NĐ-CP: Mọi Cổ trùng, Ngọc, Tiên Thạch, Nguyên Thạch và vật phẩm ảo là dữ liệu số nội bộ trò chơi, <strong>TUYỆT ĐỐI KHÔNG CÓ GIÁ TRỊ QUY ĐỔI RA TIỀN MẶT (VND)</strong>, thẻ cào viễn thông hoặc tài sản ngoài đời thực. BQT không bao giờ mua lại tài sản ảo.
                </p>
                <p className="text-red-400 font-semibold">
                  Nghiêm cấm tuyệt đối hành vi mua bán tài khoản, vật phẩm bằng tiền thật bên ngoài (RMT). Mọi giao dịch chỉ được phép diễn ra qua Chợ Đấu Giá hoặc Giao Dịch P2P in-game bằng Tiên Thạch. Tài khoản vi phạm sẽ bị KHÓA VĨNH VIỄN ngay lập tức.
                </p>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">3. Hành Vi Bị Nghiêm Cấm (Luật An Ninh Mạng 2018)</h3>
                <ul className="list-disc pl-5 space-y-1 text-zinc-400">
                  <li>Nghiêm cấm tuyên truyền chống phá Nhà nước, xúc phạm danh dự người khác, kích động bạo lực, cờ bạc trên kênh Chat và tên nhân vật.</li>
                  <li>Nghiêm cấm sử dụng phần mềm thứ ba can thiệp (cheat, bot cày tự động không phép, can thiệp bộ nhớ hoặc gói tin mạng).</li>
                  <li>Nghiêm cấm cố tình lợi dụng lỗi (Bug exploit) để trục lợi.</li>
                </ul>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">4. Chế Tài Xử Lý & Bảo Lưu Dữ Liệu</h3>
                <p>
                  BQT có quyền áp dụng các mức phạt: Cảnh cáo & thu hồi vật phẩm lỗi, Khóa chat từ 24h đến vĩnh viễn, Khóa tài khoản tạm thời (3–30 ngày), hoặc Khóa vĩnh viễn không bồi hoàn chi phí.
                </p>
                <p className="mt-1 text-zinc-400">
                  Khi có đợt cập nhật lớn cần Reset dữ liệu Alpha (Wipe), BQT sẽ thông báo trước 07 ngày và <strong>bảo lưu 100% số Ngọc</strong> người chơi đã ủng hộ qua PayOS.
                </p>
              </div>
            </div>
          )}

          {activeTab === "PRIVACY" && (
            <div className="space-y-4">
              <div className="p-3 bg-cyan-950/30 border-l-4 border-cyan-500 rounded-r-lg text-cyan-200 text-xs">
                <p className="font-bold text-cyan-400 mb-1">
                  TUÂN THỦ NGHỊ ĐỊNH 13/2023/NĐ-CP VỀ BẢO VỆ DỮ LIỆU CÁ NHÂN:
                </p>
                <p>
                  GuWorld cam kết bảo vệ quyền riêng tư và chỉ thu thập dữ liệu cá nhân cơ bản tối thiểu cần thiết để vận hành trò chơi.
                </p>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">1. Dữ Liệu Thu Thập & Mục Đích Xử Lý</h3>
                <ul className="list-disc pl-5 space-y-1 text-zinc-400">
                  <li><strong>Dữ liệu cơ bản</strong>: Tên tài khoản (Đạo hiệu), Email, Mật khẩu (mã hóa một chiều an toàn bằng Bcrypt có Salt), Địa chỉ IP, Lịch sử nạp PayOS.</li>
                  <li><strong>Mục đích</strong>: Xác thực đăng nhập, lưu trữ tiến trình tu luyện, bảo vệ an ninh mạng và đối soát nạp Ngọc tự động.</li>
                  <li><strong>Tuyên bố không thu thập dữ liệu nhạy cảm</strong>: KHÔNG lưu CCCD, thông tin tài khoản ngân hàng, sinh trắc học hay định vị GPS vệ tinh.</li>
                </ul>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">2. Chia Sẻ Bên Thứ Ba (PayOS)</h3>
                <p>
                  Chúng tôi không bao giờ bán dữ liệu cho bên thứ ba vì mục đích quảng cáo. Khi nạp tiền, hệ thống chỉ truyền mã giao dịch và số tiền VND sang cổng PayOS để tạo mã VietQR; tuyệt đối không chia sẻ email hay mật khẩu.
                </p>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">3. Quyền Của Người Chơi & Xóa Dữ Liệu (Right to be Forgotten)</h3>
                <p>
                  Căn cứ Điều 9 NĐ 13/2023/NĐ-CP, người chơi có đầy đủ 11 quyền đối với dữ liệu cá nhân của mình, bao gồm quyền yêu cầu xóa tài khoản / dữ liệu cá nhân.
                </p>
                <p className="mt-1 text-zinc-400">
                  Người chơi có thể thực hiện yêu cầu xóa tài khoản trực tiếp bên dưới hoặc gửi yêu cầu qua email <code>privacy@guworld.local</code>. BQT cam kết thực hiện vô danh hóa dữ liệu trong tối đa 72 giờ làm việc.
                </p>
                {onRequestDeleteAccount && (
                  <div className="mt-3 pt-3 border-t border-zinc-800">
                    <button
                      type="button"
                      onClick={() => {
                        onClose();
                        onRequestDeleteAccount();
                      }}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-red-950/40 hover:bg-red-900/60 border border-red-800/60 text-red-400 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
                    >
                      <AlertTriangle className="w-3.5 h-3.5" />
                      Yêu Cầu Xóa Tài Khoản & Dữ Liệu Cá Nhân
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTab === "PAYMENT" && (
            <div className="space-y-4">
              <div className="p-3 bg-amber-950/30 border-l-4 border-amber-500 rounded-r-lg text-amber-200 text-xs">
                <p className="font-bold text-amber-400 mb-1">
                  ĐÓNG GÓP DUY TRÌ MÁY CHỦ (SERVER UPKEEP DONATION):
                </p>
                <p>
                  Các gói nạp qua PayOS (10.000 VND đến 200.000 VND) là sự ủng hộ tài chính tự nguyện của người chơi nhằm duy trì chi phí máy chủ và băng thông.
                </p>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">1. Chính Sách Không Hoàn Tiền (Non-refundable)</h3>
                <p>
                  Do số Ngọc (Jade) được cộng tức thì vào tài khoản sau khi chuyển khoản thành công và có thể tiêu thụ ngay trong game, mọi khoản đóng góp đều <strong>không được hoàn lại tiền mặt</strong> dưới bất kỳ lý do nào.
                </p>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">2. Cam Kết Bảo Lưu Ngọc Khi Wipe Server</h3>
                <p>
                  Trong trường hợp cập nhật cấu trúc hệ thống lớn ở giai đoạn thử nghiệm Alpha bắt buộc phải Reset dữ liệu, BQT cam kết <strong>bảo lưu 100% số lượng Ngọc</strong> người chơi đã nạp để hoàn trả vào tài khoản phiên bản mới.
                </p>
              </div>

              <div>
                <h3 className="text-amber-400 font-bold text-sm mb-1">3. Tuyên Bố Bản Quyền Tác Phẩm Phái Sinh</h3>
                <p>
                  Thế giới quan Cổ Sư, Cổ Tiên, Đạo Ngân và hệ thống Cổ trùng trong game được lấy cảm hứng từ tiểu thuyết tiên hiệp <strong>"Cổ Chân Nhân"</strong> của tác giả <strong>Cổ Chân Nhân</strong>.
                </p>
                <p className="mt-1 text-zinc-400">
                  GuWorld là sản phẩm sáng tạo độc lập của người hâm mộ (Fan-game) phi thương mại phục vụ mục đích nghiên cứu công nghệ; không mạo danh tác giả gốc, không cạnh tranh thương mại và hoàn toàn tôn trọng quyền tác giả tiểu thuyết gốc.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="shrink-0 flex items-center justify-between gap-3 px-4 py-3 sm:px-6 sm:py-3.5 border-t border-zinc-800 bg-zinc-950/80">
          <span className="text-[11px] text-zinc-400">
            Phiên bản: 2024.1 — Hiệu lực từ 25/12/2024
          </span>
          <button
            type="button"
            onClick={onClose}
            className="shrink-0 whitespace-nowrap px-4 py-2 sm:px-5 rounded-xl bg-gradient-to-r from-amber-600 to-yellow-600 text-zinc-950 font-serif font-bold text-xs hover:from-amber-500 hover:to-yellow-500 transition-all cursor-pointer shadow-md"
          >
            Ta Đã Hiểu & Đóng
          </button>
        </div>
      </div>
    </div>
  );
};
