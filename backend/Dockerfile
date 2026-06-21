# ============================================
# Stage 1: Build Go binary
# ============================================
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.version=1.0.0" \
    -o /app/censer-sim .

# ============================================
# Stage 2: Runtime image (scratch for minimal size)
# ============================================
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/censer-sim /app/censer-sim
COPY config /app/config

ENV GIN_MODE=release
ENV PORT=8080
ENV PPROF_ENABLED=true
ENV PROMETHEUS_ENABLED=true

EXPOSE 8080
EXPOSE 6060

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/censer-sim"]
