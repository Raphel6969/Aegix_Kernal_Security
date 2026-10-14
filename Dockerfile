# ==============================================================================
# Multi-stage Dockerfile for Aegix AI Kernel Security System
# Production all-in-one container for Render, Railway, Docker, or standalone.
# Hosts the Go backend API, WebSocket server, and static React dashboard SPA.
# ==============================================================================

# ── Stage 1: Build React Dashboard ──────────────────────────────────────────
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci || npm install

COPY frontend/ ./
ARG VITE_API_URL=""
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

# ── Stage 2: Build Go Backend Binary ────────────────────────────────────────
FROM golang:alpine AS backend-builder
WORKDIR /app/backend

RUN apk add --no-cache git

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/aegix ./cmd/aegix

# ── Stage 3: Minimal Production Alpine Runtime ──────────────────────────────
FROM alpine:latest

# Install TLS certificates (for Groq, Gemini, Gmail, OAuth) and curl for healthcheck
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Copy compiled binary
COPY --from=backend-builder /app/aegix /usr/local/bin/aegix

# Copy built frontend assets (served natively by Go Chi backend)
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

# Copy seed / training data
COPY data/ /app/data/

# Default Environment
ENV API_HOST=0.0.0.0
ENV PORT=8000
ENV API_PORT=8000
ENV API_LOG_LEVEL=info
ENV DB_PATH=/app/data/events.db
ENV FRONTEND_DIST=/app/frontend/dist

# Expose ports (8000 for standard, 10000 for Render default)
EXPOSE 8000 10000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:${PORT:-8000}/healthz || exit 1

CMD ["aegix"]
