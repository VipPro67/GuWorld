import React from "react";
import { Activity, AlertTriangle, BarChart3, BookOpen, Layers, Sparkles, Swords, Target, Zap } from "lucide-react";
import { WikiPage } from "./types";
import {
  COMBAT_STATUSES,
  EFFECT_CONDITIONS,
  EFFECT_CONDITION_OPERATORS,
  EFFECT_OPERATIONS,
  EFFECT_RESOURCES,
  EFFECT_STACK_POLICIES,
  EFFECT_STATS,
  EFFECT_TARGETS,
  EFFECT_TRIGGERS,
  EFFECT_UNITS,
  HOOK_GROUP_LABELS,
  HOOK_STATUS_LABELS,
  NATIVE_HOOKS,
  PERSISTENT_STATUSES,
  TIMING_GROUPS,
  type HookGroup,
  type HookStatus,
} from "./effect-glossary";

const card = "p-3 rounded-lg bg-[var(--surface)] border border-[var(--border)]";
const codeClass = "font-mono text-[10px] px-1.5 py-0.5 rounded bg-zinc-950/80 border border-zinc-800 text-emerald-300";

function Code({ children }: { children: React.ReactNode }) {
  return <span className={codeClass}>{children}</span>;
}

function SectionTitle({ icon, children }: { icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
      {icon}
      {children}
    </h3>
  );
}

function Table({ columns, rows }: { columns: string[]; rows: React.ReactNode[][] }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-[var(--border)]">
      <table className="w-full text-left text-xs">
        <thead className="bg-zinc-950 text-[10px] uppercase tracking-wider text-[var(--text-secondary)] font-mono">
          <tr>
            {columns.map((c) => (
              <th key={c} className="p-2.5 whitespace-nowrap">{c}</th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-900 bg-[var(--surface)] text-[var(--text-secondary)]">
          {rows.map((row, i) => (
            <tr key={i} className="align-top">
              {row.map((cell, j) => (
                <td key={j} className="p-2.5 leading-relaxed">{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Note({ children, tone = "warning" }: { children: React.ReactNode; tone?: "warning" | "info" }) {
  const styles = tone === "warning"
    ? "bg-amber-950/25 border-amber-500/30 text-amber-100"
    : "bg-indigo-950/25 border-indigo-500/30 text-indigo-100";
  return (
    <div className={`p-3 rounded-xl border text-xs leading-relaxed flex gap-2 ${styles}`}>
      <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
      <div>{children}</div>
    </div>
  );
}

function StatusBadge({ status }: { status: HookStatus }) {
  const tone: Record<HookStatus, string> = {
    active: "bg-emerald-950/60 text-emerald-300 border-emerald-800/50",
    partial: "bg-amber-950/60 text-amber-300 border-amber-800/50",
    legacy: "bg-sky-950/60 text-sky-300 border-sky-800/50",
    unimplemented: "bg-zinc-800/80 text-zinc-400 border-zinc-700/60",
  };
  return (
    <span className={`inline-block whitespace-nowrap font-mono text-[10px] px-1.5 py-0.5 rounded border ${tone[status]}`}>
      {HOOK_STATUS_LABELS[status]}
    </span>
  );
}

const HOOK_GROUP_ORDER: HookGroup[] = ["combat", "steal", "growth", "stat", "world"];

export const effectsPages: WikiPage[] = [
  {
    id: "effects",
    title: "Hiệu Ứng Cổ: Cách Đọc Toàn Bộ Hiệu Ứng",
    category: "effects",
    keywords: [
      "hiệu ứng", "effect", "trigger", "op", "operation", "target", "mục tiêu",
      "điều kiện", "condition", "native hook", "unique", "block", "hiệu ứng cổ",
      "cách đọc hiệu ứng", "chỉ số", "trạng thái",
    ],
    content: (
      <div className="space-y-6 text-xs leading-relaxed text-[var(--text-secondary)]">
        <div className="p-4 rounded-2xl bg-gradient-to-r from-emerald-950/40 via-indigo-950/30 to-purple-950/40 border border-emerald-500/25 flex gap-3 items-start">
          <BookOpen className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-serif font-bold text-emerald-200">Bộ từ điển hiệu ứng</h4>
            <p className="mt-1">
              Mỗi Cổ trong game được mô tả bằng các <strong>khối hiệu ứng</strong>. Một khối trả lời bốn câu hỏi:
              <strong> khi nào</strong> (trigger), <strong>nhắm ai</strong> (target), <strong>nếu thoả điều kiện gì</strong> (condition)
              và <strong>làm gì</strong> (operation). Hiểu bốn lớp này là đọc được mọi Cổ, kể cả Cổ mới thêm.
            </p>
          </div>
        </div>

        <section className="space-y-3">
          <SectionTitle icon={<Layers className="w-4 h-4 text-emerald-400" />}>1. Một khối hiệu ứng gồm gì?</SectionTitle>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            {[
              ["Trigger", "Thời điểm kích hoạt: vào trận, khi ra chiêu, khi bị đánh, mỗi chu kỳ...", "ON_ACTION"],
              ["Target", "Đối tượng nhận hiệu ứng: bản thân, một địch, hàng, toàn bộ địch...", "ALL_ENEMIES"],
              ["Condition", "Điều kiện phụ (không bắt buộc): % HP, số tầng, lượt, cảnh giới...", "TARGET_HP_PERCENT ≤ 0.5"],
              ["Operation", "Việc hiệu ứng làm: gây sát thương, tạo khiên, hồi máu, gắn trạng thái...", "DAMAGE"],
            ].map(([name, desc, example]) => (
              <div key={name} className={card}>
                <div className="flex items-center justify-between gap-2">
                  <span className="font-bold text-[var(--text-primary)]">{name}</span>
                  <Code>{example}</Code>
                </div>
                <p className="mt-1">{desc}</p>
              </div>
            ))}
          </div>
          <div className="bg-zinc-950/80 border border-[var(--border)] rounded-xl p-4 font-mono text-[11px] text-[var(--text-primary)] space-y-1 overflow-x-auto">
            <div className="text-[var(--text-muted)]">{"// ví dụ: một chiêu đánh toàn bộ địch, chỉ khi địch còn dưới 50% máu"}</div>
            <div>trigger: <span className="text-amber-300">ON_ACTION</span></div>
            <div>target: <span className="text-sky-300">ALL_ENEMIES</span></div>
            <div>conditions: [ TARGET_HP_PERCENT <span className="text-rose-300">LTE</span> 0.5 ]</div>
            <div>operations: [ <span className="text-emerald-300">DAMAGE</span> · sub_type <span className="text-emerald-300">FLAT</span> · value 120 ]</div>
          </div>
        </section>

        <section className="space-y-3">
          <SectionTitle icon={<Zap className="w-4 h-4 text-amber-400" />}>2. Năm nhóm thời điểm</SectionTitle>
          <Table
            columns={["Nhóm", "Ý nghĩa"]}
            rows={TIMING_GROUPS.map((g) => [
              <span key={g.id} className="font-mono text-[10px] uppercase text-[var(--text-primary)]">{g.label}</span>,
              g.summary,
            ])}
          />
        </section>

        <section className="space-y-3">
          <SectionTitle icon={<BookOpen className="w-4 h-4 text-sky-400" />}>3. Thứ tự đọc một Cổ</SectionTitle>
          <ol className="list-decimal list-inside space-y-1.5">
            <li>Đọc <strong>trigger</strong> để biết hiệu ứng chạy lúc nào — đây là thứ quyết định giá trị thực tế.</li>
            <li>Đọc <strong>target</strong> để biết hiệu ứng mạnh trên một mục tiêu hay cả đội hình.</li>
            <li>Đọc <strong>operation + subtype</strong> để biết nó gây sát thương thường, sát thương trực tiếp hay chỉ tạo khiên.</li>
            <li>Đọc <strong>điều kiện, số lượt, tỉ lệ</strong> — cùng một chiêu nhưng 30% và 90% là hai Cổ khác nhau.</li>
            <li>Nếu gặp <Code>UNIQUE</Code>, tra mã ở trang <strong>Cơ Chế Đặc Biệt</strong> vì tác dụng nằm ngoài bảng op.</li>
          </ol>
        </section>

        <section className="space-y-3">
          <SectionTitle icon={<AlertTriangle className="w-4 h-4 text-rose-400" />}>4. Điều cần nhớ</SectionTitle>
          <div className="space-y-2">
            <Note tone="info">
              Cổ chỉ phát huy hiệu ứng khi <strong>được trang bị, trạng thái ACTIVE và còn độ no</strong> (trừ Cổ Năng Lực ghi rõ chỉ cần nằm trong túi).
            </Note>
            <Note>
              <strong>Chênh cảnh giới:</strong> sát thương và sát thương trực tiếp bị giảm khi cảnh giới mục tiêu cao hơn cảnh giới Cổ.
              Tỉ lệ gắn trạng thái được cộng thêm 5% mỗi rank Cổ, trần 95%.
            </Note>
            <Note>
              <strong>Bất Hoại</strong> chặn mọi sát thương (kể cả % MaxHP); <strong>Bất Tử</strong> vẫn nhận sát thương nhưng HP không xuống dưới 1.
              Chỉ một số cơ chế đặc biệt như Sát Khí trên 10 tầng hoặc Chiến Thắng Tuyệt Đối mới đi xuyên qua.
            </Note>
          </div>
        </section>
      </div>
    ),
    cards: [
      { title: "Trigger — Khi Nào Kích Hoạt", targetId: "effects-triggers", description: "15 thời điểm kích hoạt và ví dụ." },
      { title: "Operation — Hiệu Ứng Làm Gì", targetId: "effects-operations", description: "9 loại op, subtype và tương tác." },
      { title: "Mục Tiêu & Điều Kiện", targetId: "effects-targets-conditions", description: "Target, condition, unit, cộng dồn." },
      { title: "Chỉ Số & Tài Nguyên", targetId: "effects-stats-resources", description: "24 chỉ số và 9 loại tài nguyên." },
      { title: "Cơ Chế Đặc Biệt (UNIQUE)", targetId: "effects-special", description: "Toàn bộ native hook đang có trong game." },
      { title: "Trạng Thái", targetId: "effects-statuses", description: "Trạng thái combat và trạng thái thường trực." },
    ],
  },

  {
    id: "effects-triggers",
    title: "Trigger — Khi Nào Hiệu Ứng Kích Hoạt",
    category: "effects",
    subCategory: "effects",
    keywords: ["trigger", "kích hoạt", "on action", "on hit", "on battle start", "on tick", "always on", "capability", "on consume", "on death"],
    content: (
      <div className="space-y-5 text-xs leading-relaxed text-[var(--text-secondary)]">
        <p>
          Trigger là <strong>thời điểm</strong> hiệu ứng chạy. Cùng một operation nhưng đặt sai trigger sẽ có giá trị rất khác:
          <Code>DAMAGE</Code> ở <Code>ON_ACTION</Code> là chiêu đánh, còn ở <Code>ON_TICK</Code> gần như vô nghĩa trong trận.
        </p>
        {TIMING_GROUPS.map((group) => (
          <section key={group.id} className="space-y-2">
            <SectionTitle icon={<Zap className="w-4 h-4 text-emerald-400" />}>{group.label}</SectionTitle>
            <p className="text-[11px] text-[var(--text-muted)]">{group.summary}</p>
            <Table
              columns={["Mã", "Tên", "Khi nào chạy", "Ghi chú"]}
              rows={EFFECT_TRIGGERS.filter((t) => t.group === group.id).map((t) => [
                <Code key={t.code}>{t.code}</Code>,
                <span key={`${t.code}-l`} className="text-[var(--text-primary)]">{t.label}</span>,
                t.when,
                t.note || "—",
              ])}
            />
          </section>
        ))}
        <Note tone="info">
          <strong>Dễ nhầm:</strong> <Code>ON_ACTION</Code> là lúc Cổ sư ra chiêu; <Code>ON_CAST</Code> là lúc chuẩn bị thôi động Cổ;
          <Code>ON_HIT</Code> chỉ chạy khi đòn đã trúng; <Code>ON_BATTLE_START</Code> chỉ chạy đúng một lần trước Lượt 1.
        </Note>
      </div>
    ),
  },

  {
    id: "effects-operations",
    title: "Operation — Hiệu Ứng Làm Gì",
    category: "effects",
    subCategory: "effects",
    keywords: ["operation", "op", "damage", "shield", "heal", "direct hp", "stat mod", "status", "cleanse", "barrier", "sát thương", "khiên", "hồi máu"],
    content: (
      <div className="space-y-5 text-xs leading-relaxed text-[var(--text-secondary)]">
        <p>
          Operation là <strong>hành động</strong> của khối hiệu ứng. Mỗi op có bộ <Code>sub_type</Code> riêng; đọc đúng subtype mới biết
          con số <Code>value</Code> là điểm, % hay hệ số nhân.
        </p>
        {EFFECT_OPERATIONS.map((op) => (
          <section key={op.code} className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <Code>{op.code}</Code>
              <span className="font-serif font-bold text-[var(--text-primary)]">{op.label}</span>
            </div>
            <p>{op.summary}</p>
            {op.subTypes.length > 0 && (
              <Table
                columns={["Sub-type", "Nghĩa", "Cách đọc value"]}
                rows={op.subTypes.map((st) => [
                  <Code key={st.code}>{st.code}</Code>,
                  <span key={`${st.code}-l`} className="text-[var(--text-primary)]">{st.label}</span>,
                  st.note,
                ])}
              />
            )}
            <div className="flex flex-wrap items-center gap-1.5 text-[10px] text-[var(--text-muted)]">
              <span className="uppercase font-mono">Trường dùng được:</span>
              {op.fields.map((f) => <Code key={f}>{f}</Code>)}
            </div>
            {op.note && <p className="text-[11px] text-amber-200/90">{op.note}</p>}
          </section>
        ))}
        <section className="space-y-3">
          <SectionTitle icon={<Swords className="w-4 h-4 text-rose-400" />}>Tương tác giữa các op</SectionTitle>
          <div className="space-y-2">
            <Note tone="info"><strong>Sát thương thường</strong> bị phòng thủ, khiên và chênh cảnh giới giảm. <strong>Sát thương trực tiếp</strong> (<Code>DIRECT_HP</Code>) bỏ qua phòng thủ và thường bỏ qua khiên, chỉ bị Bất Hoại chặn hoặc Bất Tử giữ lại 1 HP.</Note>
            <Note><Code>MAX_HP_PERCENT</Code> dựa trên máu tối đa của mục tiêu — đánh quái/boss máu cao rất mạnh, nhưng vẫn chịu suy giảm cảnh giới.</Note>
            <Note tone="info"><Code>SHIELD</Code> hấp thụ trước HP và thường ngắn hạn; <Code>BARRIER</Code> kéo dài nhiều lượt và có thể tính theo % MaxHP.</Note>
          </div>
        </section>
      </div>
    ),
  },

  {
    id: "effects-targets-conditions",
    title: "Mục Tiêu, Điều Kiện & Cách Cộng Dồn",
    category: "effects",
    subCategory: "effects",
    keywords: ["target", "mục tiêu", "condition", "điều kiện", "stack", "cộng dồn", "unit", "percent", "flat", "bypass frontline"],
    content: (
      <div className="space-y-5 text-xs leading-relaxed text-[var(--text-secondary)]">
        <section className="space-y-2">
          <SectionTitle icon={<Target className="w-4 h-4 text-sky-400" />}>Target — Nhắm Ai</SectionTitle>
          <Table
            columns={["Mã", "Nghĩa", "Ghi chú"]}
            rows={EFFECT_TARGETS.map((t) => [<Code key={t.code}>{t.code}</Code>, <span key={`${t.code}-l`} className="text-[var(--text-primary)]">{t.label}</span>, t.note])}
          />
          <p className="text-[11px] text-[var(--text-muted)]">
            <Code>bypass_frontline</Code>: bỏ qua hàng trước để chọn thẳng mục tiêu phía sau.
          </p>
        </section>

        <section className="space-y-2">
          <SectionTitle icon={<Activity className="w-4 h-4 text-emerald-400" />}>Condition — Nếu Thoả Điều Kiện</SectionTitle>
          <Table
            columns={["Trường", "Nghĩa", "Ghi chú"]}
            rows={EFFECT_CONDITIONS.map((c) => [<Code key={c.code}>{c.code}</Code>, <span key={`${c.code}-l`} className="text-[var(--text-primary)]">{c.label}</span>, c.note])}
          />
          <Table
            columns={["Toán tử", "Nghĩa", "Ký hiệu"]}
            rows={EFFECT_CONDITION_OPERATORS.map((o) => [<Code key={o.code}>{o.code}</Code>, <span key={`${o.code}-l`} className="text-[var(--text-primary)]">{o.label}</span>, o.note])}
          />
        </section>

        <section className="space-y-2">
          <SectionTitle icon={<BarChart3 className="w-4 h-4 text-amber-400" />}>Unit — Đơn Vị Giá Trị</SectionTitle>
          <Table
            columns={["Mã", "Nghĩa", "Ghi chú"]}
            rows={EFFECT_UNITS.map((u) => [<Code key={u.code}>{u.code}</Code>, <span key={`${u.code}-l`} className="text-[var(--text-primary)]">{u.label}</span>, u.note])}
          />
        </section>

        <section className="space-y-2">
          <SectionTitle icon={<Layers className="w-4 h-4 text-purple-400" />}>Stack Policy — Cộng Dồn</SectionTitle>
          <Table
            columns={["Mã", "Nghĩa", "Ghi chú"]}
            rows={EFFECT_STACK_POLICIES.map((s) => [<Code key={s.code}>{s.code}</Code>, <span key={`${s.code}-l`} className="text-[var(--text-primary)]">{s.label}</span>, s.note])}
          />
          <Note tone="info">
            Nhiều buff cùng loại không cộng tuyến tính: <Code>MAX</Code> chỉ giữ cái mạnh nhất, <Code>REFRESH</Code> làm mới thời hạn,
            <Code>STACK</Code> mới thực sự cộng tầng.
          </Note>
        </section>
      </div>
    ),
  },

  {
    id: "effects-stats-resources",
    title: "Chỉ Số & Tài Nguyên",
    category: "effects",
    subCategory: "effects",
    keywords: ["stat", "chỉ số", "resource", "tài nguyên", "attack", "defense", "hp", "speed", "mana", "soul", "thought", "luck"],
    content: (
      <div className="space-y-5 text-xs leading-relaxed text-[var(--text-secondary)]">
        <section className="space-y-2">
          <SectionTitle icon={<BarChart3 className="w-4 h-4 text-emerald-400" />}>24 Chỉ Số</SectionTitle>
          <Table
            columns={["Mã", "Tên", "Tác dụng"]}
            rows={EFFECT_STATS.map((s) => [<Code key={s.code}>{s.code}</Code>, <span key={`${s.code}-l`} className="text-[var(--text-primary)] whitespace-nowrap">{s.label}</span>, s.note])}
          />
        </section>
        <section className="space-y-2">
          <SectionTitle icon={<Sparkles className="w-4 h-4 text-sky-400" />}>9 Loại Tài Nguyên</SectionTitle>
          <Table
            columns={["Mã", "Tên", "Tác dụng"]}
            rows={EFFECT_RESOURCES.map((r) => [<Code key={r.code}>{r.code}</Code>, <span key={`${r.code}-l`} className="text-[var(--text-primary)] whitespace-nowrap">{r.label}</span>, r.note])}
          />
        </section>
        <Note>
          Cổ dùng <Code>op: STAT_MOD</Code> với <Code>stat</Code> tương ứng. Một số mã native hook chỉ số (STAT_MODS, LUCK, CC_RESIST...)
          được khai báo <strong>trên Cổ</strong> nhưng hiện chỉ đọc từ chủng tộc/thể chất/trạng thái — xem trang Cơ Chế Đặc Biệt.
        </Note>
      </div>
    ),
  },

  {
    id: "effects-special",
    title: "Cơ Chế Đặc Biệt (UNIQUE) — Toàn Bộ Mã",
    category: "effects",
    subCategory: "effects",
    keywords: ["unique", "native hook", "cơ chế đặc biệt", "hồi sinh", "đoạt tiên cơ", "triệu hồi", "ngụy trang", "cổng", "chợ", "chat"],
    content: (
      <div className="space-y-5 text-xs leading-relaxed text-[var(--text-secondary)]">
        <p>
          Op <Code>UNIQUE</Code> gọi một mã <strong>native hook</strong> — hành vi do engine/handler xử lý chứ không nằm trong bảng op.
          Dưới đây là toàn bộ mã đang xuất hiện trong dữ liệu Cổ. Cột <strong>Trạng thái</strong> cho biết mã đó có tác dụng thật hay chưa.
        </p>
        <div className="flex flex-wrap gap-1.5">
          {(Object.keys(HOOK_STATUS_LABELS) as HookStatus[]).map((s) => (
            <span key={s} className="flex items-center gap-1.5">
              <StatusBadge status={s} />
              <span className="text-[10px] text-[var(--text-muted)]">
                {s === "active" && "đã chạy trong game"}
                {s === "partial" && "có điều kiện hoặc thiếu phần phụ"}
                {s === "legacy" && "tên cũ, vẫn dùng được"}
                {s === "unimplemented" && "khai báo nhưng chưa có hiệu lực"}
              </span>
            </span>
          ))}
        </div>
        {HOOK_GROUP_ORDER.map((group) => {
          const hooks = NATIVE_HOOKS.filter((h) => h.group === group);
          return (
            <section key={group} className="space-y-2">
              <SectionTitle icon={<Sparkles className="w-4 h-4 text-emerald-400" />}>{HOOK_GROUP_LABELS[group]}</SectionTitle>
              <Table
                columns={["Mã", "Tên", "Trigger", "Hiệu ứng", "Trạng thái"]}
                rows={hooks.map((h) => [
                  <Code key={h.code}>{h.code}</Code>,
                  <span key={`${h.code}-l`} className="text-[var(--text-primary)] whitespace-nowrap">{h.label}</span>,
                  <Code key={`${h.code}-t`}>{h.trigger}</Code>,
                  h.behavior,
                  <StatusBadge key={`${h.code}-s`} status={h.status} />,
                ])}
              />
            </section>
          );
        })}
      </div>
    ),
  },

  {
    id: "effects-statuses",
    title: "Trạng Thái Trong Game",
    category: "effects",
    subCategory: "effects",
    keywords: ["trạng thái", "status", "buff", "debuff", "choáng", "độc", "chảy máu", "bất hoại", "bất tử", "áp chế giới vực"],
    content: (
      <div className="space-y-5 text-xs leading-relaxed text-[var(--text-secondary)]">
        <section className="space-y-2">
          <SectionTitle icon={<Swords className="w-4 h-4 text-rose-400" />}>Trạng Thái Combat</SectionTitle>
          <p className="text-[11px] text-[var(--text-muted)]">
            Gắn bởi op <Code>STATUS</Code>. Số lượt ghi trong dữ liệu Cổ có thể khác mặc định dưới đây.
          </p>
          <Table
            columns={["Mã", "Tên", "Nhóm", "Hiệu ứng", "Lượt mặc định"]}
            rows={COMBAT_STATUSES.map((s) => [
              <Code key={s.id}>{s.id}</Code>,
              <span key={`${s.id}-l`} className="text-[var(--text-primary)] whitespace-nowrap">{s.label}</span>,
              s.category,
              s.behavior,
              s.turns,
            ])}
          />
        </section>

        <section className="space-y-2">
          <SectionTitle icon={<Activity className="w-4 h-4 text-emerald-400" />}>Trạng Thái Thường Trực</SectionTitle>
          <p className="text-[11px] text-[var(--text-muted)]">
            Gắn ngoài trận và tồn tại theo thời gian thực; hiển thị trong hồ sơ nhân vật.
          </p>
          <Table
            columns={["Tên", "Hiệu ứng", "Thời hạn", "Chặn hành động"]}
            rows={PERSISTENT_STATUSES.map((s) => [
              <span key={s.key} className="text-[var(--text-primary)] whitespace-nowrap">{s.name}</span>,
              s.description,
              s.duration,
              s.blocked || "—",
            ])}
          />
        </section>
        <Note tone="info">
          Trạng thái xấu có thể bị <strong>kháng</strong> (Kháng Khống Chế, Kháng Tinh Thần) hoặc bị <strong>giải</strong> (Thanh Tẩy, Kiên Cường).
          <Code>Bất Hoại</Code> và <Code>Bất Tử</Code> là hai trạng thái khác nhau — đọc kỹ trước khi lao vào đối thủ.
        </Note>
      </div>
    ),
  },
];
