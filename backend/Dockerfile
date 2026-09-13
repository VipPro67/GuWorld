# syntax=docker/dockerfile:1.7
# Stage 1: Build the Go binary
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

# Copy manifests để cache dependency
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build static binary siêu nhẹ với cache mount và lược bỏ debug symbols
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o guworld_server ./cmd/server

# Stage 2: Runtime container
FROM alpine:3.22

WORKDIR /app

# Cài đặt chứng chỉ SSL và Timezone
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /app/storage/images \
    && chown -R app:app /app

ENV TZ=Asia/Ho_Chi_Minh

# Copy binary và các file tài nguyên tĩnh
COPY --from=builder /app/guworld_server .
COPY --from=builder /app/data ./data
COPY --from=builder /app/google.json* ./

USER app

EXPOSE 8080

CMD ["./guworld_server"]