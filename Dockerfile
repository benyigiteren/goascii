# syntax=docker/dockerfile:1.6

# ---------- Build stage ----------
FROM golang:1.22-alpine AS build
WORKDIR /src

# Cache module downloads
COPY go.mod ./
RUN go mod download

COPY . .

# Build a fully static binary so it can run on a minimal image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/go-ascii .

# ---------- Runtime stage ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Static assets live next to the binary so embed still works.
COPY --from=build /out/go-ascii /app/go-ascii
COPY --from=build /src/web /app/web

# Persistent data (db.json + uploaded custom animations) lives in a volume.
RUN mkdir -p /app/data/animations
VOLUME ["/app/data"]

# Dokploy / compose commonly exposes the app on 8080.
# If you want to expose on 80 / 443 directly, set the env vars below.
ENV PORT=8080 \
    HTTP_PORT_2="" \
    HTTPS_PORT="" \
    TLS_CERT_FILE="" \
    TLS_KEY_FILE=""

EXPOSE 8080

ENTRYPOINT ["/app/go-ascii"]
