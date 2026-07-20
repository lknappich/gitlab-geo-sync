# syntax=docker/dockerfile:1.7
FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/geoctl /usr/local/bin/geoctl
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/geoctl"]