# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache Go module downloads separately from source code.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -extldflags '-static'" \
    -o /build/flowdo-api ./cmd/api

# ── Final stage ───────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /build/flowdo-api /app/flowdo-api
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

EXPOSE 8080

ENTRYPOINT ["/app/flowdo-api"]
