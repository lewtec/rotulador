# Runtime image for GoReleaser (dockers_v2).
# Binary is built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/rotulador

FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/rotulador /usr/local/bin/rotulador

ENTRYPOINT ["/usr/local/bin/rotulador"]
