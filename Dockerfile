# Stage 1: Build with CGO enabled
FROM golang:1.24 AS builder

WORKDIR /app

# Pre-cache modules
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Enable CGO for SQLite support
RUN apt-get update && apt-get install -y gcc libc6-dev
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o chat .

# Stage 2: Use slim Debian image (not Alpine)
FROM debian:bookworm-slim

WORKDIR /root/

# Install runtime deps
RUN apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*

# Copy built binary and static files
COPY --from=builder /app/chat .
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./chat"]
