FROM golang:1.24.10 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/pull-request-service ./cmd/pull-request-service

FROM debian:12-slim

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/pull-request-service /usr/local/bin/pull-request-service

EXPOSE 8080

CMD ["pull-request-service"]
