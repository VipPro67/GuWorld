# Choose a visual direction and icon readability rules

Type: grilling
Status: resolved
Blocked by: none

Part of [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433); canonical ticket: [Choose a visual direction and icon readability rules](https://github.com/VipPro67/GuWorld/issues/434).

## Question

Which visual language should the new and replacement gameplay icons follow, using the supplied atlases and current GuWorld art as references? Decide pixel scale/detail, palette and contrast, outline/shading treatment, background/transparency, and how icons should remain distinguishable at the smallest inventory and map sizes. Identify whether the target is close to the detailed legacy atlas, the cleaner V2 atlas, or a consistent hybrid, and define how to keep new art coherent with existing assets.

## Resolution

- Chọn phong cách hybrid, ưu tiên độ rõ và bảng màu của atlas V2; chỉ giữ mức chi tiết cao hơn của atlas cũ khi vẫn nhất quán với silhouette/palette chung.
- Ở kích thước gameplay nhỏ, ưu tiên silhouette dễ phân biệt, ít chi tiết li ti; kích thước nhỏ nhất phải đọc rõ là 24 px (đã được người dùng xác nhận).
- Dùng nền trong suốt, không gắn khung cố định vào bản thân icon; giao diện chịu trách nhiệm về frame/rarity treatment.
- Giữ pixel-art và tương phản đủ mạnh để chủ thể nổi trên nền UI sáng/tối. Ở cỡ lớn có thể bổ sung chi tiết bên trong silhouette, không đổi hình dạng nhận diện chính.

Không cần thêm thuật ngữ vào `GLOSSARY.md`: đây là tiêu chí visual cho effort, không tạo khái niệm gameplay mới.
