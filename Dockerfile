FROM golang:1.24.10 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Собираем основной сервис
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/pull-request-service ./cmd/pull-request-service

FROM debian:12-slim

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/pull-request-service /usr/local/bin/pull-request-service

# кладём openapi.yaml рядом с бинарником
COPY openapi.yaml ./openapi.yaml

EXPOSE 8080

CMD ["pull-request-service"]
