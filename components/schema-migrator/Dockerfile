FROM golang:1.18.3-alpine3.15 AS build

WORKDIR /migrate

RUN apk --no-cache add --update git make
RUN git clone https://github.com/golang-migrate/migrate.git .

RUN go mod tidy && make build

FROM alpine:3.15.4

LABEL source=git@github.com:kyma-project/control-plane.git

ARG MIGRATE_VER=4.15.1

WORKDIR /migrate

SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

RUN apk --no-cache add --update openssl openssl-dev bash --repository=https://dl-cdn.alpinelinux.org/alpine/edge/main
RUN apk --no-cache add postgresql-client
RUN apk --no-cache add --update curl --repository=https://dl-cdn.alpinelinux.org/alpine/edge/main
COPY --from=build /migrate/migrate /usr/local/bin/migrate

COPY ./migrations/ ./migrations
COPY ./run.sh ./run.sh

ENTRYPOINT ["./run.sh"]
