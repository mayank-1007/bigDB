# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod ./
COPY . .

RUN go build -o /out/bigdb ./cmd/db

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/bigdb /usr/local/bin/bigdb

VOLUME ["/data"]
ENV bigdb_DATA_DIR=/data

ENTRYPOINT ["bigdb"]
CMD ["-data-dir", "/data"]
