# Runtime image for GoReleaser (dockers_v2).
# Binary is built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/rotulador

FROM alpine:3.23@sha256:f3d55900449227ae9e237ede521e7b665d53a253b42065ba90b9db9275e198ad

RUN apk add --no-cache ca-certificates iptables iproute2 ip6tables

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/rotulador /usr/local/bin/rotulador

ENTRYPOINT ["/usr/local/bin/rotulador"]
