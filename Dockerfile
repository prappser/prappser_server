# Build stage - compile Go binary (no CGO needed for PostgreSQL)
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o prappser_server .

# Runtime stage - minimal Alpine image
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy application binary
COPY --from=builder /build/prappser_server .

# Copy migrations
COPY files/migrations ./files/migrations

EXPOSE 4545

CMD ["./prappser_server"]
