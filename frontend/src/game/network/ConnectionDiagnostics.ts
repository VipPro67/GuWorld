/** Health over HTTPS cannot prove that QUIC/HTTP/3 works. Never infer TLS or CORS from fetch failure. */
export async function diagnoseConnectionFailure(error: unknown, endpoint: string): Promise<string> {
  const raw = error instanceof Error ? `${error.name}: ${error.message}` : String(error);
  let url: URL;
  try {
    url = new URL(endpoint);
    url.search = "";
    url.hash = "";
    url.username = "";
    url.password = "";
  } catch {
    return "Địa chỉ WebTransport không hợp lệ. Kiểm tra NEXT_PUBLIC_REALTIME_URL.";
  }
  const origin = typeof location === "undefined" ? "không xác định" : location.origin;
  const details = [
    "Chưa thiết lập được kết nối WebTransport với backend.",
    `Endpoint: ${url.href}`,
    `Origin frontend: ${origin}`,
    `Lỗi trình duyệt: ${raw.replace(/([?&](?:token|tab_id)=)[^\s&#]*/gi, "$1[ẩn]")}`,
  ];
  if (typeof isSecureContext !== "undefined" && !isSecureContext) {
    return [...details, "Trang frontend không chạy trong secure context. Mở frontend qua HTTPS hoặc localhost."].join("\n");
  }
  const health = new URL("/api/health", url);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 2000);
  try {
    const response = await fetch(health.href, {
      signal: controller.signal, cache: "no-store", credentials: "omit", redirect: "error",
    });
    if (response.ok) {
      const body: unknown = await response.json().catch(() => null);
      const validHealth = typeof body === "object" && body !== null && "status" in body && body.status === "ok";
      details.push(`Kiểm tra HTTPS: ${health.href} trả HTTP ${response.status}.`,
        validHealth
          ? "Backend báo health OK qua HTTPS; kết nối WebTransport vẫn thất bại."
          : "Phản hồi không có status=ok của health API; có thể URL đang trỏ tới proxy hoặc route khác.",
        `Kiểm tra log backend cho endpoint ${url.pathname}, quyền chấp nhận Origin ${origin}, chứng chỉ dùng cho QUIC và HTTP/3 qua UDP ${url.port || "443"}. HTTPS thành công chưa xác nhận HTTP/3 hoạt động.`);
    } else {
      details.push(`Kiểm tra HTTPS: ${health.href} trả HTTP ${response.status}.`,
        "Máy chủ/proxy có trả lời nhưng health không thành công. Kiểm tra route health và log backend/proxy; mã HTTP này thuộc health, không phải mã lỗi bắt tay WebTransport.");
    }
  } catch {
    details.push(controller.signal.aborted
      ? `Kiểm tra HTTPS: ${health.href} không phản hồi trong 2 giây.`
      : `Kiểm tra HTTPS: trình duyệt không đọc được ${health.href}.`,
    "Chưa thể xác định backend đã dừng, lỗi mạng, TLS hay CORS. Mở URL health trong tab mới để kiểm tra chứng chỉ và phản hồi; kiểm tra Console/Network cùng log backend. Không thể xác nhận UDP bị chặn từ lỗi này.");
  } finally {
    clearTimeout(timeout);
  }
  return details.join("\n");
}
