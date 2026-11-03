# Stage 1: Build the Go binary
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

# Copy manifests để cache dependency
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary lược bỏ debug symbols
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o guworld_server ./cmd/server

# Stage 2: Runtime container
FROM alpine:3.22

WORKDIR /app

# Cài đặt chứng chỉ SSL và Timezone
RUN apk --no-cache add ca-certificates tzdata curl \
    && addgroup -S -g 10001 app \
    && adduser -S -G app -u 10001 app \
    && mkdir -p /app/storage/images /data/certs \
    && chown -R app:app /app /data/certs

ENV TZ=Asia/Ho_Chi_Minh

# Copy binary và các file tài nguyên tĩnh
COPY --from=builder /app/guworld_server .
COPY --from=builder /app/data ./data

USER app

# 8080: plain HTTP (dev/health), 8443: HTTPS over TCP and QUIC/WebTransport over UDP.
EXPOSE 8080 8443 8443/udp

CMD ["./guworld_server"]