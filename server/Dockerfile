# Multi-stage build for the Yeonly Go backend.
FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/yeonly-server ./cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates \
    && adduser -D -u 1001 appuser

USER appuser
COPY --from=builder /out/yeonly-server /usr/local/bin/yeonly-server

ENV PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT}/health" || exit 1

CMD ["yeonly-server"]
