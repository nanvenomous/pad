# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache curl unzip bash libgcc libstdc++

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Install bun
ENV BUN_INSTALL=/usr/local/bun
RUN curl -fsSL https://bun.sh/install | bash
ENV PATH="/usr/local/bun/bin:/go/bin:$PATH"

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

COPY package.json bun.lockb ./
RUN bun install

# Copy source code
COPY . .

# Build frontend assets
RUN bun run tw
RUN bun run js

# Generate templ files
RUN templ generate ./ui/...

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o pad ./main.go

# Production stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/pad .

# Create notes directory
RUN mkdir -p /data/notes

# Set environment variables
ENV PAD_NOTES_DIR=/data/notes
ENV PORT=4000

EXPOSE 4000

# Run as non-root user
RUN addgroup -g 1000 pad && \
    adduser -D -u 1000 -G pad pad && \
    chown -R pad:pad /app /data

USER pad

CMD ["./pad", "serve"]
