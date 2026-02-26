# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26.0-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -buildvcs=false -ldflags="-w -s -buildid=" -o /out/koditon ./cmd/main.go

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -buildvcs=false -ldflags="-w -s -buildid=" -o /out/healthcheck ./cmd/healthcheck

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

ARG APP_PORT=8080
ENV APP_PORT=${APP_PORT}

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /out/koditon /app/koditon
COPY --from=builder /out/healthcheck /app/healthcheck

USER nonroot:nonroot

EXPOSE ${APP_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
    CMD ["/app/healthcheck"]

ENTRYPOINT ["/app/koditon"]
