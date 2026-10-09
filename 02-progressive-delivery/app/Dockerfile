# syntax=docker/dockerfile:1.7
# Build stage: dependencies are declared in go.mod/go.sum and vendored into the
# image at build time, never assumed to exist on the host (12-factor II).
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/ ./cmd/...

# Run stage: one static binary, no shell, no package manager, non-root.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/notes /notes
# Numeric UID (distroless "nonroot"), so Kubernetes can verify runAsNonRoot.
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/notes"]
CMD ["serve"]
