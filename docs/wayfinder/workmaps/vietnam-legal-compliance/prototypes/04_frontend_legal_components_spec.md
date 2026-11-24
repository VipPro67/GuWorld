# ĐẶC TẢ KIẾN TRÚC UI/UX TÍCH HỢP ĐIỀU KHOẢN, CHECKBOX CONSENT & CẢNH BÁO SỨC KHỎE

**Mục tiêu**: Định hình toàn diện các thành phần giao diện người dùng (React / Next.js), luồng tương tác (User Flow) và cơ chế cảnh báo thời gian chơi tuân thủ **Nghị định 147/2024/NĐ-CP** và **Nghị định 13/2023/NĐ-CP** trên GuWorld.

---

## 1. TỔNG THỂ CÁC THÀNH PHẦN GIAO DIỆN (COMPONENT HIERARCHY)

```
frontend/src/
├── components/
│   ├── Registration.tsx                 ← [Bổ sung] Checkbox Consent + Validation + Links mở modal
│   ├── legal/
│   │   ├── LegalModal.tsx               ← [Mới] Modal hiển thị toàn văn 3 văn bản pháp lý
│   │   ├── LegalFooter.tsx              ← [Mới] Chân trang chứa Cảnh báo 180 phút + Links pháp lý
│   │   └── PlaytimeWarningModal.tsx     ← [Mới] Modal cảnh báo khi đạt mốc 180 phút tu luyện
│   └── Profile/
│       └── AccountErasureSection.tsx    ← [Mới] Nút yêu cầu xóa tài khoản (Right to be Forgotten)
├── hooks/
│   └── usePlaytimeWarning.ts            ← [Mới] Hook quản lý đếm giờ phiên chơi & phát cảnh báo
└── app/
    └── page.tsx                         ← [Tích hợp] Gắn LegalFooter & hook usePlaytimeWarning
```

---

## 2. CHI TIẾT THIẾT KẾ CÁC THÀNH PHẦN

### 2.1. Cập Nhật Form Đăng Ký (`Registration.tsx`)

#### A. Trạng thái (State)
```typescript
const [agreeTerms, setAgreeTerms] = useState<boolean>(false); // Mặc định false (Opt-in)
const [activeLegalDoc, setActiveLegalDoc] = useState<'TOS' | 'PRIVACY' | 'PAYMENT' | null>(null);
```

#### B. Checkbox Chấp thuận (Tuân thủ Điều 9 & 11 NĐ 13/2023/NĐ-CP)
- **Vị trí**: Nằm ngay phía trên nút "Thức Tỉnh Không Khiếu" (Đăng ký), chỉ hiển thị khi `!isLoginTab`.
- **Yêu cầu kỹ thuật**:
  - Checkbox bắt buộc không được chọn sẵn (`defaultChecked={false}`).
  - Nhãn text chứa các link bấm mở modal:
    ```tsx
    <label className="flex items-start gap-2.5 text-xs text-zinc-400 cursor-pointer select-none mt-3">
      <input
        type="checkbox"
        id="agreeTermsCheckbox"
        checked={agreeTerms}
        onChange={(e) => setAgreeTerms(e.target.checked)}
        className="mt-0.5 w-4 h-4 rounded border-zinc-700 bg-zinc-900 text-amber-500 focus:ring-amber-500 cursor-pointer"
        required
      />
      <span>
        Tôi xác nhận từ <strong>đủ 18 tuổi trở lên</strong>, đã đọc kỹ và đồng ý với{" "}
        <button
          type="button"
          onClick={() => setActiveLegalDoc('TOS')}
          className="text-amber-400 hover:text-amber-300 underline font-semibold cursor-pointer"
        >
          Điều Khoản Dịch Vụ
        </button>{" "}
        và{" "}
        <button
          type="button"
          onClick={() => setActiveLegalDoc('PRIVACY')}
          className="text-amber-400 hover:text-amber-300 underline font-semibold cursor-pointer"
        >
          Chính Sách Bảo Mật
        </button>{" "}
        của GuWorld.
      </span>
    </label>
    ```
- **Điều kiện kích hoạt Nút Submit**:
  ```typescript
  const isSubmitDisabled = isLoading ||
    (isLoginTab
      ? email.trim().length === 0
      : username.trim().length < 3 || email.trim().length === 0 || !agreeTerms) ||
    password.length < 6;
  ```

---

### 2.2. Modal Hiển Thị Văn Bản Pháp Lý (`LegalModal.tsx`)

- **Đặc điểm**:
  - Không reload trang, giữ nguyên dữ liệu đang nhập trong form đăng ký.
  - Hỗ trợ thanh Tab chuyển đổi linh hoạt giữa 3 văn bản:
    1. **Điều Khoản Dịch Vụ** (Terms of Service - ToS)
    2. **Chính Sách Bảo Mật Dữ Liệu** (Privacy Policy - NĐ 13/2023)
    3. **Quy Chế Nạp Tiền & Bản Quyền** (Payment & IP Disclaimer)
  - Có thanh công cụ tìm kiếm từ khóa nhanh trong văn bản.
  - Nút "Tôi Đã Hiểu & Đóng" ở chân modal.

---

### 2.3. Chân Trang Pháp Lý & Cảnh Báo 180 Phút (`LegalFooter.tsx`)

Căn cứ **Điều 39 & Điều 40 Nghị định 147/2024/NĐ-CP**:
- **Nội dung cảnh báo bắt buộc**:
  *"Trò chơi điện tử dành cho người từ 18 tuổi trở lên (18+). Chơi quá 180 phút một ngày sẽ ảnh hưởng xấu đến sức khỏe."*
- **Thiết kế Footer**:
  - Nằm cố định ở chân màn hình Đăng nhập / Chọn nhân vật / Lobby chính.
  - Màu nền tối (`bg-zinc-950/80 backdrop-blur-sm border-t border-zinc-900`).
  - Gồm 3 khối:
    1. **Khối Cảnh Báo**: Biểu tượng cảnh báo vàng + Badge `18+` + Dòng chữ pháp định 180 phút.
    2. **Khối Links Văn Bản**: 3 liên kết mở Modal: `[Điều khoản]`, `[Chính sách bảo mật]`, `[Quy chế nạp tiền]`.
    3. **Khối Bản Quyền**: `GuWorld © 2024 — Fan-game Cổ Chân Nhân. Tuân thủ NĐ 147/2024 & NĐ 13/2023.`

---

### 2.4. Hook Quản Lý Thời Gian Chơi & Cảnh Báo Sức Khỏe (`usePlaytimeWarning.ts`)

- **Cơ chế hoạt động**:
  - Khi `player` đăng nhập thành công (`player !== null`), hook bắt đầu tính giây phiên chơi `sessionSeconds`.
  - Lưu mốc thời gian bắt đầu vào `sessionStorage` để không bị reset khi người chơi F5 / chuyển tab.
- **Các mốc thông báo (Milestones)**:
  - **Mốc 60 phút**: Gửi Toast Notification (Loại: `info`):  
    *“🔔 Đạo hữu đã tu luyện liên tục 60 phút. Hãy chớp mắt và uống ngụm nước để bảo vệ thị lực!”*
  - **Mốc 120 phút**: Gửi Toast Notification (Loại: `warning`):  
    *“⚠️ Đạo hữu đã nhập định 120 phút. Vận chuyển chân nguyên quá lâu có thể gây tổn hao tinh thần!”*
  - **Mốc 180 phút & mỗi 60 phút tiếp theo**: Bật Modal Pop-up Cảnh Báo Pháp Định (`PlaytimeWarningModal`):  
    *“🚨 CẢNH BÁO SỨC KHỎE (NGHỊ ĐỊNH 147/2024/NĐ-CP): Đạo hữu đã tu luyện liên tục 180 phút! Chơi quá 180 phút một ngày sẽ ảnh hưởng xấu đến sức khỏe. Xin vui lòng tạm ngưng xuất thế, nghỉ ngơi điều hòa khí tức.”*  
    Nút bấm: *"Ta đã hiểu, sẽ nghỉ ngơi sớm"* (Tắt modal và tiếp tục sau 60 phút nữa sẽ nhắc lại).

---

## 3. MA TRẬN LUỒNG NGƯỜI DÙNG (USER FLOW)

| Tình huống người dùng | Hành vi giao diện | Kết quả kỹ thuật |
|:---|:---|:---|
| Người dùng đăng ký mới nhưng chưa tick Checkbox | Nút "Thức Tỉnh Không Khiếu" bị mờ (disabled), hiển thị con trỏ `not-allowed`. | Form không gửi request `RegisterPlayer`. |
| Người dùng bấm vào chữ "[Điều Khoản Dịch Vụ]" | Mở `LegalModal` đè lên form đăng ký, form phía dưới giữ nguyên giá trị đã nhập. | Không mất dữ liệu email/username/password. |
| Người dùng đọc xong, đóng modal và tick Checkbox | Nút "Thức Tỉnh Không Khiếu" đổi sang màu hoàng kim sáng bóng, sẵn sàng bấm. | `agreeTerms === true`. |
| Gửi đăng ký thành công | Payload gửi sang Backend kèm `agree_terms: true`. | Backend ghi nhận consent audit timestamp. |
| Người chơi trong game đạt 180 phút | Xuất hiện `PlaytimeWarningModal` cảnh báo sức khỏe theo NĐ 147. | Tuân thủ 100% nghĩa vụ cảnh báo định kỳ. |
