FROM golang:1.25-alpine@sha256:aee43c3ccbf24fdffb7295693b6e33b21e01baec1b2a55acc351fde345e9ec34 as builder

COPY . /src/discord-community
WORKDIR /src/discord-community

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly


FROM alpine:3.22@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412

ENV TZ=Europe/Berlin

LABEL maintainer "Knut Ahlers <knut@ahlers.me>"

RUN set -ex \
 && apk --no-cache add \
      ca-certificates \
      tzdata

COPY --from=builder /go/bin/discord-community /usr/local/bin/discord-community

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/discord-community"]
CMD ["--"]

# vim: set ft=Dockerfile:
