# infra/docker/auth.Dockerfile
#
# Multi-stage build — two stages:
#   Stage 1 (builder): full Go toolchain, compiles the binary
#   Stage 2 (final):   distroless image, only the binary — nothing else
#
# Why multi-stage?
# The Go compiler and all build tools are NOT in the final image.
# Final image is ~10MB vs ~800MB for a full Go image.
# No shell, no package manager = almost zero attack surface.
# This is how Stripe, Cloudflare, and Google deploy Go services.

# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# Install git (needed by some Go modules) and CA certificates (for HTTPS calls)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go.work first — this defines the workspace
COPY go.work go.work.sum ./

# Copy all go.mod files so Docker can cache dependency downloads
# If go.mod files don't change, Docker uses cached layers — faster builds
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

# Download dependencies (cached unless go.mod changes)
RUN cd services/auth && go mod download

# Copy source code
COPY pkg/ ./pkg/
COPY services/auth/ ./services/auth/

# Build the binary
# CGO_ENABLED=0: pure Go binary, no C dependencies
# -ldflags="-w -s": strip debug info — smaller binary
# -o /app/bin/auth: output path
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s" \
    -o /app/bin/auth \
    ./services/auth/cmd/server/...

# ── Stage 2: Final image ──────────────────────────────────────────────────────
# distroless/static: Google's minimal base image
# Contains: CA certificates, timezone data, /etc/passwd — nothing else
# No shell means attackers can't exec into your container
FROM gcr.io/distroless/static-debian12

# Copy CA certs and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/bin/auth /auth

# Run as non-root user (distroless provides user 65532 = "nonroot")
# Never run production containers as root
USER nonroot:nonroot

# Document which port this service listens on
EXPOSE 8080

# The binary IS the entrypoint — no shell wrapper
ENTRYPOINT ["/auth"]
