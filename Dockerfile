FROM golang:1.26.4-alpine@sha256:7a3e50096189ad57c9f9f865e7e4aa8585ed1585248513dc5cda498e2f41812c as builder

COPY . /src/discord-community
WORKDIR /src/discord-community

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly


FROM alpine:3.24@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4

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
