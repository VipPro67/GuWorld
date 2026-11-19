# Choose the asset production and delivery pipeline

Type: grilling
Status: resolved
Blocked by: [Choose a visual direction and icon readability rules](https://github.com/VipPro67/GuWorld/issues/434), [Define gameplay asset groups and coverage boundaries](https://github.com/VipPro67/GuWorld/issues/435)

Part of [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433); canonical ticket: [Choose the asset production and delivery pipeline](https://github.com/VipPro67/GuWorld/issues/436).

## Question

Given the agreed visual rules and coverage, how should assets be authored or generated, reviewed, stored, and served? Decide individual files versus atlases/sprite sheets (or a mix), source/editable artifact expectations, output formats and dimensions, transparency, naming/manifest conventions, licensing/provenance requirements, and any tooling needed to produce a maintainable library that fits current frontend usage.

## Resolution

- Mỗi asset phát hành là file riêng theo ID ổn định; WebP là định dạng runtime, PNG là master lossless để tái xuất và sửa.
- Hai atlas được chia sẻ chỉ dùng làm tham chiếu phong cách. Tạo artwork mới; không crop hay tái sử dụng sprite của atlas trực tiếp.
- Duyệt mẫu đại diện theo từng nhóm asset trước khi mở rộng sản xuất. Với mỗi batch, tự kiểm tra tên/ID, kích thước chuẩn và alpha/nền trong suốt; lỗi khiến batch cần sửa trước khi tích hợp.
- Tên đầu ra dựa trên ID ổn định trong catalog, không dựa vào tên hiển thị có thể đổi. Manifest đi kèm lưu ID, đường dẫn master/runtime và nguồn tạo/prompt để truy xuất, kiểm tra coverage và tái tạo batch.
