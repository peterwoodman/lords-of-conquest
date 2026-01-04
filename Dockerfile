# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install litestream
RUN apk add --no-cache ca-certificates wget \
    && wget -O /tmp/litestream.tar.gz https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.tar.gz \
    && tar -xzf /tmp/litestream.tar.gz -C /usr/local/bin \
    && rm /tmp/litestream.tar.gz

# Copy binary and configs
COPY --from=builder /build/server /app/server
COPY litestream.yml /app/litestream.yml
COPY scripts/start-server.sh /app/start-server.sh

# Make scripts executable
RUN chmod +x /app/start-server.sh /app/server

# Create data directory
RUN mkdir -p /data

# Set environment defaults
ENV DB_PATH=/data/lords.db
ENV PORT=10000

EXPOSE 10000

CMD ["/app/start-server.sh"]
