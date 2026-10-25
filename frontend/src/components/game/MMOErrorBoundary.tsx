"use client";

import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

/** Keeps a Phaser failure from taking the legacy React UI down with it. */
export class MMOErrorBoundary extends Component<Props, State> {
  public state: State = { error: null };

  public static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  public componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error("MMO world renderer failed", error, info.componentStack);
  }

  public render(): ReactNode {
    if (!this.state.error) return this.props.children;
    return (
      <div className="flex h-full min-h-[320px] w-full flex-col items-center justify-center gap-3 bg-zinc-950 text-center text-zinc-200">
        <div className="font-serif text-lg text-emerald-200">
          Không thể khởi động thế giới realtime
        </div>
        <p className="max-w-md text-sm text-zinc-400">
          Trình duyệt hoặc WebGL gặp lỗi khi tải Phaser. Thế giới cũ vẫn hoạt
          động bình thường.
        </p>
        <pre className="max-w-lg overflow-x-auto rounded bg-black/40 p-3 text-left text-xs text-red-300">
          {this.state.error.message}
        </pre>
        <button
          type="button"
          onClick={() => this.setState({ error: null })}
          className="rounded-lg border border-emerald-400/40 bg-emerald-400/10 px-4 py-2 text-sm text-emerald-100 transition hover:bg-emerald-400/20"
        >
          Thử lại
        </button>
      </div>
    );
  }
}
