# Stage 1: Build Frontend (Svelte 5 / Vite)
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Backend (Go)
FROM golang:alpine AS backend-builder
WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server/yeonly-server ./cmd/server

# Stage 3: Runner
FROM alpine:3.21 AS runner
RUN apk --no-cache add ca-certificates tzdata wget
RUN adduser -D -u 1001 appuser

WORKDIR /app
COPY --from=frontend-builder /app/web/dist ./web/dist
COPY --from=backend-builder /app/server/yeonly-server ./yeonly-server

RUN chown -R appuser:appuser /app
USER appuser

ENV PORT=8080
ENV WEB_DIST=/app/web/dist
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["/app/yeonly-server"]
