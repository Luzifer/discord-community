FROM golang:alpine as builder

COPY . /go/src/github.com/Luzifer/tezrian-discord
WORKDIR /go/src/github.com/Luzifer/tezrian-discord

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly

FROM alpine:latest

LABEL maintainer "Knut Ahlers <knut@ahlers.me>"

RUN set -ex \
 && apk --no-cache add \
      ca-certificates

COPY --from=builder /go/bin/tezrian-discord /usr/local/bin/tezrian-discord

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/tezrian-discord"]
CMD ["--"]

# vim: set ft=Dockerfile:
