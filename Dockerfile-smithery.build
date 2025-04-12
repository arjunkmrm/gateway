#############################################
# Builder stage: build in Debian environment
#############################################
FROM golang:1.24-bullseye AS builder

WORKDIR /app

# Install necessary build tools and UPX
RUN apt-get update && \
    apt-get install -y --no-install-recommends git gcc build-essential upx && \
    rm -rf /var/lib/apt/lists/*

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with CGO enabled (required for duckdb-go-bindings)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o gateway

# Compress binary using UPX
RUN upx --best --lzma gateway

#############################################
# Final stage: minimal Debian-based image
#############################################
FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND=noninteractive

# Install tzdata and ca-certificates, configure timezone
RUN apt-get update && \
    apt-get install -y --no-install-recommends tzdata ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    ln -fs /usr/share/zoneinfo/Etc/UTC /etc/localtime && \
    echo "Etc/UTC" > /etc/timezone && \
    groupadd --system gateway && \
    useradd --system --create-home --home-dir /home/gateway --gid gateway gateway

# Set necessary environment variables
ENV TZ=Etc/UTC
ENV ROTATION_TZ=Etc/UTC
ENV HOME=/home/gateway

# Copy compiled binary from build stage
COPY --from=builder /app/gateway /usr/local/bin/gateway

# Switch to non-privileged user
USER gateway

ENTRYPOINT ["/usr/local/bin/gateway"]
