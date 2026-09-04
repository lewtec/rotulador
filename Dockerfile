# Runtime image for GoReleaser (dockers_v2).
# Binary is built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/rotulador

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/rotulador /usr/local/bin/rotulador

ENTRYPOINT ["/usr/local/bin/rotulador"]
