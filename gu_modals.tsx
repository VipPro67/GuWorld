  // Derived state for Drawers
  const activeGroup = activeGroupKey ? (() => {
    const [guId, rankStr] = activeGroupKey.split("::");
    const rank = parseInt(rankStr, 10);
    const insts = guList.filter(g => g.gu_id === guId && (g.rank ?? 0) === rank);
    const spec = guSpecFor(guId, rank);
    return { guId, rank, instances: insts, spec };
  })() : null;

  const activeInstance = activeInstanceId ? (() => {
    const inst = guList.find(g => g.instance_id === activeInstanceId);
    if (!inst) return null;
    const spec = guSpecFor(inst.gu_id, inst.rank);
    return { instance: inst, spec };
  })() : null;

  const renderGroupDrawer = () => {
    if (!activeGroup || !activeGroup.spec) return null;
    const { spec, instances, guId, rank } = activeGroup;
    
    // Summary Data
    const equippedCount = instances.filter(i => i.equipped).length;
    const idleCount = instances.filter(i => i.status !== "DEAD" && i.status !== "SEALED" && !i.equipped).length;
    
    // For List view
    const filteredInstances = instances.filter(inst => {
      if (groupListTab === "equipped" && !inst.equipped) return false;
      if (groupListTab === "feeding" && (inst.status === "DEAD" || inst.status === "SEALED" || inst.hunger >= spec.max_hunger)) return false;
      if (groupListTab === "idle" && (inst.equipped || inst.status === "DEAD" || inst.status === "SEALED")) return false;
      if (groupListSearch && !inst.instance_id.toLowerCase().includes(groupListSearch.toLowerCase())) return false;
      return true;
    }).sort((a, b) => {
      if (groupListSort === "hunger_asc") return a.hunger - b.hunger;
      return b.hunger - a.hunger;
    });

    return (
      <div className="fixed inset-0 z-50 flex flex-col justify-end bg-black/80 animate-in fade-in">
        <div className="bg-zinc-950 border-t border-zinc-900 rounded-t-3xl w-full max-h-[90vh] flex flex-col relative pb-4">
          {/* Handle & Header */}
          <div className="w-full flex justify-center py-3 shrink-0">
            <div className="w-10 h-1.5 bg-zinc-800 rounded-full" />
          </div>
          <button 
            onClick={() => setActiveGroupKey(null)}
            className="absolute top-4 right-4 p-2 bg-zinc-900 rounded-xl text-zinc-400 hover:text-white"
          >
            <X className="w-5 h-5" />
          </button>
          
          <div className="px-5 pb-4 shrink-0">
            <div className="flex items-center gap-3">
              {activeGroupViewMode === "list" && (
                <button onClick={() => setActiveGroupViewMode("summary")} className="p-2 -ml-2 text-zinc-400 hover:text-white">
                  <ArrowLeft className="w-5 h-5" />
                </button>
              )}
              <div>
                <h2 className="text-xl font-bold text-white font-serif flex items-center gap-2">
                  {spec.name}
                  <span className="text-sm font-mono text-zinc-400 font-normal">×{instances.length}</span>
                </h2>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-[11px] text-zinc-400 font-mono">{instances.length} cá thể</span>
                  <span className="text-[11px] text-zinc-600">•</span>
                  <span className="text-[11px] text-zinc-400 font-mono">Bậc {rank}</span>
                  {spec.path && spec.path.map((p: string) => (
                    <React.Fragment key={p}>
                      <span className="text-[11px] text-zinc-600">•</span>
                      <span className={`text-[11px] font-mono ${getPathColor(p)} bg-transparent border-none p-0`}>{translatePath(p)}</span>
                    </React.Fragment>
                  ))}
                </div>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto px-5">
            {activeGroupViewMode === "summary" ? (
              <div className="space-y-6">
                <div>
                  <h3 className="text-sm font-bold text-white mb-3">Tổng quan</h3>
                  <div className="grid grid-cols-3 gap-2">
                    <div className="bg-zinc-900/50 border border-zinc-800 rounded-xl p-3 flex flex-col items-center justify-center text-center">
                      <span className="text-[10px] text-zinc-400 mb-1">Đang trang bị</span>
                      <span className="text-xl font-bold text-white font-mono">{equippedCount}</span>
                    </div>
                    <div className="bg-zinc-900/50 border border-zinc-800 rounded-xl p-3 flex flex-col items-center justify-center text-center">
                      <span className="text-[10px] text-zinc-400 mb-1">Đang nuôi</span>
                      <span className="text-xl font-bold text-white font-mono">{instances.length - equippedCount}</span>
                    </div>
                    <div className="bg-zinc-900/50 border border-zinc-800 rounded-xl p-3 flex flex-col items-center justify-center text-center">
                      <span className="text-[10px] text-zinc-400 mb-1">Có thể sử dụng</span>
                      <span className="text-xl font-bold text-white font-mono">{idleCount}</span>
                    </div>
                  </div>
                </div>

                {spec.gu_type !== "Consumable" && (
                  <div>
                    <h3 className="text-sm font-bold text-white mb-3">Chọn số lượng</h3>
                    <div className="flex items-center gap-2 mb-3">
                      <button 
                        onClick={() => setGroupFeedAmount(Math.max(1, groupFeedAmount - 1))}
                        className="w-12 h-12 flex items-center justify-center bg-zinc-900 border border-zinc-800 rounded-xl text-zinc-400 hover:bg-zinc-800 text-xl"
                      >
                        -
                      </button>
                      <div className="flex-1 h-12 flex items-center justify-center bg-zinc-950 border border-zinc-800 rounded-xl">
                        <span className="text-lg font-mono text-white">{groupFeedAmount}</span>
                      </div>
                      <button 
                        onClick={() => setGroupFeedAmount(Math.min(idleCount, groupFeedAmount + 1))}
                        className="w-12 h-12 flex items-center justify-center bg-zinc-900 border border-zinc-800 rounded-xl text-zinc-400 hover:bg-zinc-800 text-xl"
                      >
                        +
                      </button>
                      <span className="text-[10px] text-zinc-500 w-16 text-center">Tối đa {idleCount}</span>
                    </div>
                    
                    <button 
                      disabled={actionLoading}
                      onClick={() => {
                        // Backend only supports feed group by rank for now, which feeds all hungry.
                        handleFeedGroupClick(guId, rank);
                      }}
                      className="w-full py-3.5 bg-emerald-600 text-white font-bold rounded-xl mb-3 shadow-[0_0_15px_rgba(16,185,129,0.2)]"
                    >
                      Cho ăn nhóm
                    </button>
                    
                    <button 
                      onClick={() => setActiveGroupViewMode("list")}
                      className="w-full py-3.5 bg-zinc-950 border border-emerald-900/50 text-emerald-400 font-bold rounded-xl mb-3"
                    >
                      Xem từng cá thể
                    </button>
                    
                    <button 
                      className="w-full py-3.5 bg-zinc-950 border border-zinc-800 text-zinc-400 font-bold rounded-xl flex items-center justify-center gap-2"
                    >
                      Khóa đã chọn
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-4">
                {/* Search */}
                <div className="relative">
                  <Search className="w-4 h-4 text-zinc-500 absolute left-3 top-3" />
                  <input 
                    type="text" 
                    placeholder="Tìm theo mã Cổ..." 
                    value={groupListSearch}
                    onChange={(e) => setGroupListSearch(e.target.value)}
                    className="w-full bg-zinc-900/50 border border-zinc-800 text-zinc-300 text-sm rounded-xl pl-9 pr-4 py-2.5 outline-none focus:border-emerald-500/50 font-mono"
                  />
                </div>

                {/* Tabs */}
                <div className="flex gap-2 overflow-x-auto pb-1 scrollbar-hide">
                  {[
                    { id: "all", label: `Tất cả ${instances.length}` },
                    { id: "equipped", label: `Trang bị ${equippedCount}` },
                    { id: "feeding", label: `Đang nuôi ${instances.length - equippedCount}` },
                    { id: "idle", label: `Rảnh ${idleCount}` },
                  ].map(tab => (
                    <button
                      key={tab.id}
                      onClick={() => setGroupListTab(tab.id as any)}
                      className={`whitespace-nowrap px-4 py-2 rounded-xl text-xs font-medium transition-all ${
                        groupListTab === tab.id 
                          ? "bg-emerald-950/30 text-emerald-400 border border-emerald-900" 
                          : "bg-zinc-900 border border-zinc-800 text-zinc-400"
                      }`}
                    >
                      {tab.label}
                    </button>
                  ))}
                </div>

                {/* Sort */}
                <div>
                  <button 
                    onClick={() => setGroupListSort(groupListSort === "hunger_desc" ? "hunger_asc" : "hunger_desc")}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-[10px] text-zinc-400"
                  >
                    Độ no {groupListSort === "hunger_desc" ? "↓" : "↑"}
                  </button>
                </div>

                {/* List Header */}
                <div className="grid grid-cols-12 text-[10px] text-zinc-500 px-2 pb-2 border-b border-zinc-800">
                  <div className="col-span-5">Mã Cổ</div>
                  <div className="col-span-3 text-center">Trạng thái</div>
                  <div className="col-span-4 text-right">Độ no</div>
                </div>

                {/* List Items */}
                <div className="space-y-1">
                  {filteredInstances.map(inst => {
                    const isHungry = inst.hunger < spec.max_hunger;
                    const hungerPercent = Math.min(100, Math.max(0, (inst.hunger / spec.max_hunger) * 100));
                    let statusColor = "text-zinc-500 border-zinc-800";
                    let statusText = "Rảnh";
                    if (inst.equipped) {
                      statusColor = "text-emerald-400 border-emerald-900/50";
                      statusText = "Trang bị";
                    } else if (isHungry) {
                      statusColor = "text-amber-400 border-amber-900/50";
                      statusText = "Đang nuôi";
                    }

                    return (
                      <div 
                        key={inst.instance_id}
                        onClick={() => setActiveInstanceId(inst.instance_id)}
                        className="grid grid-cols-12 items-center py-3 px-2 hover:bg-zinc-900/30 rounded-xl cursor-pointer transition-colors border-b border-zinc-900/50 last:border-0"
                      >
                        <div className="col-span-5 text-xs text-zinc-300 font-mono">
                          #{inst.instance_id.slice(0, 6)}
                        </div>
                        <div className="col-span-3 flex justify-center">
                          <span className={`text-[9px] px-2 py-0.5 rounded border ${statusColor}`}>
                            {statusText}
                          </span>
                        </div>
                        <div className="col-span-4 flex items-center justify-end gap-2">
                          <div className="flex flex-col items-end w-full">
                            <span className="text-[10px] text-zinc-300 font-mono mb-1">{inst.hunger} / {spec.max_hunger}</span>
                            <div className="w-16 h-1 bg-zinc-800 rounded-full overflow-hidden">
                              <div 
                                className={`h-full ${hungerPercent < 30 ? "bg-amber-500" : hungerPercent < 10 ? "bg-rose-500" : "bg-emerald-500"}`}
                                style={{ width: `${hungerPercent}%` }}
                              />
                            </div>
                          </div>
                          <ChevronRight className="w-4 h-4 text-zinc-600 shrink-0 ml-1" />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  };

  const renderSingleInstanceDrawer = () => {
    if (!activeInstance || !activeInstance.spec) return null;
    const { instance: inst, spec } = activeInstance;
    const isDead = inst.status === "DEAD";
    const hungerPercent = Math.min(100, Math.max(0, (inst.hunger / spec.max_hunger) * 100));
    
    return (
      <div className="fixed inset-0 z-[60] flex flex-col justify-end bg-black/80 animate-in fade-in">
        <div className="bg-zinc-950 border-t border-zinc-900 rounded-t-3xl w-full flex flex-col relative pb-6">
          <div className="w-full flex justify-center py-3 shrink-0">
            <div className="w-10 h-1.5 bg-zinc-800 rounded-full" />
          </div>
          <button 
            onClick={() => setActiveInstanceId(null)}
            className="absolute top-4 right-4 p-2 bg-zinc-900 rounded-xl text-zinc-400 hover:text-white"
          >
            <X className="w-5 h-5" />
          </button>

          <div className="px-5">
            <h2 className="text-xl font-bold text-white font-serif mb-2">{spec.name}</h2>
            <div className="flex items-center gap-1.5 flex-wrap mb-6">
              {inst.is_vital && (
                <span className="text-[10px] font-bold text-amber-300 border border-amber-500/50 px-2 py-0.5 rounded flex items-center gap-1">
                  <Zap className="w-3 h-3" /> Bản Mệnh
                </span>
              )}
              {spec.path && spec.path.map((p: string) => (
                <span key={p} className={`text-[10px] px-2 py-0.5 rounded border bg-transparent ${getPathColor(p)}`}>
                  {translatePath(p)}
                </span>
              ))}
              <span className="text-[10px] text-amber-500 border border-amber-900/50 px-2 py-0.5 rounded bg-transparent">
                Bậc {spec.rank}
              </span>
            </div>

            <div className="space-y-4 mb-6">
              <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
                <span className="text-xs text-zinc-500">Mã Cổ</span>
                <span className="text-xs text-zinc-300 font-mono">#{inst.instance_id.slice(0, 6)}</span>
              </div>
              <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
                <span className="text-xs text-zinc-500">Trạng thái</span>
                <span className="text-xs text-emerald-400 font-mono">
                  {inst.equipped ? "Xuất trận" : "Rảnh"} • {inst.status === "ACTIVE" ? "Hoạt động" : inst.status === "SLEEPING" ? "Ngủ đông" : "Phong ấn"}
                </span>
              </div>
              <div className="border-b border-zinc-900 pb-3">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-xs text-zinc-500">Độ no</span>
                  <span className="text-xs text-zinc-300 font-mono">{inst.hunger} / {spec.max_hunger}</span>
                </div>
                <div className="w-full h-1.5 bg-zinc-900 rounded-full overflow-hidden mb-1">
                  <div 
                    className={`h-full ${hungerPercent < 30 ? "bg-amber-500" : "bg-emerald-500"}`}
                    style={{ width: `${hungerPercent}%` }}
                  />
                </div>
                <span className="text-[10px] text-zinc-500">
                  {inst.hunger >= spec.max_hunger ? "Đã no, chưa thể cho ăn" : "Đang đói"}
                </span>
              </div>
              <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
                <span className="text-xs text-zinc-500">Nhóm nuôi</span>
                <span className="text-xs text-zinc-300">0 Cổ</span>
              </div>
            </div>

            {/* Actions */}
            <div className="flex gap-2 mb-3">
              <button
                disabled={actionLoading || inst.hunger >= spec.max_hunger || isDead}
                onClick={() => handleFeedClick(inst.instance_id)}
                className={`flex-1 py-3.5 rounded-xl text-sm font-bold transition-all ${
                  inst.hunger < spec.max_hunger && !isDead
                    ? "bg-zinc-900 text-white border border-zinc-700"
                    : "bg-zinc-950 text-zinc-600 border border-zinc-900"
                }`}
              >
                Cho ăn
              </button>
              {onEquipToggle && (
                <button
                  disabled={actionLoading || isDead}
                  onClick={() => handleEquipClick(inst.instance_id, !inst.equipped)}
                  className="flex-1 py-3.5 bg-indigo-950/40 border border-indigo-900 text-indigo-400 font-bold rounded-xl"
                >
                  {inst.equipped ? "Tháo" : "Trang bị"}
                </button>
              )}
            </div>
            
            <div className="flex gap-2">
              <button className="flex-[3] py-3.5 bg-emerald-950/20 border border-emerald-900/50 text-emerald-500 font-bold rounded-xl">
                Khóa
              </button>
              <button 
                onClick={async () => {
                   if (!await useGameStore.getState().showConfirm(`Xác nhận thả Cổ trùng này?`)) return;
                   if (onDropGu) await onDropGu(inst.instance_id);
                   setActiveInstanceId(null);
                }}
                className="flex-1 flex items-center justify-center py-3.5 bg-zinc-950 border border-zinc-800 text-zinc-400 font-bold rounded-xl"
              >
                Thả
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  };
