FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS build-env

WORKDIR /go/src/rotulador

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

ARG VERSION_LONG
ENV VERSION_LONG=$VERSION_LONG

ARG VERSION_GIT
ENV VERSION_GIT=$VERSION_GIT

RUN go build -v -o rotulador ./cmd/rotulador

FROM alpine:3.23@sha256:865b95f46d98cf867a156fe4a135ad3fe50d2056aa3f25ed31662dff6da4eb62

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

COPY --from=build-env /go/src/rotulador/rotulador /usr/local/bin

ENTRYPOINT [ "/usr/local/bin/rotulador" ]

