FROM golang:1.19 AS build
COPY . /go/src/reloader
WORKDIR /go/src/reloader
ARG VERSION
RUN VERSION=$VERSION make memphis-config-reloader.docker

FROM alpine:latest as osdeps
RUN apk add --no-cache ca-certificates

FROM alpine:3.17
COPY --from=build /go/src/reloader/memphis-config-reloader.docker /usr/local/bin/memphis-config-reloader
COPY --from=osdeps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["memphis-config-reloader"]
