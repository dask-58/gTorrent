FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/gtorrent ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S gtorrent && adduser -S gtorrent -G gtorrent

WORKDIR /app

COPY --from=builder /out/gtorrent .
COPY data/ data/

RUN mkdir -p output && chown -R gtorrent:gtorrent /app
USER gtorrent

ENTRYPOINT ["./gtorrent"]
