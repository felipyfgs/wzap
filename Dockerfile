FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /app/wzap cmd/wzap/main.go

# --- runtime ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl ffmpeg

RUN addgroup -S wzap && adduser -S wzap -G wzap -h /app

WORKDIR /app

COPY --from=builder /app/wzap /app/wzap

RUN chown -R wzap:wzap /app

USER wzap

ENV PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    ENVIRONMENT=production

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${PORT}/health || exit 1

ENTRYPOINT ["/app/wzap"]
