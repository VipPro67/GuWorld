# Ma trận kiểm thử Admin RBAC

Ma trận này là contract tối thiểu cho phân quyền quản trị. Quyền được resolve từ
database ở **mỗi request**, không lưu trong JWT.

| Actor | Resource chính được thấy | Route đại diện | Kết quả |
|---|---|---|---|
| Anonymous | Không có | mọi `/api/admin/*` | `401` |
| Player thường | Không có | mọi `/api/admin/*` | `403` |
| Observer | Các resource read-only, trừ billing | `GET /players` / `PUT /players/:id` | `2xx` / `403` |
| Support admin | players, Gu history, audit | update player / import data | `2xx` / `403` |
| Content admin | game data và dictionary/content | import data / update player | `2xx` / `403` |
| Game admin | game/world operations | update mine / import data | `2xx` / `403` |
| Finance viewer | billing | list topup / update player | `2xx` / `403` |
| Super admin | Toàn bộ catalog | mọi admin route | `2xx` (nếu payload hợp lệ) |

## Checklist bắt buộc

- Backend matrix phải kiểm tra `401`, `403`, `2xx` và xác nhận handler không
  chạy khi gọi trực tiếp endpoint bị cấm.
- Frontend matrix phải kiểm tra navigation và action bị ẩn theo permission;
  tên role không được dùng để quyết định UI.
- Dùng cùng session: request trước khi revoke thành công, request kế tiếp sau
  revoke trả `403`; không yêu cầu đăng nhập lại.
- Demote super admin cuối cùng trả `409`.
- Hai request đồng thời demote hai super admin phải được serialize: đúng một
  request thành công, request còn lại trả `409`, và database luôn còn ít nhất
  một `super_admin`.

Các test unit chạy trong `go test ./...` và `npm test`. Hai invariant cuối được
bảo vệ trong production bằng transaction, PostgreSQL advisory transaction lock
và kiểm tra số lượng dưới cùng lock.
