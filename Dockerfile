# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -s -w" \
      -o /out/goldilocks \
      .

FROM alpine:3.23.4

LABEL org.opencontainers.image.authors="FairwindsOps, Inc." \
      org.opencontainers.image.vendor="FairwindsOps, Inc." \
      org.opencontainers.image.title="goldilocks" \
      org.opencontainers.image.description="Goldilocks is a utility that can help you identify a starting point for resource requests and limits." \
      org.opencontainers.image.documentation="https://goldilocks.docs.fairwinds.com/" \
      org.opencontainers.image.source="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.url="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.licenses="Apache License 2.0"

# Install CA bundle for TLS.
RUN apk --no-cache add ca-certificates

# 'nobody' user in alpine
USER 65534

COPY --from=builder /out/goldilocks /goldilocks

CMD ["/goldilocks"]
