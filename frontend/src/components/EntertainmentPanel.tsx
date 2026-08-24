"use client";

import React, { useEffect, useState } from "react";
import { Gift, Sparkles, Trophy, ChevronDown, ChevronUp, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";
import { GachaPanel } from "./GachaPanel";

interface BettingHistoryRound {
	id: string;
	status: string;
	winner: string;
	team_a_name: string;
	team_b_name: string;
	total_bet_a: number;
	total_bet_b: number;
	battle_log: string;
	created_at: string;
	resolved_at: string | null;
}

export const EntertainmentPanel: React.FC = () => {
	const [activeTab, setActiveTab] = useState<"gacha" | "betting">("gacha");
	const {
		currentBettingRound,
		myBetTeam,
		myBetAmount,
		bettingHistory,
		fetchCurrentBettingRound,
		fetchBettingHistory,
		placeBeastBet,
		actionLoading,
		player,
	} = useGameStore();

	const [selectedTeam, setSelectedTeam] = useState<"team_a" | "team_b">("team_a");
	const [betAmount, setBetAmount] = useState<number>(1000);
	const [expandedRoundId, setExpandedRoundId] = useState<string | null>(null);

	// Timers for locking/resolving
	const [timeLeft, setTimeLeft] = useState("");
	const [isBetLocked, setIsBetLocked] = useState(false);

	useEffect(() => {
		if (activeTab === "betting") {
			fetchCurrentBettingRound();
			fetchBettingHistory();
		}
	}, [activeTab, fetchCurrentBettingRound, fetchBettingHistory]);

	useEffect(() => {
		const updateTimer = () => {
			const now = new Date();
			const mins = now.getMinutes();

			if (mins >= 55) {
				setIsBetLocked(true);
				const nextHour = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours() + 1, 0, 0);
				const diffMs = nextHour.getTime() - now.getTime();
				const diffMins = Math.floor(diffMs / 60000);
				const diffSecs = Math.floor((diffMs % 60000) / 1000);
				setTimeLeft(`Đã khóa cược. Trận đấu bắt đầu sau: ${diffMins}p ${diffSecs}s`);
			} else {
				setIsBetLocked(false);
				const lockTime = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours(), 55, 0);
				const diffMs = lockTime.getTime() - now.getTime();
				const diffMins = Math.floor(diffMs / 60000);
				const diffSecs = Math.floor((diffMs % 60000) / 1000);
				setTimeLeft(`Thời gian đặt cược còn lại: ${diffMins}p ${diffSecs}s`);
			}
		};
		updateTimer();
		const interval = setInterval(updateTimer, 1000);
		return () => clearInterval(interval);
	}, []);

	const handlePlaceBet = async (e: React.FormEvent) => {
		e.preventDefault();
		if (isBetLocked) return;
		if (betAmount <= 0) return;
		const result = await placeBeastBet(selectedTeam, betAmount);
		if (result?.success) {
			setBetAmount(1000);
		}
	};

	const totalA = currentBettingRound?.total_bet_a || 0;
	const totalB = currentBettingRound?.total_bet_b || 0;
	const totalPool = totalA + totalB;

	const estMultA = totalA > 0 ? (totalPool / totalA) : 1;
	const estMultB = totalB > 0 ? (totalPool / totalB) : 1;

	const formatStones = (v: number) => {
		return Math.floor(v).toLocaleString();
	};

	return (
		<div className="bg-zinc-950 border border-zinc-800 rounded-3xl p-6 max-w-4xl w-full mx-auto shadow-2xl space-y-6">
			{/* Header / Tabs */}
			<div className="flex flex-col sm:flex-row items-center justify-between border-b border-zinc-800 pb-4 gap-4">
				<div className="flex items-center gap-3">
					<div className="w-10 h-10 rounded-2xl bg-gradient-to-br from-pink-500/20 to-rose-600/20 flex items-center justify-center border border-pink-500/30">
						<Sparkles className="w-5 h-5 text-pink-400" />
					</div>
					<div>
						<h2 className="font-serif font-bold text-xl text-zinc-100">Khu Giải Trí</h2>
						<p className="text-xs text-zinc-500">Mở đá cầu cơ duyên hoặc cược đấu thú đọ may mắn!</p>
					</div>
				</div>

				<div className="flex bg-zinc-900/60 p-1 rounded-2xl border border-zinc-800">
					<button
						onClick={() => setActiveTab("gacha")}
						className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-all cursor-pointer ${
							activeTab === "gacha"
								? "bg-zinc-800 text-amber-400 shadow-sm"
								: "text-zinc-400 hover:text-zinc-200"
						}`}
					>
						<Gift className="w-4 h-4" />
						Cổ Từ Bảo Khố
					</button>
					<button
						onClick={() => setActiveTab("betting")}
						className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-medium transition-all cursor-pointer ${
							activeTab === "betting"
								? "bg-zinc-800 text-amber-400 shadow-sm"
								: "text-zinc-400 hover:text-zinc-200"
						}`}
					>
						<Trophy className="w-4 h-4" />
						Đấu Thú Đặt Cược
					</button>
				</div>
			</div>

			{/* Tab Contents */}
			{activeTab === "gacha" ? (
				<GachaPanel />
			) : (
				<div className="space-y-6">
					{/* Current Round Card */}
					<div className="bg-zinc-900/40 border border-zinc-800/80 rounded-2xl p-5 space-y-4">
						<div className="flex items-center justify-between">
							<span className="text-xs font-bold font-serif text-amber-400">TRẬN ĐẤU ĐANG DIỄN RA</span>
							<button
								onClick={() => {
									fetchCurrentBettingRound();
									fetchBettingHistory();
								}}
								className="text-zinc-500 hover:text-zinc-300 transition-colors p-1"
								title="Làm mới dữ liệu"
							>
								<RefreshCw className="w-4 h-4" />
							</button>
						</div>

						<div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-center">
							{/* Match Pools Visualizer */}
							<div className="space-y-3">
								<div className="flex items-center justify-between text-xs font-semibold">
									<span className="text-rose-400">🔴 Tả Thanh Long</span>
									<span className="text-zinc-500">VS</span>
									<span className="text-blue-400">🔵 Hữu Bạch Hổ</span>
								</div>

								{/* Progress Bar representation */}
								<div className="h-4 bg-zinc-950 rounded-full overflow-hidden border border-zinc-800 flex">
									<div
										style={{ width: `${totalPool > 0 ? (totalA / totalPool) * 100 : 50}%` }}
										className="bg-gradient-to-r from-red-600 to-rose-500 transition-all duration-500"
									/>
									<div
										style={{ width: `${totalPool > 0 ? (totalB / totalPool) * 100 : 50}%` }}
										className="bg-gradient-to-l from-blue-600 to-indigo-500 transition-all duration-500"
									/>
								</div>
							</div>

							{/* User Bet Details and Timer */}
							<div className="bg-zinc-950/50 rounded-xl p-4 border border-zinc-800/60 flex flex-col justify-between h-full min-h-[100px] gap-2">
								<div className="flex items-center gap-2 text-zinc-400 text-xs font-mono">
									<AlertCircle className="w-4 h-4 text-amber-500/80" />
									<span className={isBetLocked ? "text-rose-400" : "text-emerald-400"}>{timeLeft}</span>
								</div>

								{myBetTeam ? (
									<div className="bg-amber-500/5 border border-amber-500/20 rounded-lg p-2.5 text-xs">
										🎉 Đạo hữu đã đặt cược:{" "}
										<span className={myBetTeam === "team_a" ? "text-rose-400 font-bold" : "text-blue-400 font-bold"}>
											{myBetTeam === "team_a" ? "Phe Đỏ (Tả Thanh Long)" : "Phe Xanh (Hữu Bạch Hổ)"}
										</span>{" "}
										với số tiền <span className="text-amber-400 font-mono font-bold">{formatStones(myBetAmount || 0)}</span> Nguyên Thạch.
									</div>
								) : (
									<div className="text-zinc-500 text-xs italic">
										Chưa đặt cược cho trận đấu này. Hãy chọn bên và nhập số tiền cược bên dưới!
									</div>
								)}
							</div>
						</div>
					</div>

					{/* Place Bet Form */}
					{!myBetTeam && !isBetLocked && (
						<form onSubmit={handlePlaceBet} className="bg-zinc-900/20 border border-zinc-800/80 rounded-2xl p-5 space-y-4">
							<div className="text-xs font-bold text-zinc-300">ĐẶT CƯỢC PHOÊN ĐẤU</div>

							<div className="grid grid-cols-2 gap-4">
								<button
									type="button"
									onClick={() => setSelectedTeam("team_a")}
									className={`py-3.5 rounded-xl border text-sm font-semibold transition-all cursor-pointer ${
										selectedTeam === "team_a"
											? "bg-rose-950/40 border-rose-500/60 text-rose-200 shadow-[0_0_15px_rgba(244,63,94,0.15)]"
											: "bg-zinc-900/50 border-zinc-800 text-zinc-400 hover:border-zinc-700"
									}`}
								>
									🔴 Tả Thanh Long (Phe Đỏ)
								</button>
								<button
									type="button"
									onClick={() => setSelectedTeam("team_b")}
									className={`py-3.5 rounded-xl border text-sm font-semibold transition-all cursor-pointer ${
										selectedTeam === "team_b"
											? "bg-blue-950/40 border-blue-500/60 text-blue-200 shadow-[0_0_15px_rgba(59,130,246,0.15)]"
											: "bg-zinc-900/50 border-zinc-800 text-zinc-400 hover:border-zinc-700"
									}`}
								>
									🔵 Hữu Bạch Hổ (Phe Xanh)
								</button>
							</div>

							<div className="flex flex-col sm:flex-row items-center gap-4">
								<div className="relative flex-1 w-full">
									<input
										type="number"
										min={100}
										step={100}
										value={betAmount}
										onChange={(e) => setBetAmount(Math.max(0, parseInt(e.target.value) || 0))}
										className="w-full bg-zinc-950 border border-zinc-800 rounded-xl px-4 py-3 text-sm font-mono font-bold text-amber-400 outline-none focus:border-amber-600"
										placeholder="Số lượng Nguyên Thạch..."
									/>
									<span className="absolute right-4 top-1/2 -translate-y-1/2 text-[10px] text-zinc-500 font-mono">
										Nguyên Thạch
									</span>
								</div>

								<div className="flex gap-1.5 w-full sm:w-auto">
									{[1000, 5000, 20000, 100000].map((v) => (
										<button
											key={v}
											type="button"
											onClick={() => setBetAmount(v)}
											className="px-2.5 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-[10px] font-mono text-zinc-400 hover:text-zinc-200 hover:border-zinc-700 cursor-pointer"
										>
											{formatStones(v)}
										</button>
									))}
									<button
										type="button"
										onClick={() => setBetAmount(Math.floor(player?.primeval_stones || 0))}
										className="px-2.5 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-[10px] font-mono text-amber-500 hover:text-amber-400 hover:border-zinc-700 cursor-pointer"
									>
										Tất Tay
									</button>
								</div>
							</div>

							<button
								type="submit"
								disabled={actionLoading || (player?.primeval_stones || 0) < betAmount || betAmount < 100}
								className="w-full py-3 bg-amber-600 hover:bg-amber-500 disabled:bg-zinc-800 disabled:text-zinc-600 text-white rounded-xl text-xs font-bold transition-all shadow-md cursor-pointer flex items-center justify-center gap-2"
							>
								{actionLoading ? (
									<Loader2 className="w-4 h-4 animate-spin" />
								) : (
									"Xác Nhận Đặt Cược"
								)}
							</button>
						</form>
					)}

					{isBetLocked && !myBetTeam && (
						<div className="bg-zinc-900/10 border border-zinc-800 rounded-2xl p-5 text-center text-zinc-500 text-xs italic">
							🔒 Vòng đặt cược hiện tại đã bị khóa (Khóa sau phút 55 mỗi giờ). Hãy chờ trận đấu phân định kết quả và mở cược vòng tiếp theo.
						</div>
					)}

					{myBetTeam && (
						<div className="bg-zinc-900/10 border border-zinc-800 rounded-2xl p-5 text-center text-emerald-500/80 text-xs font-medium">
							✓ Đã lưu thông tin cược thành công. Trận đấu sẽ tự động giải quyết vào cuối giờ.
						</div>
					)}

					{/* History */}
					<div className="space-y-3">
						<div className="text-xs font-bold text-zinc-300">LỊCH SỬ PHIÊN ĐẤU</div>

						{bettingHistory.length === 0 ? (
							<div className="text-center py-6 text-zinc-600 italic text-xs">Chưa có lịch sử đấu thú...</div>
						) : (
							<div className="space-y-3">
								{bettingHistory.map((h: BettingHistoryRound) => {
									const isExpanded = expandedRoundId === h.id;
									const dateLabel = new Date(h.created_at).toLocaleTimeString("vi-VN", {
										hour: "2-digit",
										minute: "2-digit",
										day: "2-digit",
										month: "2-digit",
									});

									return (
										<div key={h.id} className="bg-zinc-900/30 border border-zinc-800 rounded-xl overflow-hidden">
											<div
												onClick={() => setExpandedRoundId(isExpanded ? null : h.id)}
												className="flex items-center justify-between p-4 cursor-pointer hover:bg-zinc-900/50 transition-colors"
											>
												<div className="space-y-1">
													<div className="text-xs text-zinc-400 font-mono">{dateLabel}</div>
													<div className="text-xs font-semibold text-zinc-300">
														Kết quả:{" "}
														{h.winner === "team_a" ? (
															<span className="text-rose-400 font-bold">Phe Đỏ Thắng</span>
														) : h.winner === "team_b" ? (
															<span className="text-blue-400 font-bold">Phe Xanh Thắng</span>
														) : (
															<span className="text-zinc-400 font-bold">Hòa</span>
														)}
													</div>
												</div>

												<div className="flex items-center gap-4 text-xs font-mono">
													<div className="text-zinc-500">
														Tổng Pool:{" "}
														<span className="text-zinc-400">{formatStones(h.total_bet_a + h.total_bet_b)}</span>
													</div>
													{isExpanded ? (
														<ChevronUp className="w-4 h-4 text-zinc-500" />
													) : (
														<ChevronDown className="w-4 h-4 text-zinc-500" />
													)}
												</div>
											</div>

											{isExpanded && (
												<div className="border-t border-zinc-800 bg-zinc-950 p-4 space-y-2">
													<div className="text-[10px] uppercase font-bold text-amber-500 font-serif tracking-wider mb-2">
														Chi tiết trận đấu
													</div>
													<div className="bg-zinc-900/60 p-3 rounded-lg border border-zinc-800 font-mono text-[10px] text-zinc-300 leading-relaxed whitespace-pre-line max-h-60 overflow-y-auto">
														{h.battle_log}
													</div>
												</div>
											)}
										</div>
									);
								})}
							</div>
						)}
					</div>
				</div>
			)}
		</div>
	);
};
