# ---- Builder ----
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN set -eux; \
    n=0; \
    until [ $n -ge 5 ]; do \
        go mod download && break; \
        n=$((n+1)); \
        echo "go mod download failed, retry $n/5"; \
        sleep $((n*2)); \
    done; \
    [ $n -lt 5 ]

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/seed   ./cmd/seed && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/seedfake ./cmd/seedfake

# ---- Final ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    adduser -D -u 1000 appuser

WORKDIR /app

COPY --from=builder /out/server  ./server
COPY --from=builder /out/seed    ./seed
COPY --from=builder /out/seedfake ./seedfake
COPY --from=builder /build/views ./views
COPY entrypoint.sh               ./entrypoint.sh

# Strip CRLF (safe on Windows dev machines), set permissions
RUN sed -i 's/\r//' ./entrypoint.sh && \
    chmod +x ./server ./seed ./seedfake ./entrypoint.sh && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8000

ENTRYPOINT ["./entrypoint.sh"]
