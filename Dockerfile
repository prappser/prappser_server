# Build stage - compile Go binary with CGO for SQLite
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o prappser_server .

# Runtime stage - minimal Alpine image
# Railway volumes handle persistence, no Litestream needed
FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

# Copy application binary
COPY --from=builder /build/prappser_server .

# Copy migrations
COPY files/migrations ./files/migrations

# Create data directory for SQLite database
RUN mkdir -p /app/files

EXPOSE 4545

CMD ["./prappser_server"]
