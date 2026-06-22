FROM golang:alpine AS builder

ARG branch

RUN apk update && apk upgrade && apk add git zlib-dev gcc musl-dev

COPY . /go/src/github.com/TicketsBot/autoclosedaemon
WORKDIR /go/src/github.com/TicketsBot/autoclosedaemon

RUN set -Eeux && \
    go mod download && \
    go mod verify

RUN GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -o main cmd/autoclosedaemon/main.go

# Prod
FROM alpine:latest

RUN apk update && apk upgrade

COPY --from=builder /go/src/github.com/TicketsBot/autoclosedaemon/main /srv/daemon/main
RUN chmod +x /srv/daemon/main

RUN adduser container --disabled-password --no-create-home
USER container
WORKDIR /srv/daemon

CMD ["/srv/daemon/main"]