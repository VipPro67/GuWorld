"use client";

import React, { useState, useEffect } from "react";
import { Sparkles, Compass, ShieldAlert, KeyRound, User, Eye, EyeOff, Mail } from "lucide-react";

interface RegistrationProps {
  onRegister: (username: string, password: string, email: string, startingRegion: string, setVitalNow: boolean, referralCode?: string) => Promise<boolean>;
  onLogin: (email: string, password: string) => Promise<boolean>;
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
  const [showPassword, setShowPassword] = useState(false);
  const [startingRegion, setStartingRegion] = useState("Southern_Border");
  const [setVitalNow, setSetVitalNow] = useState(true);
  const [referralCode, setReferralCode] = useState("");

  useEffect(() => {
    if (typeof window !== "undefined") {
      const params = new URLSearchParams(window.location.search);
      const ref = params.get("ref");
      if (ref) {
        setReferralCode(ref);
      }
    }
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const usernameValue = username.trim();
    const emailValue = email.trim();

    if (isLoginTab) {
      if (!emailValue || password.length < 6) return;
      await onLogin(emailValue, password);
    } else {
      if (!usernameValue || !emailValue || password.length < 6) return;
      await onRegister(usernameValue, password, emailValue, startingRegion, setVitalNow, referralCode);
    }
  };

  const handleTabChange = (login: boolean) => {
    setIsLoginTab(login);
    onClearError();
    setPassword("");
    setEmail("");
    setUsername("");
  };

  const handleGoogleLogin = () => {
    const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
    const queryParams = referralCode ? `?ref=${encodeURIComponent(referralCode)}` : "";
    window.location.href = `${backendUrl}/api/auth/google/login${queryParams}`;
  };

  return (
    <div className="glass-panel-gold rounded-3xl p-8 w-full max-w-md relative overflow-hidden group shadow-[0_0_50px_rgba(234,179,8,0.15)]">
      {referralCode && (
        <div className="mb-4 text-[10px] font-mono text-center text-amber-400 bg-amber-950/40 border border-amber-500/30 p-2 rounded-xl animate-pulse">
          ⚡ LIÊN KẾT SƯ ĐỒ ĐÃ KÍCH HOẠT: <span className="font-bold underline text-yellow-300">{referralCode}</span>
        </div>
      )}
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
          ? "Đã có Tiên cốt Không Khiếu? Nhập Email cùng Mật mã để tiếp tục nghịch thiên hành trình."
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
        {/* Login email */}
        {isLoginTab && (
          <div className="relative">
            <label
              htmlFor="login-email"
              className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
            >
              Email
            </label>
            <div className="relative">
              <Mail className="absolute left-3.5 top-3.5 w-4 h-4 text-zinc-600" />
              <input
                type="email"
                id="login-email"
                disabled={isLoading}
                value={email}
                onChange={(e) => {
                  onClearError();
                  setEmail(e.target.value);
                }}
                placeholder="example@gmail.com"
                className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-3 pl-10 pr-4 text-sm text-zinc-100 placeholder-zinc-700 focus:outline-none focus:border-amber-500/50 focus:shadow-[0_0_15px_rgba(234,179,8,0.1)] transition-all font-mono"
                maxLength={255}
                required
              />
            </div>
          </div>
        )}

        {/* Username - Only in registration */}
        {!isLoginTab && (
          <div className="relative">
            <label
              htmlFor="username"
              className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
            >
              Đạo Hiệu
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
                placeholder="Nhập đạo hiệu..."
                className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-3 pl-10 pr-4 text-sm text-zinc-100 placeholder-zinc-700 focus:outline-none focus:border-amber-500/50 focus:shadow-[0_0_15px_rgba(234,179,8,0.1)] transition-all font-serif"
                maxLength={50}
                required
              />
            </div>
          </div>
        )}

        {/* Email - Only in registration */}
        {!isLoginTab && (
          <div className="relative">
            <label
              htmlFor="email"
              className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
            >
              Email
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

        {/* Password */}
        <div className="relative">
          <label
            htmlFor="password"
            className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5"
          >
            Mật Mã (Tối thiểu 6 ký tự)
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
            (isLoginTab ? email.trim().length === 0 : username.trim().length < 3 || email.trim().length === 0) ||
            password.length < 6
          }
          className={`w-full py-4 px-6 rounded-xl font-serif text-sm font-semibold tracking-wider transition-all duration-300 flex items-center justify-center gap-2 cursor-pointer shadow-lg mt-6 ${
            isLoading ||
            (isLoginTab ? email.trim().length === 0 : username.trim().length < 3 || email.trim().length === 0) ||
            password.length < 6
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

        {/* Google OAuth Login Button */}
        <div className="relative flex py-2 items-center">
          <div className="flex-grow border-t border-zinc-900"></div>
          <span className="flex-shrink mx-4 text-[10px] text-zinc-550 font-mono uppercase tracking-widest">Hoặc</span>
          <div className="flex-grow border-t border-zinc-900"></div>
        </div>

        <button
          type="button"
          onClick={handleGoogleLogin}
          className="w-full py-3.5 px-4 rounded-xl border border-zinc-850 hover:border-zinc-750 bg-zinc-950/60 text-zinc-300 text-xs font-semibold tracking-wider font-mono hover:text-white transition-all flex items-center justify-center gap-2.5 hover:shadow-[0_0_15px_rgba(234,179,8,0.08)] active:scale-[0.98] cursor-pointer"
        >
          <svg className="w-4.5 h-4.5 shrink-0" viewBox="0 0 24 24" fill="currentColor">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
            <path d="M5.84 14.09c-.22-.66-.35-1.38-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z" fill="#FBBC05"/>
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
          </svg>
          {isLoginTab ? "Đăng nhập bằng Google" : "Đăng ký bằng Google"}
        </button>
      </form>
    </div>
  );
};
