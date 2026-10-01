export default function AdminLoading() {
  return (
    <div className="min-h-screen animate-pulse bg-[var(--bg-primary)] p-6" aria-label="Đang tải trang quản trị">
      <div className="mx-auto max-w-6xl space-y-5">
        <div className="h-16 rounded-xl bg-zinc-900" />
        <div className="h-10 w-2/3 rounded-lg bg-zinc-900" />
        <div className="h-72 rounded-xl bg-zinc-900/70" />
      </div>
    </div>
  );
}
