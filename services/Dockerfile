# syntax=docker/dockerfile:1
FROM golang:1.26-bookworm AS builder

ENV GOTOOLCHAIN=auto

RUN apt-get update && apt-get install -y --no-install-recommends \
  libolm-dev \
  gcc \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ingestor ./cmd/ingestor
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bot ./cmd/bot

FROM debian:bookworm-slim AS ingestor
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  libolm3 \
  && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/ingestor /usr/local/bin/kg-ingestor
ENTRYPOINT ["/usr/local/bin/kg-ingestor"]

FROM debian:bookworm-slim AS bot
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  libolm3 \
  && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /data && chmod 1777 /data
COPY --from=builder /out/bot /usr/local/bin/matrix-bot
ENTRYPOINT ["/usr/local/bin/matrix-bot"]
