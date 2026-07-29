# Stage 1: Build the Go binary
FROM golang:alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build clean static binary
# syntax=docker/dockerfile:1.7

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -o guworld_server ./cmd/server
    
# Stage 2: Create the runner container
FROM alpine:latest

WORKDIR /app

# Install ca-certificates (needed for external integrations like PayOS and Google OAuth)
RUN apk --no-cache add ca-certificates

# Copy compiled binary and CSV data files from builder
COPY --from=builder /app/guworld_server .
COPY --from=builder /app/cmd/server/data ./data
COPY --from=builder /app/google.json* ./

EXPOSE 8080

CMD ["./guworld_server"]
