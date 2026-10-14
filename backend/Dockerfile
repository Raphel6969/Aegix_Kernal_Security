FROM golang:alpine AS build

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o aegix ./cmd/aegix

FROM alpine:latest

# Install CA certificates for HTTPS/TLS (Groq, Gemini, Gmail, OAuth) and curl for healthcheck
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app
COPY --from=build /app/aegix /usr/local/bin/aegix

ENV API_HOST=0.0.0.0
ENV API_PORT=8000
EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:${API_PORT:-8000}/healthz || exit 1

CMD ["aegix"]
