# Build static binaries (no cgo) for a small runtime image.
FROM golang:1.21-bookworm AS build
WORKDIR /src
COPY go.mod go.sum /src/
COPY . /src
ARG VERSION=0.1.0
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/stefiix92/Trailhead/internal/version.Version=${VERSION}" \
    -o /out/trailhead-mcp ./cmd/trailhead-mcp && \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/stefiix92/Trailhead/internal/version.Version=${VERSION}" \
    -o /out/trailhead ./cmd/trailhead

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/trailhead-mcp /trailhead-mcp
COPY --from=build /out/trailhead /trailhead
USER nonroot:nonroot
ENTRYPOINT ["/trailhead-mcp"]
