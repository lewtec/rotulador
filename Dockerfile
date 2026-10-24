# Runtime image for GoReleaser (dockers_v2).
# Binary is built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/rotulador

FROM alpine:3.23@sha256:ff7f0c727dd796aa36787bfcd9a64bf35a6ce72a73451b3359fde849b6703d63

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/rotulador /usr/local/bin/rotulador

ENTRYPOINT ["/usr/local/bin/rotulador"]
