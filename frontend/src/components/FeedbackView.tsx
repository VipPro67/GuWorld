import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import {
  ThumbsUp,
  ThumbsDown,
  Sparkles,
  MessageSquare,
  Clock,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Award,
  Shield,
  Send,
  Loader2,
  HelpCircle,
} from "lucide-react";

export const FeedbackView: React.FC = () => {
  const {
    player,
    feedbacks,
    fetchFeedbacks,
    submitFeedback,
    voteFeedback,
    updateFeedbackStatus,
    actionLoading,
    setView,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"list" | "create">("list");
  const [filterCategory, setFilterCategory] = useState<string>("");
  const [filterStatus, setFilterStatus] = useState<string>("");

  // Create feedback form state
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [category, setCategory] = useState("BUG");

  // Admin moderation form state
  const [modifyingFeedbackId, setModifyingFeedbackId] = useState<string | null>(null);
  const [adminStatus, setAdminStatus] = useState("");
  const [adminResponse, setAdminResponse] = useState("");

  useEffect(() => {
    fetchFeedbacks(filterCategory, filterStatus);
  }, [filterCategory, filterStatus, fetchFeedbacks]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !description.trim()) return;
    const success = await submitFeedback(title.trim(), description.trim(), category);
    if (success) {
      setTitle("");
      setDescription("");
      setCategory("BUG");
      setActiveTab("list");
    }
  };

  const handleAdminUpdate = async (feedbackId: string) => {
    if (!adminStatus) return;
    const success = await updateFeedbackStatus(feedbackId, adminStatus, adminResponse);
    if (success) {
      setModifyingFeedbackId(null);
      setAdminStatus("");
      setAdminResponse("");
    }
  };

  const categories = [
    { value: "", label: "Tất cả hạng mục" },
    { value: "BUG", label: "Lỗi game" },
    { value: "GU", label: "Thêm cổ trùng" },
    { value: "ITEM", label: "Vật phẩm mới" },
    { value: "RECIPE", label: "Công thức luyện cổ" },
    { value: "BALANCE", label: "Cân bằng sức mạnh" },
    { value: "FEATURE", label: "Tính năng mới" },
    { value: "OTHER", label: "Đóng góp khác" },
  ];

  const statuses = [
    { value: "", label: "Tất cả trạng thái" },
    { value: "pending", label: "Đợi kiểm duyệt" },
    { value: "active", label: "Đang bình chọn" },
    { value: "in_progress", label: "Thiên Đạo ghi nhận" },
    { value: "completed", label: "Đã hoàn thành" },
    { value: "rejected", label: "Không khả thi" },
  ];

  const getCategoryLabel = (cat: string) => {
    switch (cat.toUpperCase()) {
      case "BUG":
        return "Lỗi game";
      case "GU":
        return "Thêm cổ trùng";
      case "ITEM":
        return "Vật phẩm";
      case "RECIPE":
        return "Công thức";
      case "BALANCE":
        return "Cân bằng";
      case "FEATURE":
        return "Tính năng";
      default:
        return "Khác";
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status.toLowerCase()) {
      case "pending":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[10px] font-semibold bg-zinc-900 text-zinc-400 border border-zinc-800">
            <Clock className="w-3.5 h-3.5" />
            Chờ Duyệt
          </span>
        );
      case "active":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[10px] font-semibold bg-cyan-950/60 text-cyan-400 border border-cyan-800/40">
            <Sparkles className="w-3.5 h-3.5" />
            Đang Bình Chọn
          </span>
        );
      case "in_progress":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[10px] font-semibold bg-amber-950/60 text-amber-400 border border-amber-800/40 animate-pulse">
            <AlertCircle className="w-3.5 h-3.5" />
            Đang Thực Hiện
          </span>
        );
      case "completed":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[10px] font-semibold bg-emerald-950/60 text-emerald-400 border border-emerald-800/40">
            <CheckCircle2 className="w-3.5 h-3.5" />
            Đã Hoàn Thành
          </span>
        );
      case "rejected":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-[10px] font-semibold bg-rose-950/60 text-rose-400 border border-rose-800/40">
            <XCircle className="w-3.5 h-3.5" />
            Bị Từ Chối
          </span>
        );
      default:
        return null;
    }
  };

  return (
    <div className="w-full max-w-4xl mx-auto flex flex-col gap-6 py-2 px-1">
      {/* 1. Header & Overview banner */}
      <div className="bg-zinc-950/80 border border-zinc-900 rounded-3xl p-6 relative overflow-hidden backdrop-blur-md">
        <div className="absolute top-0 right-0 w-64 h-64 bg-fuchsia-500/5 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute bottom-0 left-0 w-48 h-48 bg-emerald-500/5 rounded-full blur-2xl pointer-events-none" />

        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 relative z-10">
          <div>
            <h2 className="text-xl sm:text-2xl font-serif font-bold text-zinc-100 flex items-center gap-2">
              <Sparkles className="w-6 h-6 text-fuchsia-400 shrink-0" />
              Thỉnh Thiên Đạo
            </h2>
            <p className="text-xs text-zinc-500 mt-1 font-sans">
              Dâng sớ trình lỗi, hiến tế cổ trùng, kiến nghị cân bằng tu vi lên Thiên Đạo.
            </p>
          </div>

          <div className="flex gap-2 shrink-0">
            <button
              onClick={() => {
                setActiveTab("list");
                setFilterCategory("");
                setFilterStatus("");
              }}
              className={`px-4 py-2 rounded-xl text-xs font-semibold font-serif border transition-all cursor-pointer ${
                activeTab === "list"
                  ? "bg-zinc-900 border-zinc-800 text-zinc-100"
                  : "bg-transparent border-transparent text-zinc-500 hover:text-zinc-300"
              }`}
            >
              Xem Sớ Trình
            </button>
            <button
              onClick={() => setActiveTab("create")}
              className={`px-4 py-2 rounded-xl text-xs font-semibold font-serif border transition-all cursor-pointer flex items-center gap-1.5 ${
                activeTab === "create"
                  ? "bg-gradient-to-r from-fuchsia-600 to-pink-600 text-white border-transparent"
                  : "bg-zinc-950 border-zinc-900 text-zinc-400 hover:text-zinc-200"
              }`}
            >
              <span>+</span>
              <span>Dâng Sớ Mới</span>
            </button>
          </div>
        </div>

        {/* Reward announcement */}
        <div className="mt-5 p-3 rounded-2xl bg-fuchsia-950/20 border border-fuchsia-900/30 flex items-start gap-3">
          <Award className="w-5 h-5 text-fuchsia-400 shrink-0 mt-0.5" />
          <div className="text-[11px] leading-relaxed text-fuchsia-300/90 font-mono">
            <span className="font-bold text-fuchsia-200">Bổng lộc Thiên Đạo:</span> Mọi sớ trình được phê duyệt triển khai (
            <span className="text-amber-300">Đang thực hiện</span> hoặc <span className="text-emerald-300">Đã hoàn thành</span>)
            sẽ nhận được bổng lộc trực tiếp từ Bảo Hoàng Thiên: <span className="font-bold text-yellow-500">500 Nguyên Thạch</span> cho Cổ sư phàm nhân hoặc <span className="font-bold text-purple-400">5 Tiên Nguyên Thạch</span> cho Cổ Tiên!
          </div>
        </div>
      </div>

      {activeTab === "create" ? (
        /* Form dâng sớ mới */
        <div className="bg-zinc-950/50 border border-zinc-900 rounded-3xl p-6 backdrop-blur-md flex flex-col gap-5">
          <h3 className="text-sm uppercase tracking-widest text-zinc-400 font-mono font-bold border-b border-zinc-900 pb-2">
            Thỉnh Nguyện Chi Sớ
          </h3>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <label className="text-xs text-zinc-500 font-mono">Đầu Đề</label>
              <input
                type="text"
                required
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Ví dụ: Đề xuất cân bằng sức mạnh Nguyệt Quang Cổ..."
                maxLength={100}
                className="bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-4 py-3 text-xs text-zinc-200 focus:outline-none placeholder-zinc-700 transition-all font-serif"
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="flex flex-col gap-2">
                <label className="text-xs text-zinc-500 font-mono">Hạng Mục Thỉnh Nguyện</label>
                <select
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                  className="bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-4 py-3 text-xs text-zinc-200 focus:outline-none transition-all font-mono"
                >
                  <option value="BUG">Lỗi game</option>
                  <option value="GU">Thêm Cổ Trùng hoang dã mới</option>
                  <option value="ITEM">Thêm vật phẩm tu luyện mới</option>
                  <option value="RECIPE">Thêm công thức luyện cổ mới</option>
                  <option value="BALANCE">Cân bằng lại chỉ số, hiệu ứng</option>
                  <option value="FEATURE">Đề xuất tính năng mới</option>
                  <option value="OTHER">Thỉnh nguyện khác</option>
                </select>
              </div>

              <div className="flex items-center p-3 rounded-2xl bg-zinc-900/30 border border-zinc-900/50 mt-6 sm:mt-0 text-[10px] text-zinc-500 font-mono">
                <HelpCircle className="w-4 h-4 text-zinc-600 shrink-0 mr-2" />
                Hãy chọn hạng mục phù hợp nhất để ý kiến được phân loại chính xác, Thiên Đạo sẽ dễ dàng xem xét sớ trình của đạo hữu hơn.
              </div>
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-xs text-zinc-500 font-mono">Sớ Trình Chi Tiết</label>
              <textarea
                required
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Kính thưa Thiên Đạo, tại hạ đề xuất tăng tỷ lệ luyện cổ thành công của..."
                rows={6}
                maxLength={1000}
                className="bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-4 py-3 text-xs text-zinc-200 focus:outline-none placeholder-zinc-700 transition-all font-serif resize-none"
              />
              <div className="text-right text-[10px] text-zinc-650 font-mono">
                {description.length} / 1000 kí tự
              </div>
            </div>

            <button
              type="submit"
              disabled={actionLoading || !title.trim() || !description.trim()}
              className="bg-gradient-to-r from-fuchsia-600 to-pink-600 hover:from-fuchsia-500 hover:to-pink-500 disabled:opacity-40 disabled:cursor-not-allowed text-white font-serif text-xs font-bold py-3.5 rounded-xl transition-all shadow-md active:scale-98 flex items-center justify-center gap-2 cursor-pointer mt-2"
            >
              {actionLoading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin text-zinc-200" />
                  <span>Đang truyền sớ...</span>
                </>
              ) : (
                <>
                  <Send className="w-3.5 h-3.5" />
                  <span>Kính dâng Thiên Đạo</span>
                </>
              )}
            </button>
          </form>
        </div>
      ) : (
        /* Danh sách góp ý */
        <div className="flex flex-col gap-4">
          {/* Filters Bar */}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-2xl p-3 sm:p-4 flex flex-col sm:flex-row gap-3 items-center justify-between backdrop-blur-sm">
            <div className="flex flex-wrap gap-2 w-full sm:w-auto">
              <select
                value={filterCategory}
                onChange={(e) => setFilterCategory(e.target.value)}
                className="bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-3 py-2 text-xs text-zinc-400 focus:outline-none transition-all font-mono"
              >
                {categories.map((c) => (
                  <option key={c.value} value={c.value}>
                    {c.label}
                  </option>
                ))}
              </select>

              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                className="bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-3 py-2 text-xs text-zinc-400 focus:outline-none transition-all font-mono"
              >
                {statuses.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="text-[10px] text-zinc-500 font-mono w-full sm:w-auto text-left sm:text-right mt-1 sm:mt-0 shrink-0">
              Tổng số góp ý hiển thị: <span className="font-bold text-zinc-300">{feedbacks.length}</span>
            </div>
          </div>

          {/* Cards List */}
          {feedbacks.length === 0 ? (
            <div className="bg-zinc-950/20 border border-zinc-900 border-dashed rounded-3xl p-12 text-center text-zinc-600 font-mono text-xs">
              Chưa có tờ sớ nào phù hợp bộ lọc hiện tại. Đạo hữu hãy khai mở sớ mới!
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              {feedbacks.map((f) => {
                const isAuthor = player && player.id === f.user_id;
                const isUpvoted = f.user_vote === 1;
                const isDownvoted = f.user_vote === -1;
                const userCanVote = player && player.rank >= 2;

                return (
                  <div
                    key={f.id}
                    className="bg-zinc-950/60 border border-zinc-900 rounded-2xl p-5 hover:border-zinc-800/80 transition-all duration-300 shadow-lg relative flex flex-col md:flex-row gap-5 items-start"
                  >
                    {/* Vote Column */}
                    <div className="flex md:flex-col items-center gap-2.5 shrink-0 bg-zinc-950/80 border border-zinc-900 rounded-xl p-2 md:p-3 w-full md:w-16 justify-between md:justify-center">
                      <button
                        onClick={() => voteFeedback(f.id, 1)}
                        disabled={actionLoading || !userCanVote}
                        className={`p-1.5 md:p-2 rounded-lg transition-colors cursor-pointer border ${
                          isUpvoted
                            ? "bg-emerald-950/80 border-emerald-500/35 text-emerald-400"
                            : "bg-transparent border-transparent hover:bg-zinc-900 text-zinc-500 hover:text-zinc-300"
                        }`}
                        title={
                          !userCanVote
                            ? "Đạo hữu phải đạt Nhị Chuyển mới được bình chọn!"
                            : "Ủng hộ ý kiến này"
                        }
                      >
                        <ThumbsUp className="w-4 h-4 md:w-5 md:h-5" />
                      </button>

                      <span
                        className={`text-xs md:text-sm font-mono font-bold ${
                          f.total_votes > 0
                            ? "text-emerald-400"
                            : f.total_votes < 0
                            ? "text-rose-400"
                            : "text-zinc-500"
                        }`}
                      >
                        {f.total_votes > 0 ? `+${f.total_votes}` : f.total_votes}
                      </span>

                      <button
                        onClick={() => voteFeedback(f.id, -1)}
                        disabled={actionLoading || !userCanVote}
                        className={`p-1.5 md:p-2 rounded-lg transition-colors cursor-pointer border ${
                          isDownvoted
                            ? "bg-rose-950/80 border-rose-500/35 text-rose-400"
                            : "bg-transparent border-transparent hover:bg-zinc-900 text-zinc-500 hover:text-zinc-300"
                        }`}
                        title={
                          !userCanVote
                            ? "Đạo hữu phải đạt Nhị Chuyển mới được bình chọn!"
                            : "Phản đối ý kiến này"
                        }
                      >
                        <ThumbsDown className="w-4 h-4 md:w-5 md:h-5" />
                      </button>
                    </div>

                    {/* Content Body */}
                    <div className="flex-grow flex flex-col gap-3 min-w-0 w-full">
                      {/* Meta information row */}
                      <div className="flex flex-wrap items-center gap-2 justify-between">
                        <div className="flex items-center gap-2">
                          {getStatusBadge(f.status)}
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-zinc-900 border border-zinc-800 text-zinc-400">
                            {getCategoryLabel(f.category)}
                          </span>
                        </div>

                        <span className="text-[10px] text-zinc-600 font-mono">
                          Dâng bởi: <span className="font-bold text-zinc-400">{f.username || "Ẩn danh"}</span>
                          {isAuthor && <span className="text-fuchsia-400 ml-1">(Đạo hữu)</span>}
                        </span>
                      </div>

                      {/* Title & Desc */}
                      <div className="flex flex-col gap-1.5">
                        <h4 className="text-sm font-serif font-bold text-zinc-100 leading-snug">
                          {f.title}
                        </h4>
                        <p className="text-xs text-zinc-400 leading-relaxed font-sans whitespace-pre-wrap">
                          {f.description}
                        </p>
                      </div>

                      {/* Developer response section */}
                      {f.developer_response && (
                        <div className="mt-2 p-3.5 rounded-xl bg-gradient-to-br from-zinc-900 to-zinc-950 border border-amber-950/60 relative overflow-hidden">
                          <div className="absolute top-0 right-0 w-24 h-24 bg-amber-500/3 rounded-full blur-xl pointer-events-none" />
                          <div className="flex items-center gap-1.5 mb-1.5 text-[10px] font-bold text-amber-500/90 font-mono">
                            <Shield className="w-3.5 h-3.5 text-amber-500" />
                            <span>THIÊN ĐẠO PHẢN HỒI</span>
                          </div>
                          <p className="text-[11px] text-zinc-300 font-serif leading-relaxed italic">
                            "{f.developer_response}"
                          </p>
                        </div>
                      )}

                      {/* Admin moderation form */}
                      {player && player.is_admin && (
                        <div className="mt-4 pt-3 border-t border-zinc-900 flex flex-col gap-3 bg-zinc-900/10 p-3 rounded-xl border-dashed">
                          <div className="flex items-center justify-between">
                            <span className="text-[10px] font-bold text-fuchsia-400 font-mono flex items-center gap-1">
                              <Shield className="w-3.5 h-3.5 text-fuchsia-400" />
                              Duyệt Ý Kiến
                            </span>
                            {modifyingFeedbackId !== f.id && (
                              <button
                                onClick={() => {
                                  setModifyingFeedbackId(f.id);
                                  setAdminStatus(f.status);
                                  setAdminResponse(f.developer_response || "");
                                }}
                                className="text-[10px] text-fuchsia-400 hover:text-fuchsia-300 hover:underline font-mono cursor-pointer"
                              >
                                Thay đổi trạng thái / Viết phản hồi
                              </button>
                            )}
                          </div>

                          {modifyingFeedbackId === f.id && (
                            <div className="flex flex-col gap-2">
                              <div className="grid grid-cols-2 gap-3">
                                <div className="flex flex-col gap-1">
                                  <label className="text-[9px] text-zinc-500 font-mono">Đặt trạng thái</label>
                                  <select
                                    value={adminStatus}
                                    onChange={(e) => setAdminStatus(e.target.value)}
                                    className="bg-zinc-950 border border-zinc-800 rounded-lg px-2 py-1 text-[11px] text-zinc-300 focus:outline-none font-mono"
                                  >
                                    <option value="pending">Chờ kiểm duyệt (pending)</option>
                                    <option value="active">Duyệt & Bình chọn (active)</option>
                                    <option value="in_progress">Ghi nhận phát triển (in_progress)</option>
                                    <option value="completed">Đã hoàn thành (completed)</option>
                                    <option value="rejected">Không tiếp nhận (rejected)</option>
                                  </select>
                                </div>
                                <div className="flex justify-end items-end gap-2">
                                  <button
                                    onClick={() => setModifyingFeedbackId(null)}
                                    className="px-2.5 py-1 text-[10px] font-semibold bg-zinc-900 hover:bg-zinc-800 rounded-lg border border-zinc-850 text-zinc-400 cursor-pointer"
                                  >
                                    Hủy
                                  </button>
                                  <button
                                    onClick={() => handleAdminUpdate(f.id)}
                                    disabled={actionLoading || !adminStatus}
                                    className="px-2.5 py-1 text-[10px] font-semibold bg-fuchsia-600 hover:bg-fuchsia-500 rounded-lg text-white cursor-pointer flex items-center gap-1"
                                  >
                                    {actionLoading && <Loader2 className="w-3 h-3 animate-spin" />}
                                    Cập nhật
                                  </button>
                                </div>
                              </div>

                              <div className="flex flex-col gap-1">
                                <label className="text-[9px] text-zinc-500 font-mono">Lời phản hồi của Thiên Đạo</label>
                                <textarea
                                  value={adminResponse}
                                  onChange={(e) => setAdminResponse(e.target.value)}
                                  placeholder="Ghi lý do hoặc lời chúc của Thiên Đạo khi duyệt..."
                                  rows={2}
                                  className="bg-zinc-950 border border-zinc-850 rounded-lg px-2 py-1.5 text-xs text-zinc-300 focus:outline-none font-serif resize-none"
                                />
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
