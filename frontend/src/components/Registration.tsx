"use client";

import React, { useState } from "react";
import { Sparkles, Compass, ShieldAlert, KeyRound, User, Eye, EyeOff, Mail, Phone } from "lucide-react";

interface RegistrationProps {
  onRegister: (username: string, password: string, email: string, phoneNumber: string, startingRegion: string, setVitalNow: boolean) => Promise<boolean>;
  onLogin: (username: string, password: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
  onClearError: () => void;
}

export const Registration: React.FC<RegistrationProps> = ({
  onRegister,
  onLogin,
  isLoading,
  error,
  onClearError,
}) => {
  const [isLoginTab, setIsLoginTab] = useState(true);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [email, setEmail] = useState("");
  const [phoneNumber, setPhoneNumber] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [startingRegion, setStartingRegion] = useState("Southern_Border");
  const [setVitalNow, setSetVitalNow] = useState(true);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || password.length < 6) return;

    if (isLoginTab) {
      await onLogin(username.trim(), password);
    } else {
      await onRegister(username.trim(), password, email.trim(), phoneNumber.trim(), startingRegion, setVitalNow);
    }
  };

  const handleTabChange = (login: boolean) => {
    setIsLoginTab(login);
    onClearError();
    setPassword("");
    setEmail("");
    setPhoneNumber("");
  };

  return (
    <div className="glass-panel-gold rounded-3xl p-8 w-full max-w-md relative overflow-hidden group shadow-[0_0_50px_rgba(234,179,8,0.15)]">
      {/* Decorative floating aura */}
      <div className="absolute -left-12 -bottom-12 w-28 h-28 rounded-full bg-emerald-500/10 blur-2xl pointer-events-none" />
      <div className="absolute -right-12 -top-12 w-28 h-28 rounded-full bg-amber-500/10 blur-2xl pointer-events-none" />

      {/* Header */}
      <div className="text-center mb-6">
        <div className="inline-flex p-3 rounded-2xl bg-amber-950/40 border border-amber-500/20 text-amber-400 mb-4 animate-pulse">
          <Compass className="w-6 h-6" />
        </div>
        <h2 className="text-3xl font-bold text-zinc-100 font-serif tracking-wide">
          CỔ THẾ GIỚI
        </h2>
        <p className="text-xs text-amber-500 uppercase tracking-widest font-mono mt-1">
          {isLoginTab ? "Đại Điện Tu Hành" : "Khai mở Bản Mệnh"}
        </p>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-zinc-900 mb-6 bg-zinc-950/60 p-1.5 rounded-2xl border border-zinc-900/60">
        <button
          type="button"
          onClick={() => handleTabChange(true)}
          className={`flex-1 py-2.5 text-xs font-mono uppercase tracking-wider rounded-xl transition-all font-semibold cursor-pointer ${
            isLoginTab
              ? "bg-amber-500 text-zinc-950 shadow-md font-bold"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          Đăng Nhập
        </button>
        <button
          type="button"
          onClick={() => handleTabChange(false)}
          className={`flex-1 py-2.5 text-xs font-mono uppercase tracking-wider rounded-xl transition-all font-semibold cursor-pointer ${
            !isLoginTab
              ? "bg-amber-500 text-zinc-950 shadow-md font-bold"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          Đăng Ký
        </button>
      </div>

      {/* Text Description */}
      <p className="text-xs text-zinc-400 leading-relaxed text-center mb-6 font-serif h-12 flex items-center justify-center">
        {isLoginTab
          ? "Đã có Tiên cốt Không Khiếu? Nhập Đạo hiệu cùng Mật mật mã để tiếp tục nghịch thiên hành trình."
          : '"Con người là vạn vật chi linh, Cổ là thiên địa chi tinh." Khai mở Đạo hiệu để gacha Tư chất (Giáp, Ất, Bính, Đinh).'}
      </p>

      {/* Form */}
      <form onSubmit={handleSubmit} className="space-y-4">
        {!isLoginTab && (
          <>
            <div>
              <label className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5">
                Khu Vực Khởi Đầu
              </label>
              <select
                value={startingRegion}
                onChange={(e) => setStartingRegion(e.target.value)}
                className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-3 px-4 text-sm text-zinc-100"
              >
                <option value="Southern_Border">Nam Cương</option>
                <option value="Central_Continent">Trung Châu</option>
                <option value="Northern_Plains">Bắc Nguyên</option>
                <option value="Western_Desert">Tây Mạc</option>
                <option value="Eastern_Sea">Đông Hải</option>
              </select>
            </div>
            <label className="flex items-center gap-2 text-xs text-zinc-300">
              <input
                type="checkbox"
                checked={setVitalNow}
                onChange={(e) => setSetVitalNow(e.target.checked)}
              />
              Đặt Cổ đầu tiên làm Bản Mệnh ngay (bỏ chọn để trì hoãn)
            </label>
          </>
        )}
        {/* Username */}
        <div className="relative">
          <label
            htmlFor="username"
            className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
          >
            {isLoginTab ? "Đạo Hiệu / Email / Số điện thoại" : "Đạo Hiệu (Character Name)"}
          </label>
          <div className="relative">
            <User className="absolute left-3.5 top-3.5 w-4 h-4 text-zinc-600" />
            <input
              type="text"
              id="username"
              disabled={isLoading}
              value={username}
              onChange={(e) => {
                onClearError();
                setUsername(e.target.value);
              }}
              placeholder={isLoginTab ? "Nhập đạo hiệu, email hoặc SĐT..." : "Nhập đạo hiệu..."}
              className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-3 pl-10 pr-4 text-sm text-zinc-100 placeholder-zinc-700 focus:outline-none focus:border-amber-500/50 focus:shadow-[0_0_15px_rgba(234,179,8,0.1)] transition-all font-serif"
              maxLength={50}
              required
            />
          </div>
        </div>

        {/* Email - Only in registration */}
        {!isLoginTab && (
          <div className="relative">
            <label
              htmlFor="email"
              className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
            >
              Email (Thư điện tử)
            </label>
            <div className="relative">
              <Mail className="absolute left-3.5 top-3.5 w-4 h-4 text-zinc-600" />
              <input
                type="email"
                id="email"
                disabled={isLoading}
                value={email}
                onChange={(e) => {
                  onClearError();
                  setEmail(e.target.value);
                }}
                placeholder="example@gmail.com"
                className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-3 pl-10 pr-4 text-sm text-zinc-100 placeholder-zinc-700 focus:outline-none focus:border-amber-500/50 focus:shadow-[0_0_15px_rgba(234,179,8,0.1)] transition-all font-mono"
                required
              />
            </div>
          </div>
        )}

        {/* Phone Number - Only in registration */}
        {!isLoginTab && (
          <div className="relative">
            <label
              htmlFor="phoneNumber"
              className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
            >
              Số điện thoại (Phone Number)
            </label>
            <div className="relative">
              <Phone className="absolute left-3.5 top-3.5 w-4 h-4 text-zinc-600" />
              <input
                type="tel"
                id="phoneNumber"
                disabled={isLoading}
                value={phoneNumber}
                onChange={(e) => {
                  onClearError();
                  setPhoneNumber(e.target.value);
                }}
                placeholder="0912345678"
                className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-3 pl-10 pr-4 text-sm text-zinc-100 placeholder-zinc-700 focus:outline-none focus:border-amber-500/50 focus:shadow-[0_0_15px_rgba(234,179,8,0.1)] transition-all font-mono"
                required
              />
            </div>
          </div>
        )}

        {/* Password */}
        <div className="relative">
          <label
            htmlFor="password"
            className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
          >
            Mật Mã (Password - Min 6 chars)
          </label>
          <div className="relative">
            <KeyRound className="absolute left-3.5 top-3.5 w-4 h-4 text-zinc-600" />
            <input
              type={showPassword ? "text" : "password"}
              id="password"
              disabled={isLoading}
              value={password}
              onChange={(e) => {
                onClearError();
                setPassword(e.target.value);
              }}
              placeholder="••••••••"
              className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-3 pl-10 pr-10 text-sm text-zinc-100 placeholder-zinc-700 focus:outline-none focus:border-amber-500/50 focus:shadow-[0_0_15px_rgba(234,179,8,0.1)] transition-all font-mono"
              minLength={6}
              maxLength={50}
              required
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-3.5 text-zinc-600 hover:text-zinc-400 focus:outline-none cursor-pointer"
            >
              {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>
        </div>

        {/* Error Display */}
        {error && (
          <div className="flex items-start gap-2 text-rose-400 bg-rose-950/20 border border-rose-900/40 p-3 rounded-xl text-xs mt-2">
            <ShieldAlert className="w-4.5 h-4.5 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        {/* Submit Button */}
        <button
          type="submit"
          disabled={
            isLoading ||
            username.trim().length < 3 ||
            password.length < 6 ||
            (!isLoginTab && (email.trim().length === 0 || phoneNumber.trim().length === 0))
          }
          className={`w-full py-4 px-6 rounded-xl font-serif text-sm font-semibold tracking-wider transition-all duration-300 flex items-center justify-center gap-2 cursor-pointer shadow-lg mt-6 ${
            isLoading ||
            username.trim().length < 3 ||
            password.length < 6 ||
            (!isLoginTab && (email.trim().length === 0 || phoneNumber.trim().length === 0))
              ? "bg-zinc-900 text-zinc-600 border border-zinc-850 cursor-not-allowed"
              : "bg-gradient-to-r from-amber-600 to-yellow-600 text-zinc-950 font-bold hover:from-amber-500 hover:to-yellow-500 hover:shadow-[0_0_25px_rgba(234,179,8,0.25)] active:scale-[0.98]"
          }`}
        >
          {isLoading ? (
            <>
              <div className="w-4 h-4 border-2 border-zinc-950 border-t-transparent rounded-full animate-spin" />
              {isLoginTab ? "Đang định danh..." : "Đang thức tỉnh Không Khiếu..."}
            </>
          ) : (
            <>
              <Sparkles className="w-4 h-4" />
              {isLoginTab ? "Thâm Nhập Cổ Giới" : "Thức Tỉnh Không Khiếu"}
            </>
          )}
        </button>
      </form>
    </div>
  );
};
