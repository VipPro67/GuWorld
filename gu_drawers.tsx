              const renderGroup = ({
                guId,
                groupKey,
                spec,
                instances,
                hasVital,
                activeCount,
                sleepingCount,
                sealedCount,
                dyingCount,
                deadCount,
                equippedCount,
              }: typeof guGroups[number]) => {
                const totalCount = instances.length;
                let cardStyle = "bg-zinc-950/60 border-zinc-900 hover:border-zinc-800";
                if (hasVital) {
                  cardStyle = "border-amber-400/60 bg-amber-950/5 shadow-[0_0_15px_rgba(245,158,11,0.1)] hover:border-amber-500/80";
                }

                const handleGroupClick = () => {
                  if (totalCount === 1) {
                    setActiveInstanceId(instances[0].instance_id);
                  } else {
                    setActiveGroupKey(groupKey);
                    setActiveGroupViewMode("summary");
                    setGroupFeedAmount(1); // default
                  }
                };

                return (
                  <div
                    key={groupKey}
                    onClick={handleGroupClick}
                    className={`p-3 sm:p-4 rounded-2xl border transition-all cursor-pointer ${cardStyle}`}
                  >
                    <div className="flex justify-between items-center gap-3">
                      <div className="flex flex-col gap-1.5">
                        <h4 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
                          {spec.name}
                        </h4>
                        <div className="flex items-center gap-1.5 flex-wrap">
                          {hasVital && (
                            <span className="text-[8px] font-bold text-amber-300 bg-amber-950/60 border border-amber-500/60 px-1 py-0.5 rounded uppercase tracking-wider">
                              ⚡ Bản Mệnh
                            </span>
                          )}
                          <span className="text-[8px] font-mono font-bold text-zinc-400 border border-zinc-800 bg-zinc-900 px-1 py-0.5 rounded">
                            Bậc {spec.rank}
                          </span>
                          {spec.path && spec.path.map((p: string) => (
                            <span key={p} className={`text-[8px] font-mono px-1 py-0.5 rounded border leading-none ${getPathColor(p)}`}>
                              {translatePath(p)}
                            </span>
                          ))}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-[14px] font-mono font-bold text-zinc-400">
                          x{totalCount}
                        </span>
                        <ChevronRight className="w-4 h-4 text-zinc-600" />
                      </div>
                    </div>
                  </div>
                );
              };
