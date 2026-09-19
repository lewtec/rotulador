# Runtime image for GoReleaser (dockers_v2).
# Binary is built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/rotulador

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/rotulador /usr/local/bin/rotulador

ENTRYPOINT ["/usr/local/bin/rotulador"]
