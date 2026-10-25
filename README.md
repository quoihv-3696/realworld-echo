# RealWorld API — Echo v5 + GORM + PostgreSQL

Backend cho [RealWorld](https://github.com/realworld-apps/realworld) ("Conduit", một bản sao Medium), viết bằng Go với Echo v5, GORM và PostgreSQL. Chạy xanh **toàn bộ 13 file test chính thức** (`tests/hurl`, 154 request) cùng bộ test Go chạy `-race`.

## Công nghệ

| Phần | Dùng |
|---|---|
| HTTP | [Echo v5](https://github.com/labstack/echo) |
| ORM, DB | GORM, PostgreSQL 17 |
| Xác thực | JWT HS256 (`golang-jwt/v5`, `echo-jwt/v5`), header `Authorization: Token <jwt>`; mật khẩu bcrypt |
| Cache | Redis 7 (`go-redis/v9`), cache-aside có version; Redis tắt thì app vẫn chạy |
| Cấu hình | biến môi trường, nạp thêm từ `.env` (`godotenv`) |
| Tài liệu API | `api/openapi.yml` chính thức + Swagger UI |

## Chạy thử

Cần Go 1.27+, Docker, và [hurl](https://hurl.dev) nếu muốn chạy test chính thức.

```bash
docker compose up -d --wait   # PostgreSQL (cổng 5433) + Redis (cổng 6380), chỉ mở trên 127.0.0.1
cp .env.example .env          # sửa JWT_SECRET nếu dùng ngoài máy dev
go run ./cmd/api              # http://localhost:8080
```

Lần chạy đầu, server tự tạo bảng và thêm dữ liệu mẫu: user `jake`, `anna`, `minh` (email `<tên>@example.com`, mật khẩu `password123`) cùng 5 bài viết.

- Ngoài spec RealWorld, `GET /api/articles` nhận thêm `?sort=newest|oldest|popular` (mặc định `newest`; `popular` = nhiều lượt thích trước; giá trị khác → 422). Feed bỏ qua `sort`.
- Swagger UI: http://localhost:8080/docs ("Try it out" gửi request thẳng tới server này; giao diện tải từ CDN jsDelivr nên cần mạng)
- Đặc tả OpenAPI: http://localhost:8080/openapi.yml

## Cấu hình

Biến môi trường thật luôn thắng giá trị trong `.env`.

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `PORT` | `8080` | cổng HTTP |
| `DATABASE_URL` | `postgres://realworld:realworld@localhost:5433/realworld?sslmode=disable` | PostgreSQL |
| `JWT_SECRET` | khóa dev công khai; server ghi cảnh báo khi đang dùng nó | ký JWT; máy thật phải đặt riêng, tối thiểu 32 byte |
| `REDIS_URL` | trống: không cache | ví dụ `redis://localhost:6380/0` |
| `CACHE_TTL` | `60s` | thời gian sống của dữ liệu cache |
| `TEST_DATABASE_URL` | `.../realworld_test` | DB cho `go test` (đặt trong shell: `go test` không đọc `.env`); test từ chối chạy nếu tên DB không có đuôi `_test` |
| `TEST_REDIS_URL` | `redis://localhost:6380/15` | Redis cho test của package `cache` (đặt trong shell) |

## Kiểm thử

```bash
go test -race ./...          # cần PostgreSQL và Redis của docker compose
scripts/check-day.sh 6       # 13 file hurl chính thức vào server đang chạy (mở terminal khác)
```

`check-day.sh` sinh `uid` mới cho mỗi lần chạy, nên chạy lại bao nhiêu lần cũng được, không cần xóa DB.

Ba tầng test:

| Tầng | Cách | Ở đâu |
|---|---|---|
| Unit | service với repository giả; handler và middleware gọi thẳng qua `e.NewContext` (không router; riêng test `RequireOwner` dùng DB test) | `internal/services/*_test.go`, `internal/handlers/*_test.go` |
| Tích hợp | request `httptest` đi qua router, middleware, error handler, DB test thật | `internal/server/*_test.go`, `internal/repository/*_test.go` |
| Đầu-cuối | 13 file hurl chính thức vào server đang chạy | `tests/hurl/` |

## Cấu trúc

```
cmd/api/            main: đọc cấu hình, mở DB, migrate, seed, dựng cache, chạy server, tắt êm khi Ctrl+C
api/                openapi.yml chính thức + trang Swagger UI (nhúng vào binary bằng go:embed)
internal/
  server/           đăng ký route, middleware, nối các lớp với nhau
  handlers/         đọc request (Bind, Param, Query), gọi service, trả JSON; middleware RequireOwner
  services/         nghiệp vụ, kiểm tra dữ liệu, model → DTO, cache-aside
  repository/       truy vấn GORM; đổi lỗi GORM thành lỗi của repository
  models/           struct GORM (bảng); dto/: struct JSON theo spec RealWorld
  apperr/           lỗi dùng chung + HTTPErrorHandler: mọi lỗi → {"errors": {...}}
  auth/             bcrypt, JWT; cache/: Redis + cache-aside; config/: biến môi trường
  testutil/         DB test, request giả, đếm câu SQL, cache giả trong bộ nhớ
tests/hurl/         13 file test chính thức của RealWorld (chép nguyên, giấy phép MIT, xem NOTICE)
```

Lớp trên chỉ biết lớp ngay dưới: handler không biết GORM, service không biết Echo hay Redis, repository không biết HTTP.

## Đối chiếu lộ trình 8 phần

| Phần | Nội dung | Ở đâu |
|---|---|---|
| 1. Setup | Echo, GORM, PostgreSQL, cấu trúc thư mục, model User/Article/Comment/Tag | `cmd/api`, `internal/models`, `docker-compose.yml` |
| 2. Handler, binding | `c.Bind`, `c.JSON`, Group route; `GET` articles, article, tags | `internal/handlers`, `internal/server` |
| 3. Quan hệ, DTO | BelongsTo, HasMany, Many2Many, Preload, DTO ẩn password; comments, profile | `internal/models`, `internal/dto`, `internal/repository` |
| 4. JWT | bcrypt, middleware JWT, đăng ký, đăng nhập, user hiện tại | `internal/auth`, `internal/services/user_service.go` |
| 5. Middleware, lọc, phân trang | middleware quyền tác giả, `tag`/`author`/`favorited`, sắp xếp `sort` (whitelist), `limit`/`offset`, feed | `internal/handlers/article_owner.go`, `internal/repository/article_repository.go` |
| 6. Social | follow, favorite, chỉ chính chủ xóa bình luận | `follow_repository.go`, `favorite_repository.go`, `comment_service.go` |
| 7. Test, tối ưu | unit test handler kiểu Echo Testing (`e.NewContext`), test qua `httptest`, Redis cho tags và danh sách bài, `.env` | `internal/handlers/*_test.go`, `internal/server/*_test.go`, `internal/cache`, `internal/config` |
| 8. Tổng kết | test cả luồng, Swagger, tránh N+1 | `TestFullFlow_RegisterToFeed`, `/docs`, `testutil.QueryCounter` |

## Ghi chú thiết kế

- **Thứ tự làm khác lộ trình:** xác thực (phần 4) làm trước phần 3, vì mọi file test chính thức đều bắt đầu bằng việc đăng ký tài khoản; follow (phần 6) làm cùng feed (phần 5), vì `feed.hurl` cần follow.
- **Lỗi:** mọi lỗi trả về `{"errors": {"<field>": ["<message>"]}}`. 401 thiếu hoặc sai token, 403 không phải chủ, 404 không có, 409 trùng, 422 dữ liệu không hợp lệ, 400 không đọc được request.
- **Không N+1:** `favorited`, `favoritesCount`, `following` của cả một trang được nạp bằng một câu cho mỗi loại. Một trang bài chạy 8 câu SQL khi đăng nhập, 6 câu khi là khách, dù trang có 2 hay 20 bài; có test đếm số câu.
- **Cache:** chỉ cache dữ liệu không phụ thuộc người xem: danh sách tag, và danh sách bài khi không đăng nhập. Mỗi lần ghi (tạo/sửa/xóa bài, thích/bỏ thích, sửa thông tin user) tăng một số version, nên mọi key cũ bị bỏ qua ngay và tự hết hạn theo TTL. Redis lỗi hay tắt thì đọc thẳng DB. Redis trong `docker-compose.yml` giới hạn 64 MB với chính sách `volatile-lru`: đầy thì chỉ đẩy ra dữ liệu cache (có TTL), không đẩy `cache:version`.
