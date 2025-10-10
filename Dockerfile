FROM golang:1.25-alpine@sha256:06cdd34bd531b810650e47762c01e025eb9b1c7eadd191553b91c9f2d549fae8 as builder

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
