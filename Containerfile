# syntax=docker/dockerfile:1
# Containerfile — works with both podman and docker.
#
# Stages:
#   builder  — compiles the binary
#   test     — runs all unit tests (used by CI; never shipped)
#   final    — minimal distroless image containing only the binary

# ─── builder ────────────────────────────────────────────────────────────────
# scan-fix(trivy:CVE-2026-39836,CVE-2026-42499,CVE-2026-42504): bump the
# builder base image — these are Go stdlib (net/mail, mime) CVEs fixed in
# Go 1.25.11+, and the compiled binary embeds whichever stdlib version built
# it regardless of go.mod's `toolchain` directive actually taking effect in
# this build environment. 1.25.12-alpine matches go.mod's toolchain pin.
FROM golang:1.25.12-alpine AS builder

WORKDIR /src

# Download dependencies before copying source (improves layer caching).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-w -s" \
      -o /bin/platform-guardian \
      ./cmd/platform-guardian

# ─── test ────────────────────────────────────────────────────────────────────
# This stage is only used in CI to run tests inside the same build environment.
# It is never pushed or run in production.
FROM builder AS test

# -race requires CGO; the builder stage uses CGO_ENABLED=0 for static builds.
# Run the unit test suite here without the race detector.
RUN go test ./... -count=1

# ─── final ───────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS final

COPY --from=builder /bin/platform-guardian /bin/platform-guardian

ENTRYPOINT ["/bin/platform-guardian"]
